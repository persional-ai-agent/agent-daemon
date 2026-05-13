package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/api"
	"github.com/dingjingmaster/agent-daemon/internal/cli"
	"github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/cronrunner"
	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	"github.com/dingjingmaster/agent-daemon/internal/gateway/platforms"
	"github.com/dingjingmaster/agent-daemon/internal/memory"
	"github.com/dingjingmaster/agent-daemon/internal/model"
	"github.com/dingjingmaster/agent-daemon/internal/plugins"
	"github.com/dingjingmaster/agent-daemon/internal/store"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

var (
	appVersion  = "dev"
	releaseDate = ""
	buildCommit = ""
)

type hookSpoolEntry struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func signHook(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func replaySpoolOnce(spoolPath, hookURL, secret string, timeoutSeconds, limit int, typeFilter, idFilter string) (sent int, remaining int, err error) {
	bs, err := os.ReadFile(spoolPath)
	if err != nil {
		return 0, 0, err
	}
	lines := strings.Split(string(bs), "\n")
	if len(lines) == 0 {
		return 0, 0, nil
	}
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	keep := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		var e hookSpoolEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil || strings.TrimSpace(e.Body) == "" {
			continue
		}
		if strings.TrimSpace(typeFilter) != "" && strings.TrimSpace(e.Type) != strings.TrimSpace(typeFilter) {
			keep = append(keep, ln)
			continue
		}
		if strings.TrimSpace(idFilter) != "" && strings.TrimSpace(e.ID) != strings.TrimSpace(idFilter) {
			keep = append(keep, ln)
			continue
		}
		if strings.TrimSpace(e.ID) != "" {
			if seen[e.ID] {
				continue
			}
			seen[e.ID] = true
		}
		if sent >= limit {
			keep = append(keep, ln)
			continue
		}
		bodyBytes := []byte(e.Body)
		reqCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		ts := fmt.Sprintf("%d", time.Now().Unix())
		req, rerr := http.NewRequestWithContext(reqCtx, http.MethodPost, hookURL, bytes.NewReader(bodyBytes))
		if rerr != nil {
			cancel()
			keep = append(keep, ln)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Agent-Event", strings.TrimSpace(e.Type))
		req.Header.Set("X-Agent-Timestamp", ts)
		if strings.TrimSpace(e.ID) != "" {
			req.Header.Set("X-Agent-Event-Id", strings.TrimSpace(e.ID))
		}
		if strings.TrimSpace(secret) != "" {
			req.Header.Set("X-Agent-Signature", signHook(secret, ts, bodyBytes))
		}
		resp, derr := client.Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		cancel()
		if derr == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			sent++
			continue
		}
		keep = append(keep, ln)
	}
	out := strings.Join(keep, "\n")
	if len(keep) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if werr := os.WriteFile(spoolPath, []byte(out), 0o644); werr != nil {
		return sent, len(keep), werr
	}
	return sent, len(keep), nil
}

func listSpoolFiles(basePath string) []string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return nil
	}
	dir := filepath.Dir(basePath)
	base := filepath.Base(basePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		// If directory doesn't exist, still include base path.
		return []string{basePath}
	}
	paths := make([]string, 0, 8)
	// Include rotated files first (oldest first), then base file last.
	prefix := base + "."
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == base {
			continue
		}
		if strings.HasPrefix(name, prefix) {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	sort.Strings(paths)
	paths = append(paths, basePath)
	return paths
}

func spoolStats(paths []string) map[string]any {
	type fileStat struct {
		Path      string         `json:"path"`
		Exists    bool           `json:"exists"`
		SizeBytes int64          `json:"size_bytes"`
		Count     int            `json:"count"`
		ModTime   string         `json:"mod_time,omitempty"`
		Types     map[string]int `json:"types,omitempty"`
		OldestAt  string         `json:"oldest_at,omitempty"`
		Error     string         `json:"error,omitempty"`
	}
	files := make([]fileStat, 0, len(paths))
	totalCount := 0
	totalSize := int64(0)
	totalTypes := map[string]int{}
	var oldest time.Time
	for _, path := range paths {
		st := fileStat{Path: path, Types: map[string]int{}}
		info, err := os.Stat(path)
		if err != nil {
			st.Exists = false
			st.Error = err.Error()
			files = append(files, st)
			continue
		}
		st.Exists = true
		st.SizeBytes = info.Size()
		st.ModTime = info.ModTime().Format(time.RFC3339Nano)
		totalSize += info.Size()
		bs, err := os.ReadFile(path)
		if err != nil {
			st.Error = err.Error()
			files = append(files, st)
			continue
		}
		for _, ln := range strings.Split(string(bs), "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			st.Count++
			totalCount++
			var e hookSpoolEntry
			if err := json.Unmarshal([]byte(ln), &e); err == nil {
				tp := strings.TrimSpace(e.Type)
				if tp == "" {
					tp = "(unknown)"
				}
				st.Types[tp]++
				totalTypes[tp]++
				if strings.TrimSpace(e.CreatedAt) != "" {
					if t, err := time.Parse(time.RFC3339Nano, e.CreatedAt); err == nil {
						if oldest.IsZero() || t.Before(oldest) {
							oldest = t
						}
						if st.OldestAt == "" {
							st.OldestAt = t.Format(time.RFC3339Nano)
						} else if cur, err := time.Parse(time.RFC3339Nano, st.OldestAt); err == nil && t.Before(cur) {
							st.OldestAt = t.Format(time.RFC3339Nano)
						}
					}
				}
			}
		}
		files = append(files, st)
	}
	out := map[string]any{
		"files":            files,
		"file_count":       len(files),
		"total_count":      totalCount,
		"total_size_bytes": totalSize,
		"types":            totalTypes,
	}
	if !oldest.IsZero() {
		out["oldest_at"] = oldest.Format(time.RFC3339Nano)
		out["oldest_age_seconds"] = int(time.Since(oldest).Seconds())
	}
	return out
}

func parseCutoff(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid before timestamp: %q (expected RFC3339/RFC3339Nano)", s)
}

func matchesSpoolFilter(e hookSpoolEntry, typeFilter, idFilter string, cutoff time.Time) bool {
	if strings.TrimSpace(typeFilter) != "" && strings.TrimSpace(e.Type) != strings.TrimSpace(typeFilter) {
		return false
	}
	if strings.TrimSpace(idFilter) != "" && strings.TrimSpace(e.ID) != strings.TrimSpace(idFilter) {
		return false
	}
	if !cutoff.IsZero() {
		if strings.TrimSpace(e.CreatedAt) == "" {
			return false
		}
		t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(e.CreatedAt))
		if err != nil {
			if t2, err2 := time.Parse(time.RFC3339, strings.TrimSpace(e.CreatedAt)); err2 == nil {
				t = t2
			} else {
				return false
			}
		}
		if !t.Before(cutoff) {
			return false
		}
	}
	return true
}

func collectSpoolLines(paths []string, typeFilter, idFilter string, cutoff time.Time) (lines []string, matched int, err error) {
	lines = make([]string, 0, 128)
	for _, path := range paths {
		bs, rerr := os.ReadFile(path)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				continue
			}
			return nil, matched, rerr
		}
		for _, ln := range strings.Split(string(bs), "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			var e hookSpoolEntry
			if jerr := json.Unmarshal([]byte(ln), &e); jerr != nil {
				continue
			}
			if matchesSpoolFilter(e, typeFilter, idFilter, cutoff) {
				lines = append(lines, ln)
				matched++
			}
		}
	}
	return lines, matched, nil
}

func pruneSpoolFile(path, typeFilter, idFilter string, cutoff time.Time) (removed int, remaining int, err error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	keep := make([]string, 0, 64)
	for _, ln := range strings.Split(string(bs), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var e hookSpoolEntry
		if jerr := json.Unmarshal([]byte(ln), &e); jerr != nil {
			// Keep malformed lines to avoid accidental data loss.
			keep = append(keep, ln)
			continue
		}
		if matchesSpoolFilter(e, typeFilter, idFilter, cutoff) {
			removed++
			continue
		}
		keep = append(keep, ln)
	}
	out := strings.Join(keep, "\n")
	if len(keep) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if werr := os.WriteFile(path, []byte(out), 0o644); werr != nil {
		return removed, len(keep), werr
	}
	return removed, len(keep), nil
}

func compactSpoolFile(path string, maxLines int) (before int, after int, err error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	lines := strings.Split(string(bs), "\n")
	type item struct {
		line string
		ent  hookSpoolEntry
		ts   time.Time
	}
	items := make([]item, 0, len(lines))
	seen := map[string]bool{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		before++
		var e hookSpoolEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			// Drop malformed lines during compact.
			continue
		}
		id := strings.TrimSpace(e.ID)
		if id != "" {
			if seen[id] {
				continue
			}
			seen[id] = true
		}
		t := time.Time{}
		if strings.TrimSpace(e.CreatedAt) != "" {
			if tt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(e.CreatedAt)); err == nil {
				t = tt
			} else if tt, err := time.Parse(time.RFC3339, strings.TrimSpace(e.CreatedAt)); err == nil {
				t = tt
			}
		}
		items = append(items, item{line: ln, ent: e, ts: t})
	}
	// Oldest first, so truncating to newest is easy.
	sort.Slice(items, func(i, j int) bool {
		ti, tj := items[i].ts, items[j].ts
		if ti.IsZero() && tj.IsZero() {
			return items[i].line < items[j].line
		}
		if ti.IsZero() {
			return true
		}
		if tj.IsZero() {
			return false
		}
		return ti.Before(tj)
	})
	if maxLines > 0 && len(items) > maxLines {
		items = items[len(items)-maxLines:]
	}
	outLines := make([]string, 0, len(items))
	for _, it := range items {
		outLines = append(outLines, it.line)
	}
	out := strings.Join(outLines, "\n")
	if len(outLines) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return before, 0, err
	}
	after = len(outLines)
	return before, after, nil
}

func importSpoolFile(inputPath, targetPath string, appendMode bool, typeFilter, idFilter string, cutoff time.Time) (imported int, skipped int, total int, err error) {
	inBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return 0, 0, 0, err
	}
	targetLines := make([]string, 0, 128)
	seen := map[string]bool{}
	if appendMode {
		if cur, rerr := os.ReadFile(targetPath); rerr == nil && len(cur) > 0 {
			for _, ln := range strings.Split(string(cur), "\n") {
				ln = strings.TrimSpace(ln)
				if ln == "" {
					continue
				}
				targetLines = append(targetLines, ln)
				var e hookSpoolEntry
				if jerr := json.Unmarshal([]byte(ln), &e); jerr == nil && strings.TrimSpace(e.ID) != "" {
					seen[strings.TrimSpace(e.ID)] = true
				}
			}
		}
	}
	for _, ln := range strings.Split(string(inBytes), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		total++
		var e hookSpoolEntry
		if jerr := json.Unmarshal([]byte(ln), &e); jerr != nil || strings.TrimSpace(e.Type) == "" || strings.TrimSpace(e.Body) == "" {
			skipped++
			continue
		}
		if !matchesSpoolFilter(e, typeFilter, idFilter, cutoff) {
			skipped++
			continue
		}
		id := strings.TrimSpace(e.ID)
		if id != "" && seen[id] {
			skipped++
			continue
		}
		if id != "" {
			seen[id] = true
		}
		targetLines = append(targetLines, ln)
		imported++
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return imported, skipped, total, err
	}
	out := strings.Join(targetLines, "\n")
	if len(targetLines) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if err := os.WriteFile(targetPath, []byte(out), 0o644); err != nil {
		return imported, skipped, total, err
	}
	return imported, skipped, total, nil
}

func listImportFiles(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("input path is empty")
	}
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		// If a file is given with -all, import sibling .jsonl files in same directory.
		dir := filepath.Dir(input)
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		base := filepath.Base(input)
		prefix := base
		if strings.Contains(base, ".") {
			prefix = strings.SplitN(base, ".", 2)[0]
		}
		out := make([]string, 0, 16)
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".jsonl") {
				continue
			}
			if strings.HasPrefix(name, prefix) || strings.Contains(name, "gateway_hooks_spool") {
				out = append(out, filepath.Join(dir, name))
			}
		}
		sort.Strings(out)
		if len(out) == 0 {
			return []string{input}, nil
		}
		return out, nil
	}
	files, err := os.ReadDir(input)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, 32)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".jsonl") {
			continue
		}
		if strings.Contains(name, "gateway_hooks_spool") || strings.Contains(name, "spool") || strings.Contains(name, "hook") {
			out = append(out, filepath.Join(input, name))
		}
	}
	if len(out) == 0 {
		// fallback: all jsonl files
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if strings.HasSuffix(strings.ToLower(name), ".jsonl") {
				out = append(out, filepath.Join(input, name))
			}
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no importable .jsonl files found under %s", input)
	}
	return out, nil
}

func verifySpoolFile(path string) (lines int, valid int, invalid int, invalidSamples []string, err error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, 0, nil, nil
		}
		return 0, 0, 0, nil, err
	}
	invalidSamples = make([]string, 0, 5)
	for _, raw := range strings.Split(string(bs), "\n") {
		ln := strings.TrimSpace(raw)
		if ln == "" {
			continue
		}
		lines++
		var e hookSpoolEntry
		if jerr := json.Unmarshal([]byte(ln), &e); jerr != nil {
			invalid++
			if len(invalidSamples) < 5 {
				invalidSamples = append(invalidSamples, "json:"+truncateForSample(ln, 120))
			}
			continue
		}
		if strings.TrimSpace(e.Type) == "" || strings.TrimSpace(e.Body) == "" {
			invalid++
			if len(invalidSamples) < 5 {
				invalidSamples = append(invalidSamples, "shape:"+truncateForSample(ln, 120))
			}
			continue
		}
		valid++
	}
	return lines, valid, invalid, invalidSamples, nil
}

func truncateForSample(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func parseIntEnvWithDefault(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func main() {
	cfg := config.Load()
	if len(os.Args) < 2 {
		runChat(cfg, "")
		return
	}
	switch os.Args[1] {
	case "chat":
		fs := flag.NewFlagSet("chat", flag.ExitOnError)
		message := fs.String("message", "", "first message to send")
		sessionID := fs.String("session", uuid.NewString(), "session id")
		skills := fs.String("skills", "", "comma-separated skill names to preload")
		_ = fs.Parse(os.Args[2:])
		runChat(cfg, *message, *sessionID, *skills)
	case "tui":
		fs := flag.NewFlagSet("tui", flag.ExitOnError)
		message := fs.String("message", "", "first message to send")
		sessionID := fs.String("session", uuid.NewString(), "session id")
		skills := fs.String("skills", "", "comma-separated skill names to preload")
		_ = fs.Parse(os.Args[2:])
		runTUI(cfg, *message, *sessionID, *skills)
	case "serve":
		runServe(cfg)
	case "tools":
		runTools(cfg, os.Args[2:])
	case "toolsets":
		runToolsets(os.Args[2:])
	case "config":
		runConfig(os.Args[2:])
	case "model":
		runModel(cfg, os.Args[2:])
	case "doctor":
		runDoctor(cfg, os.Args[2:])
	case "setup":
		runSetup(cfg, os.Args[2:])
	case "bootstrap":
		runBootstrap(cfg, os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	case "version":
		runVersion(os.Args[2:])
	case "gateway":
		runGateway(cfg, os.Args[2:])
	case "sessions":
		runSessions(cfg, os.Args[2:])
	case "plugins":
		runPlugins(cfg, os.Args[2:])
	default:
		runChat(cfg, "", uuid.NewString())
	}
}

func runTUI(cfg config.Config, first string, sessionID ...string) {
	eng, cronStore := mustBuildEngine(cfg)
	id := uuid.NewString()
	skills := ""
	if len(sessionID) > 0 && sessionID[0] != "" {
		id = sessionID[0]
	}
	if len(sessionID) > 1 && sessionID[1] != "" {
		skills = sessionID[1]
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.CronEnabled && cronStore != nil {
		s := &cronrunner.Scheduler{
			Store:          cronStore,
			Engine:         eng,
			Tick:           time.Duration(cfg.CronTickSeconds) * time.Second,
			MaxConcurrency: cfg.CronMaxConcurrency,
		}
		if err := s.Start(ctx); err != nil {
			log.Printf("cron scheduler start failed: %v", err)
		}
	}
	if err := cli.RunTUI(ctx, eng, id, first, skills); err != nil {
		log.Fatal(err)
	}
}

func runConfig(args []string) {
	if len(args) == 0 {
		printConfigUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("config list", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		showSecrets := fs.Bool("show-secrets", false, "show secret values")
		_ = fs.Parse(args[1:])
		entries, err := config.ListConfigValues(*path)
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			value := entry.Value
			if !*showSecrets {
				value = config.RedactConfigValue(entry.Key, value)
			}
			fmt.Printf("%s=%s\n", entry.Key, value)
		}
	case "get":
		fs := flag.NewFlagSet("config get", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd config get [-file path] section.key")
		}
		value, ok, err := config.ReadConfigValue(*path, fs.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			os.Exit(1)
		}
		fmt.Println(value)
	case "set":
		fs := flag.NewFlagSet("config set", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 2 {
			log.Fatal("usage: agentd config set [-file path] section.key value")
		}
		if err := config.SaveConfigValue(*path, fs.Arg(0), fs.Arg(1)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("updated %s in %s\n", fs.Arg(0), config.ConfigFilePath(*path))
	default:
		printConfigUsage()
		os.Exit(2)
	}
}

func runSetup(cfg config.Config, args []string) {
	if len(args) > 0 && strings.TrimSpace(args[0]) == "wizard" {
		runSetupWizard(cfg, args[1:])
		return
	}
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	provider := fs.String("provider", "", "model provider (openai/anthropic/codex or provider plugin name)")
	modelName := fs.String("model", "", "model name")
	baseURL := fs.String("base-url", "", "provider base URL")
	apiKey := fs.String("api-key", "", "provider API key")
	fallback := fs.String("fallback-provider", "", "fallback provider")
	gatewayPlatform := fs.String("gateway-platform", "", "optional gateway platform (telegram/discord/slack/yuanbao)")
	gatewayToken := fs.String("gateway-token", "", "shared gateway token (telegram/discord/yuanbao)")
	gatewayBotToken := fs.String("gateway-bot-token", "", "slack bot token")
	gatewayAppToken := fs.String("gateway-app-token", "", "slack app token")
	gatewayAppID := fs.String("gateway-app-id", "", "yuanbao app id")
	gatewayAppSecret := fs.String("gateway-app-secret", "", "yuanbao app secret")
	gatewayAllowedUsers := fs.String("gateway-allowed-users", "", "comma-separated allowed users")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd setup [-file path] -provider <openai|anthropic|codex|plugin> -model <name> [-base-url url] [-api-key key] [-fallback-provider name] [-gateway-platform name] [gateway flags] [-json]")
	}
	selectedProvider := strings.ToLower(strings.TrimSpace(*provider))
	if selectedProvider == "" {
		selectedProvider = strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	}
	if selectedProvider == "" {
		selectedProvider = "openai"
	}
	if !isProviderAvailable(cfg, selectedProvider) {
		log.Fatalf("unsupported provider: %s", selectedProvider)
	}
	selectedModel := strings.TrimSpace(*modelName)
	if selectedModel == "" {
		selectedModel, _ = currentModelConfig(cfg, selectedProvider)
	}
	if strings.TrimSpace(selectedModel) == "" {
		log.Fatal("model is required")
	}
	targetPath := config.ConfigFilePath(*path)
	written, selectedGateway, err := applySetupConfig(
		targetPath,
		selectedProvider,
		selectedModel,
		strings.TrimSpace(*baseURL),
		strings.TrimSpace(*apiKey),
		strings.TrimSpace(*fallback),
		strings.ToLower(strings.TrimSpace(*gatewayPlatform)),
		strings.TrimSpace(*gatewayToken),
		strings.TrimSpace(*gatewayBotToken),
		strings.TrimSpace(*gatewayAppToken),
		strings.TrimSpace(*gatewayAppID),
		strings.TrimSpace(*gatewayAppSecret),
		strings.TrimSpace(*gatewayAllowedUsers),
	)
	if err != nil {
		log.Fatal(err)
	}
	if *jsonOutput {
		printJSON(map[string]any{
			"success":          true,
			"path":             targetPath,
			"provider":         selectedProvider,
			"model":            selectedModel,
			"gateway_platform": selectedGateway,
			"written":          written,
		})
		return
	}
	fmt.Printf("configured provider %s:%s in %s\n", selectedProvider, selectedModel, targetPath)
	if selectedGateway != "" {
		fmt.Printf("configured gateway platform %s\n", selectedGateway)
	}
	fmt.Printf("written=%s\n", strings.Join(written, ","))
}

func runUpdate(args []string) {
	mode := "check"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		mode = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch mode {
	case "bundle":
		runUpdateBundle(args)
		return
	case "changelog":
		fs := flag.NewFlagSet("update changelog", flag.ExitOnError)
		fetchTags := fs.Bool("fetch-tags", false, "run git fetch --tags before checking releases")
		limit := fs.Int("limit", 20, "max commits to return")
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update changelog [-fetch-tags] [-limit N] [-repo path] [-json]")
		}
		info, err := gitChangelogInfo(strings.TrimSpace(*repoPath), *fetchTags, *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("repo=%v\n", info["repo"])
		fmt.Printf("from=%v\n", info["from"])
		fmt.Printf("to=%v\n", info["to"])
		fmt.Printf("commit_count=%v\n", info["commit_count"])
		if commits, ok := info["commits"].([]map[string]string); ok {
			for _, commit := range commits {
				fmt.Printf("- %s %s\n", commit["short"], commit["subject"])
			}
		}
	case "doctor":
		fs := flag.NewFlagSet("update doctor", flag.ExitOnError)
		fetch := fs.Bool("fetch", false, "run git fetch before checking upstream")
		fetchTags := fs.Bool("fetch-tags", false, "run git fetch --tags before checking releases")
		limit := fs.Int("limit", 10, "max tags to return")
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		strict := fs.Bool("strict", false, "exit non-zero when doctor status is not ok")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update doctor [-fetch] [-fetch-tags] [-limit N] [-repo path] [-strict] [-json]")
		}
		report, err := updateDoctorReport(strings.TrimSpace(*repoPath), *fetch, *fetchTags, *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(report)
		} else {
			fmt.Printf("status=%v\n", report["status"])
			fmt.Printf("repo=%v\n", report["repo"])
			if nextActions, ok := report["next_actions"].([]string); ok {
				fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
			}
		}
		if *strict && report["status"] != "ok" {
			os.Exit(1)
		}
		return
	case "status":
		fs := flag.NewFlagSet("update status", flag.ExitOnError)
		fetch := fs.Bool("fetch", false, "run git fetch before checking upstream")
		fetchTags := fs.Bool("fetch-tags", false, "run git fetch --tags before checking releases")
		limit := fs.Int("limit", 10, "max tags to return")
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update status [-fetch] [-fetch-tags] [-limit N] [-repo path] [-json]")
		}
		info, err := updateStatusSummary(strings.TrimSpace(*repoPath), *fetch, *fetchTags, *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		if update, ok := info["update"].(map[string]any); ok {
			fmt.Printf("repo=%v\n", update["repo"])
			fmt.Printf("branch=%v\n", update["branch"])
			fmt.Printf("commit=%v\n", update["commit"])
			fmt.Printf("upstream=%v\n", update["upstream"])
			fmt.Printf("ahead=%v\n", update["ahead"])
			fmt.Printf("behind=%v\n", update["behind"])
			fmt.Printf("dirty=%v\n", update["dirty"])
		}
		if release, ok := info["release"].(map[string]any); ok {
			fmt.Printf("current_tag=%v\n", release["current_tag"])
			fmt.Printf("latest_tag=%v\n", release["latest_tag"])
			fmt.Printf("tag_count=%v\n", release["tag_count"])
		}
		if install, ok := info["install"].(map[string]any); ok {
			fmt.Printf("installed=%v\n", install["installed"])
			fmt.Printf("install_dir=%v\n", install["install_dir"])
			fmt.Printf("manifest_path=%v\n", install["manifest_path"])
		}
	case "check":
		fs := flag.NewFlagSet("update check", flag.ExitOnError)
		fetch := fs.Bool("fetch", false, "run git fetch before checking")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update [check] [-fetch] [-json]")
		}
		status, err := gitUpdateStatus(*fetch)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(status)
			return
		}
		fmt.Printf("repo=%s\n", status["repo"])
		fmt.Printf("branch=%s\n", status["branch"])
		fmt.Printf("commit=%s\n", status["commit"])
		fmt.Printf("upstream=%s\n", status["upstream"])
		fmt.Printf("ahead=%v\n", status["ahead"])
		fmt.Printf("behind=%v\n", status["behind"])
		fmt.Printf("dirty=%v\n", status["dirty"])
		fmt.Printf("can_fast_forward=%v\n", status["can_fast_forward"])
	case "apply":
		fs := flag.NewFlagSet("update apply", flag.ExitOnError)
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update apply [-json]")
		}
		result, err := gitUpdateApply()
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Printf("repo=%s\n", result["repo"])
		fmt.Printf("branch=%s\n", result["branch"])
		fmt.Printf("before=%s\n", result["before"])
		fmt.Printf("after=%s\n", result["after"])
		fmt.Printf("updated=%v\n", result["updated"])
	case "release":
		fs := flag.NewFlagSet("update release", flag.ExitOnError)
		fetchTags := fs.Bool("fetch-tags", false, "run git fetch --tags before checking releases")
		limit := fs.Int("limit", 10, "max tags to return")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update release [-fetch-tags] [-limit N] [-json]")
		}
		info, err := gitReleaseInfo(*fetchTags, *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("repo=%s\n", info["repo"])
		fmt.Printf("commit=%s\n", info["commit"])
		fmt.Printf("current_tag=%v\n", info["current_tag"])
		fmt.Printf("latest_tag=%v\n", info["latest_tag"])
		fmt.Printf("tag_count=%v\n", info["tag_count"])
		if tags, ok := info["recent_tags"].([]string); ok {
			fmt.Printf("recent_tags=%s\n", strings.Join(tags, ","))
		}
	case "install":
		fs := flag.NewFlagSet("update install", flag.ExitOnError)
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update install [-repo path] [-json]")
		}
		result, err := installUpdateScripts(strings.TrimSpace(*repoPath))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Printf("installed update scripts in %s\n", result.InstallDir)
		fmt.Println("manifest=" + result.ManifestPath)
		fmt.Println("scripts=" + strings.Join(result.Scripts, ","))
	case "uninstall":
		fs := flag.NewFlagSet("update uninstall", flag.ExitOnError)
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update uninstall [-repo path] [-json]")
		}
		result, err := uninstallUpdateScripts(strings.TrimSpace(*repoPath))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Printf("uninstalled update scripts from %s\n", result.InstallDir)
		if len(result.Removed) > 0 {
			fmt.Println("removed=" + strings.Join(result.Removed, ","))
		} else {
			fmt.Println("removed=")
		}
	default:
		log.Fatal("usage: agentd update [bundle|changelog|doctor|status|check|apply|release|install|uninstall]")
	}
}

func runUpdateBundle(args []string) {
	submode := "build"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		submode = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch submode {
	case "build":
		fs := flag.NewFlagSet("update bundle", flag.ExitOnError)
		fetchTags := fs.Bool("fetch-tags", false, "run git fetch --tags before building bundle metadata")
		repoPath := fs.String("repo", "", "git repo path (defaults to current checkout root)")
		outPath := fs.String("out", "", "output tar.gz path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update bundle [build] [-fetch-tags] [-repo path] [-out file] [-json]")
		}
		info, err := buildUpdateBundle(strings.TrimSpace(*repoPath), strings.TrimSpace(*outPath), *fetchTags)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("repo=%s\n", info["repo"])
		fmt.Printf("bundle=%s\n", info["bundle_path"])
		fmt.Printf("manifest=%s\n", info["manifest_path"])
		fmt.Printf("commit=%s\n", info["commit"])
		fmt.Printf("latest_tag=%v\n", info["latest_tag"])
		fmt.Printf("file_count=%v\n", info["file_count"])
	case "inspect":
		fs := flag.NewFlagSet("update bundle inspect", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" {
			log.Fatal("usage: agentd update bundle inspect -file <bundle.tar.gz|manifest.json> [-json]")
		}
		info, err := inspectUpdateBundle(strings.TrimSpace(*path))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%s\n", info["bundle_path"])
		fmt.Printf("manifest=%s\n", info["manifest_path"])
		fmt.Printf("bundle_exists=%v\n", info["bundle_exists"])
		fmt.Printf("manifest_exists=%v\n", info["manifest_exists"])
		fmt.Printf("archive_entries=%v\n", info["archive_entries"])
		fmt.Printf("manifest_matches=%v\n", info["manifest_matches"])
	case "verify":
		fs := flag.NewFlagSet("update bundle verify", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		strict := fs.Bool("strict", false, "exit non-zero when verify status is not ok")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" {
			log.Fatal("usage: agentd update bundle verify -file <bundle.tar.gz|manifest.json> [-strict] [-json]")
		}
		info, err := verifyUpdateBundle(strings.TrimSpace(*path))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
		} else {
			fmt.Printf("status=%v\n", info["status"])
			fmt.Printf("bundle=%v\n", info["bundle_path"])
			if nextActions, ok := info["next_actions"].([]string); ok {
				fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
			}
		}
		if *strict && info["status"] != "ok" {
			os.Exit(1)
		}
	case "unpack":
		fs := flag.NewFlagSet("update bundle unpack", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path")
		dest := fs.String("dest", "", "destination directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle unpack -file <bundle.tar.gz> -dest <dir> [-json]")
		}
		info, err := unpackUpdateBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%s\n", info["bundle_path"])
		fmt.Printf("dest=%s\n", info["dest"])
		fmt.Printf("files=%v\n", info["files"])
	case "apply":
		fs := flag.NewFlagSet("update bundle apply", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		dest := fs.String("dest", "", "destination directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle apply -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]")
		}
		info, err := applyUpdateBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%s\n", info["bundle_path"])
		fmt.Printf("dest=%s\n", info["dest"])
		fmt.Printf("applied_files=%v\n", info["applied_files"])
		fmt.Printf("created_files=%v\n", info["created_files"])
		fmt.Printf("overwritten_files=%v\n", info["overwritten_files"])
		fmt.Printf("backup_bundle=%v\n", info["backup_bundle_path"])
	case "rollback":
		fs := flag.NewFlagSet("update bundle rollback", flag.ExitOnError)
		path := fs.String("file", "", "backup bundle tar.gz path or manifest json path (defaults to latest backup under dest)")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle rollback -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]")
		}
		info, err := rollbackUpdateBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("rollback_bundle=%s\n", info["rollback_bundle_path"])
		fmt.Printf("dest=%s\n", info["dest"])
		fmt.Printf("applied_files=%v\n", info["applied_files"])
		fmt.Printf("created_files=%v\n", info["created_files"])
		fmt.Printf("overwritten_files=%v\n", info["overwritten_files"])
		fmt.Printf("backup_bundle=%v\n", info["backup_bundle_path"])
	case "backups":
		fs := flag.NewFlagSet("update bundle backups", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		limit := fs.Int("limit", 10, "max backups to return")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle backups -dest <dir> [-limit N] [-json]")
		}
		info, err := listBundleBackups(strings.TrimSpace(*dest), *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("backup_dir=%s\n", info["backup_dir"])
		fmt.Printf("count=%v\n", info["count"])
		if items, ok := info["items"].([]map[string]any); ok {
			for _, item := range items {
				fmt.Printf("%s %v files=%v source=%v\n", anyString(item["bundle_path"]), item["generated_at"], item["file_count"], item["source_bundle_path"])
			}
		}
	case "prune":
		fs := flag.NewFlagSet("update bundle prune", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		keep := fs.Int("keep", 5, "number of newest backups to keep")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle prune -dest <dir> [-keep N] [-json]")
		}
		info, err := pruneBundleBackups(strings.TrimSpace(*dest), *keep)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("backup_dir=%s\n", info["backup_dir"])
		fmt.Printf("kept=%v\n", info["kept"])
		fmt.Printf("removed=%v\n", info["removed"])
	case "doctor":
		fs := flag.NewFlagSet("update bundle doctor", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		strict := fs.Bool("strict", false, "exit non-zero when doctor status is not ok")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle doctor -dest <dir> [-strict] [-json]")
		}
		info, err := doctorBundleBackups(strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
		} else {
			fmt.Printf("status=%v\n", info["status"])
			fmt.Printf("backup_dir=%v\n", info["backup_dir"])
			if nextActions, ok := info["next_actions"].([]string); ok {
				fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
			}
		}
		if *strict && anyString(info["status"]) != "ok" {
			os.Exit(1)
		}
	case "status":
		fs := flag.NewFlagSet("update bundle status", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd update bundle status [-file <bundle.tar.gz|manifest.json>] [-dest <dir>] [-json]")
		}
		info, err := bundleStatusSummary(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("status=%v\n", info["status"])
		if bundle, ok := info["bundle"].(map[string]any); ok {
			fmt.Printf("bundle=%v\n", bundle["bundle_path"])
		}
		if backup, ok := info["backups"].(map[string]any); ok {
			fmt.Printf("backup_count=%v\n", backup["count"])
		}
		if nextActions, ok := info["next_actions"].([]string); ok {
			fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
		}
	case "manifest":
		fs := flag.NewFlagSet("update bundle manifest", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" {
			log.Fatal("usage: agentd update bundle manifest -file <bundle.tar.gz|manifest.json> [-dest <dir>] [-json]")
		}
		info, err := bundleManifestSummary(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%v\n", info["bundle_path"])
		fmt.Printf("manifest=%v\n", info["manifest_path"])
		fmt.Printf("status=%v\n", info["status"])
		if dest != nil && strings.TrimSpace(*dest) != "" {
			fmt.Printf("target=%v\n", info["target_dir"])
		}
	case "plan":
		fs := flag.NewFlagSet("update bundle plan", flag.ExitOnError)
		path := fs.String("file", "", "bundle tar.gz path or manifest json path")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*path) == "" || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle plan -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]")
		}
		info, err := planUpdateBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%v\n", info["bundle_path"])
		fmt.Printf("dest=%v\n", info["dest"])
		fmt.Printf("create=%v\n", info["create_count"])
		fmt.Printf("overwrite=%v\n", info["overwrite_count"])
		fmt.Printf("backup=%v\n", info["backup_count"])
	case "rollback-plan":
		fs := flag.NewFlagSet("update bundle rollback-plan", flag.ExitOnError)
		path := fs.String("file", "", "backup bundle tar.gz path or manifest json path (defaults to latest backup under dest)")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle rollback-plan -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]")
		}
		info, err := planRollbackBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("rollback_bundle=%v\n", info["rollback_bundle_path"])
		fmt.Printf("dest=%v\n", info["dest"])
		fmt.Printf("create=%v\n", info["create_count"])
		fmt.Printf("overwrite=%v\n", info["overwrite_count"])
		fmt.Printf("backup=%v\n", info["backup_count"])
	case "snapshot":
		fs := flag.NewFlagSet("update bundle snapshot", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		outPath := fs.String("out", "", "output tar.gz path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshot -dest <dir> [-out file] [-json]")
		}
		info, err := snapshotTargetBundle(strings.TrimSpace(*dest), strings.TrimSpace(*outPath))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("bundle=%v\n", info["bundle_path"])
		fmt.Printf("manifest=%v\n", info["manifest_path"])
		fmt.Printf("file_count=%v\n", info["file_count"])
	case "snapshots":
		fs := flag.NewFlagSet("update bundle snapshots", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		limit := fs.Int("limit", 10, "max snapshots to return")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots -dest <dir> [-limit N] [-json]")
		}
		info, err := listSnapshotBundles(strings.TrimSpace(*dest), *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("snapshot_dir=%s\n", info["snapshot_dir"])
		fmt.Printf("count=%v\n", info["count"])
		if items, ok := info["items"].([]map[string]any); ok {
			for _, item := range items {
				fmt.Printf("%s %v files=%v\n", anyString(item["bundle_path"]), item["generated_at"], item["file_count"])
			}
		}
	case "snapshots-prune":
		fs := flag.NewFlagSet("update bundle snapshots-prune", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		keep := fs.Int("keep", 5, "number of newest manual snapshots to keep")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-prune -dest <dir> [-keep N] [-json]")
		}
		info, err := pruneSnapshotBundles(strings.TrimSpace(*dest), *keep)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("snapshot_dir=%s\n", info["snapshot_dir"])
		fmt.Printf("kept=%v\n", info["kept"])
		fmt.Printf("removed=%v\n", info["removed"])
	case "snapshots-doctor":
		fs := flag.NewFlagSet("update bundle snapshots-doctor", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		strict := fs.Bool("strict", false, "exit non-zero when doctor status is not ok")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-doctor -dest <dir> [-strict] [-json]")
		}
		info, err := doctorSnapshotBundles(strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
		} else {
			fmt.Printf("status=%v\n", info["status"])
			fmt.Printf("snapshot_dir=%v\n", info["snapshot_dir"])
			if nextActions, ok := info["next_actions"].([]string); ok {
				fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
			}
		}
		if *strict && anyString(info["status"]) != "ok" {
			os.Exit(1)
		}
	case "snapshots-status":
		fs := flag.NewFlagSet("update bundle snapshots-status", flag.ExitOnError)
		dest := fs.String("dest", "", "target directory")
		limit := fs.Int("limit", 5, "max snapshots to include")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-status -dest <dir> [-limit N] [-json]")
		}
		info, err := snapshotStatusSummary(strings.TrimSpace(*dest), *limit)
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("status=%v\n", info["status"])
		fmt.Printf("snapshot_ready=%v\n", info["snapshot_ready"])
		fmt.Printf("latest_snapshot_path=%v\n", info["latest_snapshot_path"])
		if nextActions, ok := info["next_actions"].([]string); ok {
			fmt.Printf("next_actions=%s\n", strings.Join(nextActions, " | "))
		}
	case "snapshots-restore":
		fs := flag.NewFlagSet("update bundle snapshots-restore", flag.ExitOnError)
		path := fs.String("file", "", "manual snapshot tar.gz path or manifest json path (defaults to latest snapshot under dest)")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-restore -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]")
		}
		info, err := restoreSnapshotBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("snapshot_bundle=%s\n", info["snapshot_bundle_path"])
		fmt.Printf("dest=%s\n", info["dest"])
		fmt.Printf("applied_files=%v\n", info["applied_files"])
		fmt.Printf("created_files=%v\n", info["created_files"])
		fmt.Printf("overwritten_files=%v\n", info["overwritten_files"])
		fmt.Printf("backup_bundle=%v\n", info["backup_bundle_path"])
	case "snapshots-restore-plan":
		fs := flag.NewFlagSet("update bundle snapshots-restore-plan", flag.ExitOnError)
		path := fs.String("file", "", "manual snapshot tar.gz path or manifest json path (defaults to latest snapshot under dest)")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-restore-plan -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]")
		}
		info, err := planRestoreSnapshotBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("snapshot_bundle=%v\n", info["snapshot_bundle_path"])
		fmt.Printf("dest=%v\n", info["dest"])
		fmt.Printf("create=%v\n", info["create_count"])
		fmt.Printf("overwrite=%v\n", info["overwrite_count"])
		fmt.Printf("backup=%v\n", info["backup_count"])
	case "snapshots-delete":
		fs := flag.NewFlagSet("update bundle snapshots-delete", flag.ExitOnError)
		path := fs.String("file", "", "manual snapshot tar.gz path or manifest json path (defaults to latest snapshot under dest)")
		dest := fs.String("dest", "", "target directory")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 || strings.TrimSpace(*dest) == "" {
			log.Fatal("usage: agentd update bundle snapshots-delete -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]")
		}
		info, err := deleteSnapshotBundle(strings.TrimSpace(*path), strings.TrimSpace(*dest))
		if err != nil {
			log.Fatal(err)
		}
		if *jsonOutput {
			printJSON(info)
			return
		}
		fmt.Printf("deleted=%v\n", info["deleted"])
		fmt.Printf("snapshot_bundle=%v\n", info["snapshot_bundle_path"])
		fmt.Printf("manifest=%v\n", info["manifest_path"])
	default:
		log.Fatal("usage: agentd update bundle [build|inspect|verify|unpack|apply|rollback|backups|prune|doctor|status|manifest|plan|rollback-plan|snapshot|snapshots|snapshots-prune|snapshots-doctor|snapshots-status|snapshots-restore|snapshots-restore-plan|snapshots-delete]")
	}
}

func buildUpdateBundle(repoPath, outPath string, fetchTags bool) (map[string]any, error) {
	repo, err := resolveUpdateRepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	releaseInfo, err := gitReleaseInfoAt(repo, fetchTags, 20)
	if err != nil {
		return nil, err
	}
	commit, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	shortCommit, err := runGit(repo, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}
	label := strings.TrimSpace(anyString(releaseInfo["latest_tag"]))
	if label == "" {
		label = shortCommit
	}
	if outPath == "" {
		outPath = filepath.Join(repo, ".agent-daemon", "release", "agent-daemon-"+sanitizeBundleLabel(label)+".tar.gz")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, err
	}
	filesOut, err := runGit(repo, "ls-files")
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, line := range strings.Split(filesOut, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		files = append(files, path)
	}
	if err := writeRepoBundle(repo, outPath, files); err != nil {
		return nil, err
	}
	manifestPath := strings.TrimSuffix(outPath, ".tar.gz") + ".json"
	manifest := map[string]any{
		"repo":         repo,
		"bundle_path":  outPath,
		"commit":       commit,
		"short_commit": shortCommit,
		"latest_tag":   releaseInfo["latest_tag"],
		"current_tag":  releaseInfo["current_tag"],
		"file_count":   len(files),
		"generated_at": time.Now().Format(time.RFC3339Nano),
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return nil, err
	}
	manifest["manifest_path"] = manifestPath
	return manifest, nil
}

func inspectUpdateBundle(path string) (map[string]any, error) {
	bundlePath := path
	manifestPath := path
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		bundlePath = strings.TrimSuffix(path, ".json")
	} else {
		manifestPath = path + ".json"
	}
	info := map[string]any{
		"bundle_path":      bundlePath,
		"manifest_path":    manifestPath,
		"bundle_exists":    fileExists(bundlePath),
		"manifest_exists":  fileExists(manifestPath),
		"manifest_matches": false,
		"archive_entries":  0,
	}
	if fileExists(bundlePath) {
		entryCount, err := countBundleEntries(bundlePath)
		if err != nil {
			return nil, err
		}
		info["archive_entries"] = entryCount
	}
	if fileExists(manifestPath) {
		var manifest map[string]any
		bs, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bs, &manifest); err != nil {
			return nil, err
		}
		info["manifest"] = manifest
		if strings.TrimSpace(anyString(manifest["bundle_path"])) == strings.TrimSpace(bundlePath) {
			info["manifest_matches"] = true
		}
	}
	return info, nil
}

func verifyUpdateBundle(path string) (map[string]any, error) {
	info, err := inspectUpdateBundle(path)
	if err != nil {
		return nil, err
	}
	status := "ok"
	issues := make([]string, 0, 4)
	nextActions := make([]string, 0, 4)
	if exists, _ := info["bundle_exists"].(bool); !exists {
		status = "warn"
		issues = append(issues, "bundle file missing")
		nextActions = append(nextActions, "确认 `update bundle` 输出路径，或重新执行 `agentd update bundle` 生成 bundle")
	}
	if exists, _ := info["manifest_exists"].(bool); !exists {
		status = "warn"
		issues = append(issues, "bundle manifest missing")
		nextActions = append(nextActions, "保留 bundle 旁的 `.json` manifest，便于后续校验与分发")
	}
	if matches, _ := info["manifest_matches"].(bool); !matches && anyBool(info["manifest_exists"]) {
		status = "warn"
		issues = append(issues, "bundle manifest does not match bundle path")
		nextActions = append(nextActions, "检查 bundle 与 manifest 是否来自同一次打包输出")
	}
	if entries, _ := info["archive_entries"].(int); entries == 0 && anyBool(info["bundle_exists"]) {
		status = "warn"
		issues = append(issues, "bundle archive has no entries")
		nextActions = append(nextActions, "重新执行 `agentd update bundle`，确认打包内容不是空归档")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "bundle 校验通过，可继续用于分发或后续安装流程")
	}
	info["status"] = status
	info["issues"] = issues
	info["next_actions"] = nextActions
	return info, nil
}

func unpackUpdateBundle(path, dest string) (map[string]any, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	files := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") {
			return nil, fmt.Errorf("unsafe bundle entry: %s", header.Name)
		}
		target := filepath.Join(dest, name)
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("bundle entry escapes destination: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return nil, err
			}
			if err := out.Close(); err != nil {
				return nil, err
			}
			files++
		}
	}
	return map[string]any{
		"bundle_path": path,
		"dest":        dest,
		"files":       files,
	}, nil
}

func applyUpdateBundle(path, dest string) (map[string]any, error) {
	info, err := inspectUpdateBundle(path)
	if err != nil {
		return nil, err
	}
	bundlePath := strings.TrimSpace(anyString(info["bundle_path"]))
	if bundlePath == "" || !anyBool(info["bundle_exists"]) {
		return nil, fmt.Errorf("bundle file missing: %s", path)
	}
	if entries, _ := info["archive_entries"].(int); entries == 0 {
		return nil, fmt.Errorf("bundle archive has no entries: %s", bundlePath)
	}
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	result, err := applyBundleArchive(bundlePath, dest, backupDir, path)
	if err != nil {
		return nil, err
	}
	result["bundle_path"] = bundlePath
	return result, nil
}

func planUpdateBundle(path, dest string) (map[string]any, error) {
	info, err := inspectUpdateBundle(path)
	if err != nil {
		return nil, err
	}
	bundlePath := strings.TrimSpace(anyString(info["bundle_path"]))
	if bundlePath == "" || !anyBool(info["bundle_exists"]) {
		return nil, fmt.Errorf("bundle file missing: %s", path)
	}
	relFiles, err := listBundleRegularEntries(bundlePath)
	if err != nil {
		return nil, err
	}
	createItems := make([]string, 0)
	overwriteItems := make([]string, 0)
	for _, rel := range relFiles {
		target := filepath.Join(dest, rel)
		stat, err := os.Stat(target)
		if err == nil && stat.Mode().IsRegular() {
			overwriteItems = append(overwriteItems, rel)
			continue
		}
		if os.IsNotExist(err) {
			createItems = append(createItems, rel)
		}
	}
	nextActions := make([]string, 0, 4)
	if len(overwriteItems) > 0 {
		nextActions = append(nextActions, "将覆盖现有文件，建议先查看 `update bundle backups` 或保留目标目录快照")
	}
	if len(createItems) > 0 {
		nextActions = append(nextActions, "将创建新文件，可继续执行 `agentd update bundle apply` 落地变更")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "目标目录与 bundle 内容已完全重合，apply 主要会重写现有文件")
	}
	return map[string]any{
		"bundle_path":     bundlePath,
		"dest":            dest,
		"archive_entries": len(relFiles),
		"create_count":    len(createItems),
		"overwrite_count": len(overwriteItems),
		"backup_count":    len(overwriteItems),
		"create_items":    createItems,
		"overwrite_items": overwriteItems,
		"next_actions":    nextActions,
	}, nil
}

func planRollbackBundle(path, dest string) (map[string]any, error) {
	bundlePath := strings.TrimSpace(path)
	if bundlePath == "" {
		var err error
		bundlePath, err = latestBundleBackupPath(dest)
		if err != nil {
			return nil, err
		}
	}
	info, err := planUpdateBundle(bundlePath, dest)
	if err != nil {
		return nil, err
	}
	info["rollback_bundle_path"] = info["bundle_path"]
	delete(info, "bundle_path")
	if nextActions, ok := info["next_actions"].([]string); ok {
		nextActions = append([]string{"这是 rollback 预演；确认后可执行 `agentd update bundle rollback` 恢复目标目录"}, nextActions...)
		info["next_actions"] = nextActions
	}
	return info, nil
}

func snapshotTargetBundle(dest, outPath string) (map[string]any, error) {
	dest = filepath.Clean(dest)
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	if outPath == "" {
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return nil, err
		}
		outPath = filepath.Join(backupDir, fmt.Sprintf("%d-manual-snapshot.tar.gz", time.Now().UnixNano()))
	}
	files, err := listSnapshotFiles(dest, outPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, err
	}
	if err := writeRepoBundle(dest, outPath, files); err != nil {
		return nil, err
	}
	manifestPath := strings.TrimSuffix(outPath, ".tar.gz") + ".json"
	manifest := map[string]any{
		"bundle_path":   outPath,
		"target_dir":    dest,
		"file_count":    len(files),
		"generated_at":  time.Now().Format(time.RFC3339Nano),
		"snapshot_type": "manual",
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return nil, err
	}
	manifest["manifest_path"] = manifestPath
	return manifest, nil
}

func listSnapshotBundles(dest string, limit int) (map[string]any, error) {
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	matches, err := filepath.Glob(filepath.Join(backupDir, "*-manual-snapshot.tar.gz"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if limit <= 0 {
		limit = len(matches)
	}
	items := make([]map[string]any, 0, minInt(limit, len(matches)))
	for idx := len(matches) - 1; idx >= 0 && len(items) < limit; idx-- {
		bundlePath := matches[idx]
		manifestPath := strings.TrimSuffix(bundlePath, ".tar.gz") + ".json"
		item := map[string]any{
			"bundle_path":     bundlePath,
			"manifest_path":   manifestPath,
			"manifest_exists": fileExists(manifestPath),
		}
		if stat, err := os.Stat(bundlePath); err == nil {
			item["bundle_size"] = stat.Size()
		}
		if fileExists(manifestPath) {
			bs, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, err
			}
			var manifest map[string]any
			if err := json.Unmarshal(bs, &manifest); err != nil {
				return nil, err
			}
			item["generated_at"] = manifest["generated_at"]
			item["file_count"] = manifest["file_count"]
			item["snapshot_type"] = manifest["snapshot_type"]
		}
		items = append(items, item)
	}
	return map[string]any{
		"snapshot_dir": backupDir,
		"count":        len(items),
		"items":        items,
	}, nil
}

func pruneSnapshotBundles(dest string, keep int) (map[string]any, error) {
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	matches, err := filepath.Glob(filepath.Join(backupDir, "*-manual-snapshot.tar.gz"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if keep < 0 {
		keep = 0
	}
	removed := make([]string, 0)
	if len(matches) > keep {
		for _, bundlePath := range matches[:len(matches)-keep] {
			manifestPath := strings.TrimSuffix(bundlePath, ".tar.gz") + ".json"
			if err := os.Remove(bundlePath); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			removed = append(removed, bundlePath)
		}
	}
	kept := minInt(keep, len(matches))
	return map[string]any{
		"snapshot_dir": backupDir,
		"kept":         kept,
		"removed":      len(removed),
		"items":        removed,
	}, nil
}

func doctorSnapshotBundles(dest string) (map[string]any, error) {
	snapshots, err := listSnapshotBundles(dest, 10)
	if err != nil {
		return nil, err
	}
	status := "ok"
	issues := make([]string, 0, 4)
	nextActions := make([]string, 0, 4)
	count, _ := snapshots["count"].(int)
	if count == 0 {
		status = "warn"
		issues = append(issues, "no manual snapshots found")
		nextActions = append(nextActions, "先执行一次 `agentd update bundle snapshot -dest <dir>` 创建手工 restore point")
	}
	if items, ok := snapshots["items"].([]map[string]any); ok {
		missingManifest := 0
		for _, item := range items {
			if exists, _ := item["manifest_exists"].(bool); !exists {
				missingManifest++
			}
		}
		if missingManifest > 0 {
			status = "warn"
			issues = append(issues, fmt.Sprintf("%d snapshot manifests missing", missingManifest))
			nextActions = append(nextActions, "保留 manual snapshot 旁的 `.json` manifest，便于后续确认 restore point 元数据")
		}
		if count > 10 {
			status = "warn"
			issues = append(issues, "too many manual snapshots retained")
			nextActions = append(nextActions, "运行 `agentd update bundle snapshots-prune -dest <dir> -keep N` 清理过旧手工快照")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "manual snapshot 状态正常，可继续保留为 restore point 或按需执行 snapshots-prune")
	}
	return map[string]any{
		"status":       status,
		"snapshot_dir": snapshots["snapshot_dir"],
		"count":        count,
		"issues":       issues,
		"next_actions": nextActions,
		"items":        snapshots["items"],
	}, nil
}

func latestSnapshotBundlePath(dest string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dest, ".agent-daemon", "release-backups", "*-manual-snapshot.tar.gz"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no manual snapshots found under %s", filepath.Join(dest, ".agent-daemon", "release-backups"))
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func snapshotStatusSummary(dest string, limit int) (map[string]any, error) {
	status := "ok"
	issues := make([]string, 0, 6)
	nextActions := make([]string, 0, 6)
	result := map[string]any{
		"status": status,
	}
	snapshots, err := listSnapshotBundles(dest, limit)
	if err != nil {
		return nil, err
	}
	result["snapshots"] = snapshots
	doctor, err := doctorSnapshotBundles(dest)
	if err != nil {
		return nil, err
	}
	result["doctor"] = doctor
	if anyString(doctor["status"]) != "ok" {
		status = "warn"
		if doctorIssues, ok := doctor["issues"].([]string); ok {
			issues = append(issues, doctorIssues...)
		}
		if doctorActions, ok := doctor["next_actions"].([]string); ok {
			nextActions = append(nextActions, doctorActions...)
		}
	}
	if count, _ := snapshots["count"].(int); count > 0 {
		if latest, err := latestSnapshotBundlePath(dest); err == nil {
			result["latest_snapshot_path"] = latest
			result["snapshot_ready"] = true
		}
	} else {
		result["snapshot_ready"] = false
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "manual snapshot 状态正常，可继续保留 restore point 或按需执行 snapshots-prune")
	}
	result["status"] = status
	result["issues"] = dedupeStrings(issues)
	result["next_actions"] = dedupeStrings(nextActions)
	return result, nil
}

func restoreSnapshotBundle(path, dest string) (map[string]any, error) {
	snapshotPath := strings.TrimSpace(path)
	if snapshotPath == "" {
		var err error
		snapshotPath, err = latestSnapshotBundlePath(dest)
		if err != nil {
			return nil, err
		}
	}
	info, err := inspectUpdateBundle(snapshotPath)
	if err != nil {
		return nil, err
	}
	resolvedPath := strings.TrimSpace(anyString(info["bundle_path"]))
	if resolvedPath == "" || !anyBool(info["bundle_exists"]) {
		return nil, fmt.Errorf("snapshot bundle file missing: %s", snapshotPath)
	}
	if entries, _ := info["archive_entries"].(int); entries == 0 {
		return nil, fmt.Errorf("snapshot bundle archive has no entries: %s", resolvedPath)
	}
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	result, err := applyBundleArchive(resolvedPath, dest, backupDir, snapshotPath)
	if err != nil {
		return nil, err
	}
	result["snapshot_bundle_path"] = resolvedPath
	return result, nil
}

func planRestoreSnapshotBundle(path, dest string) (map[string]any, error) {
	snapshotPath := strings.TrimSpace(path)
	if snapshotPath == "" {
		var err error
		snapshotPath, err = latestSnapshotBundlePath(dest)
		if err != nil {
			return nil, err
		}
	}
	info, err := planUpdateBundle(snapshotPath, dest)
	if err != nil {
		return nil, err
	}
	info["snapshot_bundle_path"] = info["bundle_path"]
	delete(info, "bundle_path")
	if nextActions, ok := info["next_actions"].([]string); ok {
		nextActions = append([]string{"这是 snapshot restore 预演；确认后可执行 `agentd update bundle snapshots-restore` 恢复目标目录"}, nextActions...)
		info["next_actions"] = nextActions
	}
	return info, nil
}

func deleteSnapshotBundle(path, dest string) (map[string]any, error) {
	snapshotPath := strings.TrimSpace(path)
	if snapshotPath == "" {
		var err error
		snapshotPath, err = latestSnapshotBundlePath(dest)
		if err != nil {
			return nil, err
		}
	}
	if strings.HasSuffix(strings.ToLower(snapshotPath), ".json") {
		snapshotPath = strings.TrimSuffix(snapshotPath, ".json")
	}
	manifestPath := strings.TrimSuffix(snapshotPath, ".tar.gz") + ".json"
	if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return map[string]any{
		"deleted":              true,
		"snapshot_bundle_path": snapshotPath,
		"manifest_path":        manifestPath,
	}, nil
}

func rollbackUpdateBundle(path, dest string) (map[string]any, error) {
	bundlePath := strings.TrimSpace(path)
	if bundlePath == "" {
		var err error
		bundlePath, err = latestBundleBackupPath(dest)
		if err != nil {
			return nil, err
		}
	}
	info, err := inspectUpdateBundle(bundlePath)
	if err != nil {
		return nil, err
	}
	resolvedPath := strings.TrimSpace(anyString(info["bundle_path"]))
	if resolvedPath == "" || !anyBool(info["bundle_exists"]) {
		return nil, fmt.Errorf("rollback bundle file missing: %s", bundlePath)
	}
	if entries, _ := info["archive_entries"].(int); entries == 0 {
		return nil, fmt.Errorf("rollback bundle archive has no entries: %s", resolvedPath)
	}
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	result, err := applyBundleArchive(resolvedPath, dest, backupDir, bundlePath)
	if err != nil {
		return nil, err
	}
	result["rollback_bundle_path"] = resolvedPath
	return result, nil
}

func applyBundleArchive(bundlePath, dest, backupDir, sourcePath string) (map[string]any, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	relFiles, err := listBundleRegularEntries(bundlePath)
	if err != nil {
		return nil, err
	}
	backupFiles := make([]string, 0, len(relFiles))
	for _, rel := range relFiles {
		target := filepath.Join(dest, rel)
		info, err := os.Stat(target)
		if err == nil && info.Mode().IsRegular() {
			backupFiles = append(backupFiles, rel)
		}
	}
	backupPath := ""
	backupManifestPath := ""
	if len(backupFiles) > 0 {
		label := bundleBackupLabel(bundlePath)
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return nil, err
		}
		backupPath = filepath.Join(backupDir, fmt.Sprintf("%d-%s.tar.gz", time.Now().UnixNano(), label))
		if err := writeRepoBundle(dest, backupPath, backupFiles); err != nil {
			return nil, err
		}
		backupManifestPath = strings.TrimSuffix(backupPath, ".tar.gz") + ".json"
		backupManifest := map[string]any{
			"source_bundle_path": sourcePath,
			"bundle_path":        backupPath,
			"file_count":         len(backupFiles),
			"generated_at":       time.Now().Format(time.RFC3339Nano),
		}
		manifestBytes, _ := json.MarshalIndent(backupManifest, "", "  ")
		if err := os.WriteFile(backupManifestPath, manifestBytes, 0o644); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(bundlePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	applied := 0
	created := 0
	overwritten := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		target, err := safeBundleTarget(dest, header.Name)
		if err != nil {
			return nil, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			_, statErr := os.Stat(target)
			if statErr == nil {
				overwritten++
			} else if os.IsNotExist(statErr) {
				created++
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return nil, err
			}
			if err := out.Close(); err != nil {
				return nil, err
			}
			applied++
		}
	}
	return map[string]any{
		"dest":                 dest,
		"applied_files":        applied,
		"created_files":        created,
		"overwritten_files":    overwritten,
		"backup_files":         len(backupFiles),
		"backup_bundle_path":   backupPath,
		"backup_manifest_path": backupManifestPath,
	}, nil
}

func latestBundleBackupPath(dest string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dest, ".agent-daemon", "release-backups", "*.tar.gz"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no bundle backups found under %s", filepath.Join(dest, ".agent-daemon", "release-backups"))
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func listBundleBackups(dest string, limit int) (map[string]any, error) {
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	matches, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if limit <= 0 {
		limit = len(matches)
	}
	items := make([]map[string]any, 0, minInt(limit, len(matches)))
	for idx := len(matches) - 1; idx >= 0 && len(items) < limit; idx-- {
		bundlePath := matches[idx]
		manifestPath := strings.TrimSuffix(bundlePath, ".tar.gz") + ".json"
		item := map[string]any{
			"bundle_path":     bundlePath,
			"manifest_path":   manifestPath,
			"manifest_exists": fileExists(manifestPath),
		}
		if stat, err := os.Stat(bundlePath); err == nil {
			item["bundle_size"] = stat.Size()
		}
		if fileExists(manifestPath) {
			bs, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, err
			}
			var manifest map[string]any
			if err := json.Unmarshal(bs, &manifest); err != nil {
				return nil, err
			}
			item["generated_at"] = manifest["generated_at"]
			item["file_count"] = manifest["file_count"]
			item["source_bundle_path"] = manifest["source_bundle_path"]
		}
		items = append(items, item)
	}
	return map[string]any{
		"backup_dir": backupDir,
		"count":      len(items),
		"items":      items,
	}, nil
}

func pruneBundleBackups(dest string, keep int) (map[string]any, error) {
	backupDir := filepath.Join(dest, ".agent-daemon", "release-backups")
	matches, err := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if keep < 0 {
		keep = 0
	}
	removed := make([]string, 0)
	if len(matches) > keep {
		for _, bundlePath := range matches[:len(matches)-keep] {
			manifestPath := strings.TrimSuffix(bundlePath, ".tar.gz") + ".json"
			if err := os.Remove(bundlePath); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			removed = append(removed, bundlePath)
		}
	}
	kept := minInt(keep, len(matches))
	return map[string]any{
		"backup_dir": backupDir,
		"kept":       kept,
		"removed":    len(removed),
		"items":      removed,
	}, nil
}

func doctorBundleBackups(dest string) (map[string]any, error) {
	backups, err := listBundleBackups(dest, 10)
	if err != nil {
		return nil, err
	}
	status := "ok"
	issues := make([]string, 0, 4)
	nextActions := make([]string, 0, 4)
	count, _ := backups["count"].(int)
	if count == 0 {
		status = "warn"
		issues = append(issues, "no backup bundles found")
		nextActions = append(nextActions, "先执行一次 `agentd update bundle apply` 或 `agentd update bundle rollback`，生成可回滚的 backup bundle")
	}
	if items, ok := backups["items"].([]map[string]any); ok {
		missingManifest := 0
		for _, item := range items {
			if exists, _ := item["manifest_exists"].(bool); !exists {
				missingManifest++
			}
		}
		if missingManifest > 0 {
			status = "warn"
			issues = append(issues, fmt.Sprintf("%d backup manifests missing", missingManifest))
			nextActions = append(nextActions, "保留 backup bundle 旁的 `.json` manifest，便于回滚前确认来源与文件数")
		}
		if count > 10 {
			status = "warn"
			issues = append(issues, "too many backup bundles retained")
			nextActions = append(nextActions, "运行 `agentd update bundle prune -dest <dir> -keep N` 清理过旧回滚点")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "bundle backup 状态正常，可继续执行 rollback 或按需 prune")
	}
	return map[string]any{
		"status":       status,
		"backup_dir":   backups["backup_dir"],
		"count":        count,
		"issues":       issues,
		"next_actions": nextActions,
		"items":        backups["items"],
	}, nil
}

func bundleStatusSummary(path, dest string) (map[string]any, error) {
	status := "ok"
	issues := make([]string, 0, 6)
	nextActions := make([]string, 0, 6)
	result := map[string]any{
		"status": status,
	}
	if path != "" {
		bundle, err := verifyUpdateBundle(path)
		if err != nil {
			return nil, err
		}
		result["bundle"] = bundle
		if anyString(bundle["status"]) != "ok" {
			status = "warn"
			if bundleIssues, ok := bundle["issues"].([]string); ok {
				issues = append(issues, bundleIssues...)
			}
			if bundleActions, ok := bundle["next_actions"].([]string); ok {
				nextActions = append(nextActions, bundleActions...)
			}
		}
	}
	if dest != "" {
		backups, err := listBundleBackups(dest, 5)
		if err != nil {
			return nil, err
		}
		result["backups"] = backups
		doctor, err := doctorBundleBackups(dest)
		if err != nil {
			return nil, err
		}
		result["doctor"] = doctor
		if anyString(doctor["status"]) != "ok" {
			status = "warn"
			if doctorIssues, ok := doctor["issues"].([]string); ok {
				issues = append(issues, doctorIssues...)
			}
			if doctorActions, ok := doctor["next_actions"].([]string); ok {
				nextActions = append(nextActions, doctorActions...)
			}
		}
		if count, _ := backups["count"].(int); count > 0 {
			if latest, err := latestBundleBackupPath(dest); err == nil {
				result["latest_backup_path"] = latest
				result["rollback_ready"] = true
			}
		} else {
			result["rollback_ready"] = false
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "bundle 状态正常，可继续 apply、rollback 或按需 prune")
	}
	result["status"] = status
	result["issues"] = dedupeStrings(issues)
	result["next_actions"] = dedupeStrings(nextActions)
	return result, nil
}

func bundleManifestSummary(path, dest string) (map[string]any, error) {
	info, err := inspectUpdateBundle(path)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"bundle_path":      info["bundle_path"],
		"manifest_path":    info["manifest_path"],
		"bundle_exists":    info["bundle_exists"],
		"manifest_exists":  info["manifest_exists"],
		"manifest_matches": info["manifest_matches"],
		"archive_entries":  info["archive_entries"],
	}
	status := "ok"
	issues := make([]string, 0, 4)
	nextActions := make([]string, 0, 4)
	if manifest, ok := info["manifest"].(map[string]any); ok {
		result["manifest"] = manifest
	}
	if !anyBool(info["bundle_exists"]) {
		status = "warn"
		issues = append(issues, "bundle file missing")
		nextActions = append(nextActions, "确认 bundle 路径是否正确，或重新执行 `agentd update bundle`")
	}
	if !anyBool(info["manifest_exists"]) {
		status = "warn"
		issues = append(issues, "bundle manifest missing")
		nextActions = append(nextActions, "保留 bundle 旁的 `.json` manifest，便于分发时读取元数据")
	}
	if anyBool(info["manifest_exists"]) && !anyBool(info["manifest_matches"]) {
		status = "warn"
		issues = append(issues, "bundle manifest does not match bundle path")
		nextActions = append(nextActions, "检查 manifest 与 bundle 是否来自同一次构建")
	}
	if dest != "" {
		result["target_dir"] = dest
		backups, err := listBundleBackups(dest, 1)
		if err != nil {
			return nil, err
		}
		result["target_backups"] = backups
		if count, _ := backups["count"].(int); count > 0 {
			if latest, err := latestBundleBackupPath(dest); err == nil {
				result["latest_backup_path"] = latest
			}
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "manifest 状态正常，可继续分发、apply 或 rollback")
	}
	result["status"] = status
	result["issues"] = dedupeStrings(issues)
	result["next_actions"] = dedupeStrings(nextActions)
	return result, nil
}

func bundleBackupLabel(bundlePath string) string {
	label := strings.TrimSuffix(filepath.Base(bundlePath), ".tar.gz")
	label = strings.TrimSuffix(label, ".tgz")
	if len(label) > 16 && label[8] == '-' && label[15] == '-' {
		label = label[16:]
	}
	if idx := strings.Index(label, "-"); idx > 0 {
		prefix := label[:idx]
		allDigits := true
		for _, r := range prefix {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			label = label[idx+1:]
		}
	}
	return sanitizeBundleLabel(label)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func listSnapshotFiles(root, outPath string) ([]string, error) {
	files := make([]string, 0, 128)
	skipDir := filepath.Clean(filepath.Join(root, ".agent-daemon", "release-backups"))
	cleanOut := filepath.Clean(outPath)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		cleanPath := filepath.Clean(path)
		if info.IsDir() {
			if cleanPath == skipDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if cleanPath == cleanOut || cleanPath == strings.TrimSuffix(cleanOut, ".tar.gz")+".json" {
			return nil
		}
		rel, err := filepath.Rel(root, cleanPath)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func listBundleRegularEntries(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	files := make([]string, 0, 64)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			target, err := safeBundleTarget(".", header.Name)
			if err != nil {
				return nil, err
			}
			files = append(files, filepath.Clean(target))
		}
	}
}

func safeBundleTarget(dest, name string) (string, error) {
	cleanName := filepath.Clean(name)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return "", fmt.Errorf("unsafe bundle entry: %s", name)
	}
	target := filepath.Join(dest, cleanName)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("bundle entry escapes destination: %s", name)
	}
	return target, nil
}

func anyBool(value any) bool {
	flag, _ := value.(bool)
	return flag
}

func countBundleEntries(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	count := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		count++
	}
}

func writeRepoBundle(repo, outPath string, files []string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	gzw := gzip.NewWriter(outFile)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()
	for _, rel := range files {
		abs := filepath.Join(repo, rel)
		info, err := os.Stat(abs)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		f, err := os.Open(abs)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func sanitizeBundleLabel(label string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", "@", "-")
	label = replacer.Replace(strings.TrimSpace(label))
	if label == "" {
		return "bundle"
	}
	return label
}

func gitChangelogInfo(repoPath string, fetchTags bool, limit int) (map[string]any, error) {
	repo, err := resolveUpdateRepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	releaseInfo, err := gitReleaseInfoAt(repo, fetchTags, limit)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	toCommit, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	fromRef := strings.TrimSpace(anyString(releaseInfo["latest_tag"]))
	logRange := toCommit
	if fromRef != "" {
		logRange = fromRef + "..HEAD"
	}
	logOut, err := runGit(repo, "log", "--max-count", strconv.Itoa(limit), "--pretty=format:%H%x09%h%x09%s", logRange)
	if err != nil {
		return nil, err
	}
	commits := make([]map[string]string, 0)
	if strings.TrimSpace(logOut) != "" {
		for _, line := range strings.Split(logOut, "\n") {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				continue
			}
			commits = append(commits, map[string]string{
				"commit":  strings.TrimSpace(parts[0]),
				"short":   strings.TrimSpace(parts[1]),
				"subject": strings.TrimSpace(parts[2]),
			})
		}
	}
	return map[string]any{
		"repo":         repo,
		"from":         fromRef,
		"to":           toCommit,
		"commit_count": len(commits),
		"commits":      commits,
		"has_tag_base": fromRef != "",
	}, nil
}

func anyString(value any) string {
	text, _ := value.(string)
	return text
}

func updateDoctorReport(repoPath string, fetch bool, fetchTags bool, limit int) (map[string]any, error) {
	summary, err := updateStatusSummary(repoPath, fetch, fetchTags, limit)
	if err != nil {
		return nil, err
	}
	nextActions := make([]string, 0, 6)
	issues := make([]string, 0, 6)
	status := "ok"
	if install, ok := summary["install"].(map[string]any); ok {
		if installed, _ := install["installed"].(bool); !installed {
			status = "warn"
			issues = append(issues, "update scripts not installed")
			nextActions = append(nextActions, "运行 `agentd update install` 生成最小 update 运维脚本")
		}
	}
	if release, ok := summary["release"].(map[string]any); ok {
		if tagCount, _ := release["tag_count"].(int); tagCount == 0 {
			status = "warn"
			issues = append(issues, "no release tags found")
			nextActions = append(nextActions, "当前仓库尚无 tags，可在发布流程接入 tag/release 管理")
		}
	}
	if update, ok := summary["update"].(map[string]any); ok {
		if dirty, _ := update["dirty"].(bool); dirty {
			status = "warn"
			issues = append(issues, "working tree is dirty")
			nextActions = append(nextActions, "提交或清理当前工作区改动后再执行 update apply")
		}
		if behind, _ := update["behind"].(int); behind > 0 {
			status = "warn"
			issues = append(issues, "branch is behind upstream")
			nextActions = append(nextActions, "运行 `agentd update apply` 拉取上游快进更新")
		}
		if ahead, _ := update["ahead"].(int); ahead > 0 {
			status = "warn"
			issues = append(issues, "branch is ahead of upstream")
			nextActions = append(nextActions, "当前分支领先上游，确认是否需要先推送再继续 update 流程")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "update 状态正常，无需额外操作")
	}
	summary["status"] = status
	summary["issues"] = issues
	summary["next_actions"] = nextActions
	return summary, nil
}

func updateStatusSummary(repoPath string, fetch bool, fetchTags bool, limit int) (map[string]any, error) {
	repo, err := resolveUpdateRepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	updateInfo, err := gitUpdateStatusAt(repo, fetch)
	if err != nil {
		return nil, err
	}
	releaseInfo, err := gitReleaseInfoAt(repo, fetchTags, limit)
	if err != nil {
		return nil, err
	}
	installInfo, err := updateInstallStatus(repo)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"repo":    repo,
		"update":  updateInfo,
		"release": releaseInfo,
		"install": installInfo,
	}, nil
}

func gitReleaseInfo(fetchTags bool, limit int) (map[string]any, error) {
	repo, err := gitRepoRoot()
	if err != nil {
		return nil, err
	}
	return gitReleaseInfoAt(repo, fetchTags, limit)
}

func gitReleaseInfoAt(repo string, fetchTags bool, limit int) (map[string]any, error) {
	if fetchTags {
		if _, err := runGit(repo, "fetch", "--tags", "--quiet"); err != nil {
			return nil, err
		}
	}
	if limit <= 0 {
		limit = 10
	}
	commit, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	currentTag, _ := runGit(repo, "describe", "--tags", "--exact-match")
	latestTag, _ := runGit(repo, "describe", "--tags", "--abbrev=0")
	tagList, err := runGit(repo, "tag", "--sort=-creatordate")
	if err != nil {
		return nil, err
	}
	allTags := make([]string, 0)
	for _, line := range strings.Split(tagList, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		allTags = append(allTags, tag)
	}
	recentTags := allTags
	if len(recentTags) > limit {
		recentTags = recentTags[:limit]
	}
	return map[string]any{
		"repo":        repo,
		"commit":      commit,
		"current_tag": strings.TrimSpace(currentTag),
		"latest_tag":  strings.TrimSpace(latestTag),
		"tag_count":   len(allTags),
		"recent_tags": recentTags,
		"fetched":     fetchTags,
	}, nil
}

func updateInstallStatus(repo string) (map[string]any, error) {
	installDir := updateInstallDir(repo)
	manifestPath := updateManifestPath(repo)
	info := map[string]any{
		"repo":          repo,
		"install_dir":   installDir,
		"manifest_path": manifestPath,
		"installed":     false,
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return nil, err
	}
	info["installed"] = true
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err == nil {
		info["manifest"] = manifest
	}
	scripts := make([]string, 0, 4)
	for _, name := range []string{"update-status.sh", "update-check.sh", "update-release.sh", "update-apply.sh"} {
		target := filepath.Join(installDir, name)
		if _, err := os.Stat(target); err == nil {
			scripts = append(scripts, target)
		}
	}
	info["scripts"] = scripts
	return info, nil
}

type updateInstallInfo struct {
	Success      bool     `json:"success"`
	Installed    bool     `json:"installed"`
	Repo         string   `json:"repo"`
	InstallDir   string   `json:"install_dir"`
	ManifestPath string   `json:"manifest_path"`
	Scripts      []string `json:"scripts,omitempty"`
	Removed      []string `json:"removed,omitempty"`
}

func installUpdateScripts(repoPath string) (updateInstallInfo, error) {
	repo, err := resolveUpdateRepoRoot(repoPath)
	if err != nil {
		return updateInstallInfo{}, err
	}
	installDir := updateInstallDir(repo)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return updateInstallInfo{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return updateInstallInfo{}, err
	}
	scriptSpecs := []struct {
		name string
		args string
	}{
		{name: "update-status.sh", args: "update status"},
		{name: "update-check.sh", args: "update -fetch"},
		{name: "update-release.sh", args: "update release"},
		{name: "update-apply.sh", args: "update apply"},
	}
	scripts := make([]string, 0, len(scriptSpecs))
	for _, spec := range scriptSpecs {
		target := filepath.Join(installDir, spec.name)
		content := "#!/usr/bin/env bash\nset -euo pipefail\ncd " + shellQuote(repo) + "\nexec " + shellQuote(exe) + " " + spec.args + " \"$@\"\n"
		if err := os.WriteFile(target, []byte(content), 0o755); err != nil {
			return updateInstallInfo{}, err
		}
		scripts = append(scripts, target)
	}
	manifestPath := updateManifestPath(repo)
	manifest := map[string]any{
		"installed":    true,
		"installed_at": time.Now().Format(time.RFC3339Nano),
		"repo":         repo,
		"executable":   exe,
		"scripts":      scripts,
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return updateInstallInfo{}, err
	}
	return updateInstallInfo{
		Success:      true,
		Installed:    true,
		Repo:         repo,
		InstallDir:   installDir,
		ManifestPath: manifestPath,
		Scripts:      scripts,
	}, nil
}

func uninstallUpdateScripts(repoPath string) (updateInstallInfo, error) {
	repo, err := resolveUpdateRepoRoot(repoPath)
	if err != nil {
		return updateInstallInfo{}, err
	}
	removed := make([]string, 0, 3)
	for _, name := range []string{"update-status.sh", "update-check.sh", "update-release.sh", "update-apply.sh"} {
		target := filepath.Join(updateInstallDir(repo), name)
		if err := os.Remove(target); err == nil {
			removed = append(removed, target)
		} else if err != nil && !os.IsNotExist(err) {
			return updateInstallInfo{}, err
		}
	}
	manifestPath := updateManifestPath(repo)
	if err := os.Remove(manifestPath); err == nil {
		removed = append(removed, manifestPath)
	} else if err != nil && !os.IsNotExist(err) {
		return updateInstallInfo{}, err
	}
	return updateInstallInfo{
		Success:      true,
		Installed:    false,
		Repo:         repo,
		InstallDir:   updateInstallDir(repo),
		ManifestPath: manifestPath,
		Removed:      removed,
	}, nil
}

func resolveUpdateRepoRoot(repoPath string) (string, error) {
	if strings.TrimSpace(repoPath) != "" {
		return gitRepoRootAt(strings.TrimSpace(repoPath))
	}
	return gitRepoRoot()
}

func gitRepoRootAt(workdir string) (string, error) {
	repo, err := runGit(workdir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("update is only available in a git checkout: %w", err)
	}
	return repo, nil
}

func updateInstallDir(repo string) string {
	return filepath.Join(repo, ".agent-daemon", "bin")
}

func updateManifestPath(repo string) string {
	return filepath.Join(updateInstallDir(repo), "update-install.json")
}

func runBootstrap(cfg config.Config, args []string) {
	if len(args) == 0 {
		runBootstrapInit(cfg, nil)
		return
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "init":
		runBootstrapInit(cfg, args[1:])
	case "status":
		runBootstrapStatus(cfg, args[1:])
	default:
		printBootstrapUsage()
		os.Exit(2)
	}
}

func runBootstrapInit(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("bootstrap init", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
	dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd bootstrap init [-file path] [-workdir dir] [-data-dir dir] [-json]")
	}
	status, err := bootstrapWorkspace(*path, strings.TrimSpace(*workdir), strings.TrimSpace(*dataDir))
	if err != nil {
		log.Fatal(err)
	}
	if *jsonOutput {
		printJSON(status)
		return
	}
	fmt.Printf("bootstrapped config=%s\n", status.ConfigPath)
	fmt.Printf("workdir=%s\n", status.Workdir)
	fmt.Printf("data_dir=%s\n", status.DataDir)
	if len(status.Created) > 0 {
		fmt.Println("created=" + strings.Join(status.Created, ","))
	} else {
		fmt.Println("created=")
	}
	if len(status.Existing) > 0 {
		fmt.Println("existing=" + strings.Join(status.Existing, ","))
	} else {
		fmt.Println("existing=")
	}
}

func runBootstrapStatus(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("bootstrap status", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
	dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd bootstrap status [-file path] [-workdir dir] [-data-dir dir] [-json]")
	}
	status := inspectBootstrapStatus(*path, strings.TrimSpace(*workdir), strings.TrimSpace(*dataDir))
	if *jsonOutput {
		printJSON(status)
		return
	}
	fmt.Printf("config=%s exists=%t\n", status.ConfigPath, status.ConfigExists)
	fmt.Printf("workdir=%s exists=%t\n", status.Workdir, status.WorkdirExists)
	fmt.Printf("state_dir=%s exists=%t\n", status.StateDir, status.StateDirExists)
	fmt.Printf("data_dir=%s exists=%t\n", status.DataDir, status.DataDirExists)
	fmt.Printf("processes_dir=%s exists=%t\n", status.ProcessesDir, status.ProcessesDirExists)
	fmt.Printf("memory_file=%s exists=%t\n", status.MemoryFile, status.MemoryFileExists)
	fmt.Printf("user_file=%s exists=%t\n", status.UserFile, status.UserFileExists)
}

type bootstrapStatus struct {
	Success            bool     `json:"success"`
	ConfigPath         string   `json:"config_path"`
	ConfigExists       bool     `json:"config_exists"`
	Workdir            string   `json:"workdir"`
	WorkdirExists      bool     `json:"workdir_exists"`
	StateDir           string   `json:"state_dir"`
	StateDirExists     bool     `json:"state_dir_exists"`
	DataDir            string   `json:"data_dir"`
	DataDirExists      bool     `json:"data_dir_exists"`
	ProcessesDir       string   `json:"processes_dir"`
	ProcessesDirExists bool     `json:"processes_dir_exists"`
	MemoryFile         string   `json:"memory_file"`
	MemoryFileExists   bool     `json:"memory_file_exists"`
	UserFile           string   `json:"user_file"`
	UserFileExists     bool     `json:"user_file_exists"`
	Created            []string `json:"created,omitempty"`
	Existing           []string `json:"existing,omitempty"`
}

func bootstrapWorkspace(path, workdir, dataDir string) (bootstrapStatus, error) {
	status := inspectBootstrapStatus(path, workdir, dataDir)
	created := make([]string, 0, 8)
	existing := make([]string, 0, 8)
	ensureDir := func(label, target string) error {
		if target == "" {
			return nil
		}
		if fileExists(target) {
			existing = append(existing, label)
			return nil
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		created = append(created, label)
		return nil
	}
	ensureFile := func(label, target string) error {
		if target == "" {
			return nil
		}
		if fileExists(target) {
			existing = append(existing, label)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
			return err
		}
		created = append(created, label)
		return nil
	}
	if err := ensureDir("workdir", status.Workdir); err != nil {
		return bootstrapStatus{}, err
	}
	if err := ensureDir("state_dir", status.StateDir); err != nil {
		return bootstrapStatus{}, err
	}
	if err := ensureDir("data_dir", status.DataDir); err != nil {
		return bootstrapStatus{}, err
	}
	if err := ensureDir("processes_dir", status.ProcessesDir); err != nil {
		return bootstrapStatus{}, err
	}
	if err := ensureFile("memory_file", status.MemoryFile); err != nil {
		return bootstrapStatus{}, err
	}
	if err := ensureFile("user_file", status.UserFile); err != nil {
		return bootstrapStatus{}, err
	}
	if err := config.SaveConfigValue(path, "agent.workdir", status.Workdir); err != nil {
		return bootstrapStatus{}, err
	}
	if err := config.SaveConfigValue(path, "agent.data_dir", status.DataDir); err != nil {
		return bootstrapStatus{}, err
	}
	status = inspectBootstrapStatus(path, workdir, dataDir)
	status.Success = true
	status.Created = created
	status.Existing = existing
	return status, nil
}

func inspectBootstrapStatus(path, workdir, dataDir string) bootstrapStatus {
	configPath := config.ConfigFilePath(path)
	if strings.TrimSpace(workdir) == "" {
		workdir = "."
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = filepath.Join(workdir, ".agent-daemon")
	}
	return bootstrapStatus{
		Success:            true,
		ConfigPath:         configPath,
		ConfigExists:       fileExists(configPath),
		Workdir:            workdir,
		WorkdirExists:      fileExists(workdir),
		StateDir:           filepath.Join(workdir, ".agent-daemon"),
		StateDirExists:     fileExists(filepath.Join(workdir, ".agent-daemon")),
		DataDir:            dataDir,
		DataDirExists:      fileExists(dataDir),
		ProcessesDir:       filepath.Join(dataDir, "processes"),
		ProcessesDirExists: fileExists(filepath.Join(dataDir, "processes")),
		MemoryFile:         filepath.Join(dataDir, "MEMORY.md"),
		MemoryFileExists:   fileExists(filepath.Join(dataDir, "MEMORY.md")),
		UserFile:           filepath.Join(dataDir, "USER.md"),
		UserFileExists:     fileExists(filepath.Join(dataDir, "USER.md")),
	}
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")
	checkUpdate := fs.Bool("check-update", false, "check git upstream update status")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd version [-json] [-check-update]")
	}
	info := map[string]any{
		"app_version":  appVersion,
		"release_date": releaseDate,
		"build_commit": buildCommit,
		"go_version":   runtime.Version(),
	}
	if strings.TrimSpace(buildCommit) == "" {
		if commit, err := runGit("", "rev-parse", "HEAD"); err == nil {
			info["git_commit"] = commit
		}
	}
	if *checkUpdate {
		if status, err := gitUpdateStatus(true); err == nil {
			info["update"] = status
		} else {
			info["update_error"] = err.Error()
		}
	}
	if *jsonOutput {
		printJSON(info)
		return
	}
	fmt.Printf("agent-daemon %s\n", appVersion)
	if strings.TrimSpace(releaseDate) != "" {
		fmt.Printf("release_date=%s\n", releaseDate)
	}
	if v, ok := info["build_commit"].(string); ok && strings.TrimSpace(v) != "" {
		fmt.Printf("build_commit=%s\n", v)
	} else if v, ok := info["git_commit"].(string); ok && strings.TrimSpace(v) != "" {
		fmt.Printf("git_commit=%s\n", v)
	}
	fmt.Printf("go_version=%s\n", runtime.Version())
	if *checkUpdate {
		if status, ok := info["update"].(map[string]any); ok {
			fmt.Printf("update_upstream=%v\n", status["upstream"])
			fmt.Printf("update_ahead=%v\n", status["ahead"])
			fmt.Printf("update_behind=%v\n", status["behind"])
			fmt.Printf("update_dirty=%v\n", status["dirty"])
		} else if errText, ok := info["update_error"].(string); ok && strings.TrimSpace(errText) != "" {
			fmt.Printf("update_error=%s\n", errText)
		}
	}
}

func gitUpdateStatus(fetch bool) (map[string]any, error) {
	repo, err := gitRepoRoot()
	if err != nil {
		return nil, err
	}
	return gitUpdateStatusAt(repo, fetch)
}

func gitUpdateStatusAt(repo string, fetch bool) (map[string]any, error) {
	if fetch {
		if _, err := runGit(repo, "fetch", "--quiet"); err != nil {
			return nil, err
		}
	}
	branch, err := runGit(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	commit, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	upstream, err := runGit(repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return nil, fmt.Errorf("update requires a configured git upstream: %w", err)
	}
	counts, err := runGit(repo, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(counts)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected rev-list output: %q", counts)
	}
	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	dirtyOut, err := runGit(repo, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"repo":             repo,
		"branch":           branch,
		"commit":           commit,
		"upstream":         upstream,
		"ahead":            ahead,
		"behind":           behind,
		"dirty":            strings.TrimSpace(dirtyOut) != "",
		"can_fast_forward": behind > 0 && ahead == 0,
	}, nil
}

func gitUpdateApply() (map[string]any, error) {
	repo, err := gitRepoRoot()
	if err != nil {
		return nil, err
	}
	before, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	branch, err := runGit(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	if _, err := runGit(repo, "fetch", "--quiet"); err != nil {
		return nil, err
	}
	if _, err := runGit(repo, "pull", "--ff-only"); err != nil {
		return nil, err
	}
	after, err := runGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"repo":    repo,
		"branch":  branch,
		"before":  before,
		"after":   after,
		"updated": before != after,
	}, nil
}

func gitRepoRoot() (string, error) {
	repo, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("update is only available in a git checkout: %w", err)
	}
	return repo, nil
}

func runGit(workdir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(workdir) != "" {
		cmd.Dir = workdir
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), text)
	}
	return text, nil
}

func runSetupWizard(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("setup wizard", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd setup wizard [-file path]")
	}
	reader := bufio.NewReader(os.Stdin)
	currentProvider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	if currentProvider == "" {
		currentProvider = "openai"
	}
	currentModel, currentBaseURL := currentModelConfig(cfg, currentProvider)
	provider := promptInput(reader, "provider [openai/anthropic/codex or plugin]", currentProvider)
	if !isProviderAvailable(cfg, provider) {
		log.Fatalf("unsupported provider: %s", provider)
	}
	if provider != currentProvider {
		currentModel = ""
		currentBaseURL = ""
	}
	modelName := promptInput(reader, "model", currentModel)
	if strings.TrimSpace(modelName) == "" {
		log.Fatal("model is required")
	}
	baseURL := promptInput(reader, "base URL (optional)", currentBaseURL)
	apiKey := promptInput(reader, "API key (optional)", "")
	fallback := promptInput(reader, "fallback provider (optional)", strings.ToLower(strings.TrimSpace(cfg.ModelFallbackProvider)))
	if fallback != "" && !isProviderAvailable(cfg, fallback) {
		log.Fatalf("unsupported fallback provider: %s", fallback)
	}
	gatewayPlatform := strings.ToLower(strings.TrimSpace(promptInput(reader, "gateway platform [none/telegram/discord/slack/yuanbao]", "none")))
	if gatewayPlatform == "none" {
		gatewayPlatform = ""
	}
	gatewayToken := ""
	gatewayBotToken := ""
	gatewayAppToken := ""
	gatewayAppID := ""
	gatewayAppSecret := ""
	gatewayAllowedUsers := ""
	switch gatewayPlatform {
	case "":
	case "telegram", "discord":
		gatewayToken = promptInput(reader, "gateway token", "")
		gatewayAllowedUsers = promptInput(reader, "gateway allowed users (optional)", "")
	case "slack":
		gatewayBotToken = promptInput(reader, "slack bot token", "")
		gatewayAppToken = promptInput(reader, "slack app token", "")
		gatewayAllowedUsers = promptInput(reader, "gateway allowed users (optional)", "")
	case "yuanbao":
		gatewayToken = promptInput(reader, "yuanbao token (optional)", "")
		gatewayAppID = promptInput(reader, "yuanbao app id (optional)", "")
		gatewayAppSecret = promptInput(reader, "yuanbao app secret (optional)", "")
		gatewayAllowedUsers = promptInput(reader, "gateway allowed users (optional)", "")
	default:
		log.Fatalf("unsupported gateway platform: %s", gatewayPlatform)
	}
	targetPath := config.ConfigFilePath(*path)
	written, selectedGateway, err := applySetupConfig(
		targetPath,
		provider,
		modelName,
		baseURL,
		apiKey,
		fallback,
		gatewayPlatform,
		gatewayToken,
		gatewayBotToken,
		gatewayAppToken,
		gatewayAppID,
		gatewayAppSecret,
		gatewayAllowedUsers,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("configured provider %s:%s in %s\n", provider, modelName, targetPath)
	if selectedGateway != "" {
		fmt.Printf("configured gateway platform %s\n", selectedGateway)
	}
	fmt.Printf("written=%s\n", strings.Join(written, ","))
}

func promptInput(reader *bufio.Reader, label, def string) string {
	if strings.TrimSpace(def) != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		log.Fatal(err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return strings.TrimSpace(def)
	}
	return line
}

func applySetupConfig(targetPath, provider, modelName, baseURL, apiKey, fallback, gatewayPlatform, gatewayToken, gatewayBotToken, gatewayAppToken, gatewayAppID, gatewayAppSecret, gatewayAllowedUsers string) ([]string, string, error) {
	if err := saveModelSelection(targetPath, provider, modelName, baseURL); err != nil {
		return nil, "", err
	}
	written := []string{"api.type"}
	modelKey, baseURLKey := modelConfigKeys(provider)
	written = append(written, modelKey)
	if baseURL != "" {
		written = append(written, baseURLKey)
	}
	if apiKeyKey := providerAPIKeyConfigKey(provider); apiKeyKey != "" && apiKey != "" {
		if err := config.SaveConfigValue(targetPath, apiKeyKey, apiKey); err != nil {
			return nil, "", err
		}
		written = append(written, apiKeyKey)
	}
	if fallback != "" {
		if err := config.SaveConfigValue(targetPath, "provider.fallback", fallback); err != nil {
			return nil, "", err
		}
		written = append(written, "provider.fallback")
	}
	selectedGateway := strings.ToLower(strings.TrimSpace(gatewayPlatform))
	if selectedGateway != "" {
		gatewayWritten, err := setupGatewayConfig(targetPath, selectedGateway, gatewayToken, gatewayBotToken, gatewayAppToken, gatewayAppID, gatewayAppSecret, gatewayAllowedUsers)
		if err != nil {
			return nil, "", err
		}
		written = append(written, gatewayWritten...)
	}
	return uniqueSortedNames(written), selectedGateway, nil
}

func runToolsets(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  agentd toolsets list")
		fmt.Fprintln(os.Stderr, "  agentd toolsets show name")
		fmt.Fprintln(os.Stderr, "  agentd toolsets resolve name[,name...]")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		printJSON(tools.ListToolsets())
	case "show":
		fs := flag.NewFlagSet("toolsets show", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd toolsets show name")
		}
		name := strings.TrimSpace(fs.Arg(0))
		ts, ok := tools.GetToolset(name)
		if !ok {
			log.Fatalf("unknown toolset: %s", name)
		}
		printJSON(ts)
	case "resolve":
		fs := flag.NewFlagSet("toolsets resolve", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd toolsets resolve name[,name...]")
		}
		names := parseNameList(fs.Arg(0))
		allowed, err := tools.ResolveToolset(names)
		if err != nil {
			log.Fatal(err)
		}
		out := make([]string, 0, len(allowed))
		for name := range allowed {
			out = append(out, name)
		}
		sort.Strings(out)
		printToolNames(out)
	default:
		log.Fatal("usage: agentd toolsets list | show | resolve")
	}
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd config list [-file path] [-show-secrets]")
	fmt.Fprintln(os.Stderr, "  agentd config get [-file path] section.key")
	fmt.Fprintln(os.Stderr, "  agentd config set [-file path] section.key value")
}

func printSetupUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd setup [-file path] -provider <openai|anthropic|codex|plugin> -model <name> [-base-url url] [-api-key key] [-fallback-provider name] [-gateway-platform name] [gateway flags] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd setup wizard [-file path]")
}

func printBootstrapUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd bootstrap init [-file path] [-workdir dir] [-data-dir dir] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd bootstrap status [-file path] [-workdir dir] [-data-dir dir] [-json]")
}

func runModel(cfg config.Config, args []string) {
	if len(args) == 0 {
		printModelUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "show", "current":
		fs := flag.NewFlagSet("model show", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd model show [-file path]")
		}
		showCfg := cfg
		if strings.TrimSpace(*path) != "" {
			var err error
			showCfg, err = config.LoadFile(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		provider := strings.ToLower(strings.TrimSpace(showCfg.ModelProvider))
		if provider == "" {
			provider = "openai"
		}
		modelName, baseURL := currentModelConfig(showCfg, provider)
		fmt.Printf("provider=%s\n", provider)
		fmt.Printf("model=%s\n", modelName)
		fmt.Printf("base_url=%s\n", baseURL)
	case "providers":
		for _, provider := range availableModelProviders(cfg) {
			fmt.Println(provider)
		}
	case "set":
		fs := flag.NewFlagSet("model set", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		baseURL := fs.String("base-url", "", "provider base URL")
		_ = fs.Parse(args[1:])
		provider, modelName, err := parseModelSetArgs(fs.Args())
		if err != nil {
			log.Fatal(err)
		}
		if !isProviderAvailable(cfg, provider) {
			log.Fatalf("unsupported provider: %s", provider)
		}
		if err := saveModelSelection(*path, provider, modelName, *baseURL); err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(*baseURL) == "" {
			fmt.Printf("updated model to %s:%s in %s\n", provider, modelName, config.ConfigFilePath(*path))
		} else {
			fmt.Printf("updated model to %s:%s (%s) in %s\n", provider, modelName, *baseURL, config.ConfigFilePath(*path))
		}
	default:
		printModelUsage()
		os.Exit(2)
	}
}

func printModelUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd model show [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd model providers  # includes builtins + provider plugins")
	fmt.Fprintln(os.Stderr, "  agentd model set [-file path] [-base-url url] provider model")
	fmt.Fprintln(os.Stderr, "  agentd model set [-file path] [-base-url url] provider:model")
}

func runTools(cfg config.Config, args []string) {
	if len(args) == 0 {
		eng, _ := mustBuildEngine(cfg)
		printToolNames(eng.Registry.Names())
		return
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("tools list", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools list")
		}
		eng, _ := mustBuildEngine(cfg)
		printToolNames(eng.Registry.Names())
	case "show":
		fs := flag.NewFlagSet("tools show", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd tools show tool_name")
		}
		eng, _ := mustBuildEngine(cfg)
		schema, ok := findToolSchema(eng.Registry.Schemas(), fs.Arg(0))
		if !ok {
			log.Fatalf("unknown tool: %s", fs.Arg(0))
		}
		printJSON(schema)
	case "schemas":
		fs := flag.NewFlagSet("tools schemas", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools schemas")
		}
		eng, _ := mustBuildEngine(cfg)
		printJSON(eng.Registry.Schemas())
	case "disabled":
		fs := flag.NewFlagSet("tools disabled", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools disabled [-file path]")
		}
		disabled := parseNameList(cfg.DisabledTools)
		if strings.TrimSpace(*path) != "" {
			var err error
			disabled, err = readDisabledToolsConfig(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		printToolNames(disabled)
	case "disable":
		path, toolName := parseToolToggleArgs(args[1:], "tools disable")
		disabled, err := readDisabledToolsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = addName(disabled, toolName)
		if err := config.SaveConfigValue(path, "tools.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("disabled tool %s in %s\n", toolName, config.ConfigFilePath(path))
	case "enable":
		path, toolName := parseToolToggleArgs(args[1:], "tools enable")
		disabled, err := readDisabledToolsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = removeName(disabled, toolName)
		if err := config.SaveConfigValue(path, "tools.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("enabled tool %s in %s\n", toolName, config.ConfigFilePath(path))
	default:
		printToolsUsage()
		os.Exit(2)
	}
}

func runPlugins(cfg config.Config, args []string) {
	if len(args) == 0 {
		printPluginsUsage()
		return
	}
	switch args[0] {
	case "list":
		items, err := loadConfiguredPlugins(cfg)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(items)
	case "show":
		if len(args) != 2 {
			log.Fatal("usage: agentd plugins show name")
		}
		name := strings.TrimSpace(args[1])
		items, err := loadConfiguredPlugins(cfg)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.Name), name) {
				printJSON(item)
				return
			}
		}
		log.Fatalf("plugin %q not found", name)
	case "validate":
		items, err := plugins.LoadFromDirs(plugins.DefaultDirs(cfg.Workdir))
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			if err := plugins.ValidateManifest(item); err != nil {
				log.Fatalf("invalid plugin %s (%s): %v", item.Name, item.File, err)
			}
		}
		fmt.Printf("ok (%d manifests)\n", len(items))
	case "disable":
		path, name := parsePluginToggleArgs(args[1:], "plugins disable")
		disabled, err := readDisabledPluginsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = addName(disabled, name)
		if err := config.SaveConfigValue(path, "plugins.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("disabled plugin %s in %s\n", name, config.ConfigFilePath(path))
	case "enable":
		path, name := parsePluginToggleArgs(args[1:], "plugins enable")
		disabled, err := readDisabledPluginsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = removeName(disabled, name)
		if err := config.SaveConfigValue(path, "plugins.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("enabled plugin %s in %s\n", name, config.ConfigFilePath(path))
	default:
		printPluginsUsage()
	}
}

func printToolNames(names []string) {
	for _, name := range names {
		fmt.Println(name)
	}
}

func printToolsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd tools")
	fmt.Fprintln(os.Stderr, "  agentd tools list")
	fmt.Fprintln(os.Stderr, "  agentd tools show tool_name")
	fmt.Fprintln(os.Stderr, "  agentd tools schemas")
	fmt.Fprintln(os.Stderr, "  agentd tools disabled [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd tools disable [-file path] tool_name")
	fmt.Fprintln(os.Stderr, "  agentd tools enable [-file path] tool_name")
}

func printPluginsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd plugins list")
	fmt.Fprintln(os.Stderr, "  agentd plugins show name")
	fmt.Fprintln(os.Stderr, "  agentd plugins validate")
	fmt.Fprintln(os.Stderr, "  agentd plugins disable [-file path] name")
	fmt.Fprintln(os.Stderr, "  agentd plugins enable [-file path] name")
}

func findToolSchema(schemas []core.ToolSchema, name string) (core.ToolSchema, bool) {
	name = strings.TrimSpace(name)
	for _, schema := range schemas {
		if schema.Function.Name == name {
			return schema, true
		}
	}
	return core.ToolSchema{}, false
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func parseToolToggleArgs(args []string, name string) (string, string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: agentd %s [-file path] tool_name", name)
	}
	toolName := strings.TrimSpace(fs.Arg(0))
	if toolName == "" {
		log.Fatal("tool_name is required")
	}
	return *path, toolName
}

func parsePluginToggleArgs(args []string, name string) (string, string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: agentd %s [-file path] name", name)
	}
	pluginName := strings.TrimSpace(fs.Arg(0))
	if pluginName == "" {
		log.Fatal("name is required")
	}
	return *path, pluginName
}

func readDisabledPluginsConfig(path string) ([]string, error) {
	value, ok, err := config.ReadConfigValue(path, "plugins.disabled")
	if err != nil {
		return nil, err
	}
	if !ok {
		return []string{}, nil
	}
	return parseNameList(value), nil
}

func loadConfiguredPlugins(cfg config.Config) ([]plugins.Manifest, error) {
	items, err := plugins.LoadFromDirs(plugins.DefaultDirs(cfg.Workdir))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.DisabledPlugins) == "" {
		return items, nil
	}
	disabled := map[string]struct{}{}
	for _, name := range parseNameList(cfg.DisabledPlugins) {
		disabled[strings.ToLower(name)] = struct{}{}
	}
	out := make([]plugins.Manifest, 0, len(items))
	for _, item := range items {
		if _, ok := disabled[strings.ToLower(strings.TrimSpace(item.Name))]; ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func readDisabledToolsConfig(path string) ([]string, error) {
	value, ok, err := config.ReadConfigValue(path, "tools.disabled")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return parseNameList(value), nil
}

func parseNameList(value string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func addName(names []string, name string) []string {
	names = append(names, name)
	return parseNameList(strings.Join(names, ","))
}

func removeName(names []string, name string) []string {
	name = strings.TrimSpace(name)
	var out []string
	for _, item := range names {
		if item != name {
			out = append(out, item)
		}
	}
	return parseNameList(strings.Join(out, ","))
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func runDoctor(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd doctor [-json]")
	}
	checks := buildDoctorChecks(cfg)
	if *jsonOutput {
		printJSON(checks)
	} else {
		for _, check := range checks {
			fmt.Printf("[%s] %s: %s\n", check.Status, check.Name, check.Detail)
		}
	}
	if hasDoctorError(checks) {
		os.Exit(1)
	}
}

func buildDoctorChecks(cfg config.Config) []doctorCheck {
	checks := []doctorCheck{
		{Name: "config_file", Status: "ok", Detail: "using " + config.ConfigFilePath("") + " when present; environment variables take precedence"},
		checkDirectory("workdir", cfg.Workdir, false),
		checkDirectory("data_dir", cfg.DataDir, true),
		checkModelConfig(cfg),
		checkProviderCredentials(cfg),
		checkMCPConfig(cfg),
		checkGatewayConfig(cfg),
		checkPluginsConfig(cfg),
		checkToolsetsConfig(cfg),
		checkStubTools(cfg),
		checkRegisteredTools(),
		checkToolCapabilities(cfg),
	}
	return checks
}

func checkDirectory(name, path string, create bool) doctorCheck {
	path = strings.TrimSpace(path)
	if path == "" {
		return doctorCheck{Name: name, Status: "error", Detail: "path is empty"}
	}
	if create {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
		}
		f, err := os.CreateTemp(path, ".agentd-doctor-*")
		if err != nil {
			return doctorCheck{Name: name, Status: "error", Detail: "not writable: " + err.Error()}
		}
		tmpName := f.Name()
		_ = f.Close()
		_ = os.Remove(tmpName)
		return doctorCheck{Name: name, Status: "ok", Detail: path + " is writable"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: name, Status: "error", Detail: "not a directory: " + path}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: path}
}

func checkModelConfig(cfg config.Config) doctorCheck {
	provider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	if provider == "" {
		provider = "openai"
	}
	if _, _, err := normalizeModelSelection(provider, selectedModelName(cfg, provider)); err != nil {
		return doctorCheck{Name: "model", Status: "error", Detail: err.Error()}
	}
	modelName, baseURL := currentModelConfig(cfg, provider)
	return doctorCheck{Name: "model", Status: "ok", Detail: fmt.Sprintf("%s:%s (%s)", provider, modelName, baseURL)}
}

func checkProviderCredentials(cfg config.Config) doctorCheck {
	provider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	if provider == "" {
		provider = "openai"
	}
	if !containsName(supportedModelProviders(), provider) {
		return doctorCheck{Name: "provider_credentials", Status: "ok", Detail: "plugin provider credentials managed by plugin runtime"}
	}
	keyName, value := selectedProviderKey(cfg, provider)
	if strings.TrimSpace(value) == "" {
		return doctorCheck{Name: "provider_credentials", Status: "warn", Detail: keyName + " is empty"}
	}
	return doctorCheck{Name: "provider_credentials", Status: "ok", Detail: keyName + " is set"}
}

func selectedProviderKey(cfg config.Config, provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "ANTHROPIC_API_KEY", cfg.AnthropicAPIKey
	case "codex":
		return "CODEX_API_KEY", cfg.CodexAPIKey
	default:
		return "OPENAI_API_KEY", cfg.ModelAPIKey
	}
}

func selectedModelName(cfg config.Config, provider string) string {
	modelName, _ := currentModelConfig(cfg, provider)
	return modelName
}

func checkMCPConfig(cfg config.Config) doctorCheck {
	transport := strings.ToLower(strings.TrimSpace(cfg.MCPTransport))
	if transport == "" {
		transport = "http"
	}
	switch transport {
	case "http":
		if strings.TrimSpace(cfg.MCPEndpoint) == "" {
			return doctorCheck{Name: "mcp", Status: "ok", Detail: "disabled"}
		}
		return doctorCheck{Name: "mcp", Status: "ok", Detail: "http endpoint configured"}
	case "stdio":
		if strings.TrimSpace(cfg.MCPStdioCommand) == "" {
			return doctorCheck{Name: "mcp", Status: "warn", Detail: "stdio transport selected but command is empty"}
		}
		return doctorCheck{Name: "mcp", Status: "ok", Detail: "stdio command configured"}
	default:
		return doctorCheck{Name: "mcp", Status: "error", Detail: "unsupported transport: " + transport}
	}
}

func checkGatewayConfig(cfg config.Config) doctorCheck {
	if !cfg.GatewayEnabled {
		return doctorCheck{Name: "gateway", Status: "ok", Detail: "disabled"}
	}
	configured := make([]string, 0, 3)
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		configured = append(configured, "telegram")
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		configured = append(configured, "discord")
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		configured = append(configured, "slack")
	}
	if len(configured) == 0 {
		return doctorCheck{Name: "gateway", Status: "warn", Detail: "enabled but no platform tokens are configured"}
	}
	return doctorCheck{Name: "gateway", Status: "ok", Detail: "configured platforms: " + strings.Join(configured, ",")}
}

func checkPluginsConfig(cfg config.Config) doctorCheck {
	items, err := loadConfiguredPlugins(cfg)
	if err != nil {
		return doctorCheck{Name: "plugins", Status: "error", Detail: err.Error()}
	}
	if len(items) == 0 {
		return doctorCheck{Name: "plugins", Status: "ok", Detail: "none discovered"}
	}
	enabled := 0
	for _, item := range items {
		if item.IsEnabled() {
			enabled++
		}
	}
	return doctorCheck{Name: "plugins", Status: "ok", Detail: fmt.Sprintf("discovered=%d enabled=%d", len(items), enabled)}
}

func checkToolsetsConfig(cfg config.Config) doctorCheck {
	names := parseNameList(cfg.EnabledToolsets)
	if len(names) == 0 {
		return doctorCheck{Name: "toolsets", Status: "ok", Detail: "disabled (full tool registry enabled)"}
	}
	allowed, err := tools.ResolveToolset(names)
	if err != nil {
		return doctorCheck{Name: "toolsets", Status: "error", Detail: err.Error()}
	}
	return doctorCheck{Name: "toolsets", Status: "ok", Detail: fmt.Sprintf("enabled=%s (resolved_tools=%d)", strings.Join(names, ","), len(allowed))}
}

func checkStubTools(cfg config.Config) doctorCheck {
	enabledToolsets := parseNameList(cfg.EnabledToolsets)
	allowed := map[string]struct{}{}
	if len(enabledToolsets) > 0 {
		resolved, err := tools.ResolveToolset(enabledToolsets)
		if err != nil {
			return doctorCheck{Name: "stub_tools", Status: "warn", Detail: "cannot resolve toolsets: " + err.Error()}
		}
		allowed = resolved
	}
	// Keep only real interface-alignment stubs here.
	// Lightweight/minimal implementations should not be listed as "not implemented".
	stubs := []string{}
	var present []string
	if len(allowed) == 0 {
		// toolsets disabled: stubs are registered, but that does not mean usable.
		present = stubs
	} else {
		for _, name := range stubs {
			if _, ok := allowed[name]; ok {
				present = append(present, name)
			}
		}
	}
	if len(present) == 0 {
		return doctorCheck{Name: "stub_tools", Status: "ok", Detail: "none enabled"}
	}
	return doctorCheck{Name: "stub_tools", Status: "warn", Detail: "not implemented: " + strings.Join(present, ",")}
}

func checkRegisteredTools() doctorCheck {
	registry := tools.NewRegistry()
	procDir, err := os.MkdirTemp("", "agentd-doctor-tools-*")
	if err != nil {
		return doctorCheck{Name: "tools", Status: "error", Detail: err.Error()}
	}
	defer os.RemoveAll(procDir)
	tools.RegisterBuiltins(registry, tools.NewProcessRegistry(procDir))
	names := registry.Names()
	if len(names) == 0 {
		return doctorCheck{Name: "tools", Status: "error", Detail: "no tools registered"}
	}
	return doctorCheck{Name: "tools", Status: "ok", Detail: fmt.Sprintf("%d builtin tools registered", len(names))}
}

func checkToolCapabilities(_ config.Config) doctorCheck {
	items := make([]string, 0, 4)
	status := "ok"
	if strings.TrimSpace(os.Getenv("BROWSER_CDP_URL")) != "" {
		items = append(items, "browser_cdp=on")
	} else {
		items = append(items, "browser_cdp=off")
	}
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		items = append(items, "openai_media=on")
	} else {
		items = append(items, "openai_media=off")
		status = "warn"
	}
	if strings.TrimSpace(os.Getenv("FAL_KEY")) != "" {
		items = append(items, "fal_image=on")
	} else {
		items = append(items, "fal_image=off")
	}
	return doctorCheck{Name: "tool_capabilities", Status: status, Detail: strings.Join(items, ",")}
}

func hasDoctorError(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "error" {
			return true
		}
	}
	return false
}

func runGateway(cfg config.Config, args []string) {
	if len(args) == 0 {
		printGatewayUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "run":
		runGatewayRun(cfg, args[1:])
	case "start":
		runGatewayStart(cfg, args[1:])
	case "stop":
		runGatewayStop(cfg, args[1:])
	case "restart":
		runGatewayRestart(cfg, args[1:])
	case "install":
		runGatewayInstall(cfg, args[1:])
	case "uninstall":
		runGatewayUninstall(cfg, args[1:])
	case "manifest":
		runGatewayManifest(args[1:])
	case "status":
		fs := flag.NewFlagSet("gateway status", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway status [-file path] [-json]")
		}
		statusCfg := cfg
		if strings.TrimSpace(*path) != "" {
			var err error
			statusCfg, err = config.LoadFile(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		status := gatewayStatus(statusCfg)
		if *jsonOutput {
			printJSON(status)
			return
		}
		fmt.Printf("enabled=%t\n", status.Enabled)
		fmt.Printf("running=%t\n", status.Running)
		if status.PID > 0 {
			fmt.Printf("pid=%d\n", status.PID)
		} else {
			fmt.Println("pid=")
		}
		fmt.Printf("locked=%t\n", status.Locked)
		fmt.Printf("stale_lock=%t\n", status.StaleLock)
		if status.LockPID > 0 {
			fmt.Printf("lock_pid=%d\n", status.LockPID)
		} else {
			fmt.Println("lock_pid=")
		}
		fmt.Println("lock_path=" + status.LockPath)
		fmt.Printf("token_locked=%t\n", status.TokenLocked)
		fmt.Printf("stale_token_lock=%t\n", status.StaleTokenLock)
		if status.TokenLockPID > 0 {
			fmt.Printf("token_lock_pid=%d\n", status.TokenLockPID)
		} else {
			fmt.Println("token_lock_pid=")
		}
		fmt.Println("token_lock_path=" + status.TokenLockPath)
		fmt.Printf("installed=%t\n", status.Installed)
		fmt.Println("install_dir=" + status.InstallDir)
		fmt.Println("manifest_path=" + status.ManifestPath)
		fmt.Println("pid_path=" + status.PIDPath)
		fmt.Println("log_path=" + status.LogPath)
		if len(status.ConfiguredPlatforms) == 0 {
			fmt.Println("configured_platforms=")
		} else {
			fmt.Println("configured_platforms=" + strings.Join(status.ConfiguredPlatforms, ","))
		}
		fmt.Println("supported_platforms=" + strings.Join(status.SupportedPlatforms, ","))
	case "platforms":
		for _, platform := range supportedGatewayPlatforms() {
			fmt.Println(platform)
		}
	case "enable":
		path := parseGatewayConfigPath(args[1:], "gateway enable")
		if err := config.SaveConfigValue(path, "gateway.enabled", "true"); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("enabled gateway in %s\n", config.ConfigFilePath(path))
	case "disable":
		path := parseGatewayConfigPath(args[1:], "gateway disable")
		if err := config.SaveConfigValue(path, "gateway.enabled", "false"); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("disabled gateway in %s\n", config.ConfigFilePath(path))
	case "setup":
		runGatewaySetup(args[1:])
	case "pairs":
		runGatewayPairs(cfg, args[1:])
	case "hooks":
		runGatewayHooks(cfg, args[1:])
	default:
		printGatewayUsage()
		os.Exit(2)
	}
}

func runGatewayHooks(cfg config.Config, args []string) {
	if len(args) == 0 {
		printGatewayHooksUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "spool":
		runGatewayHookSpool(cfg, args[1:])
	case "ping":
		runGatewayHookPing(cfg, args[1:])
	case "doctor":
		runGatewayHookDoctor(cfg, args[1:])
	default:
		printGatewayHooksUsage()
		os.Exit(2)
	}
}

func runGatewayRun(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway run", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway run [-file path]")
	}
	runCfg := cfg
	if strings.TrimSpace(*path) != "" {
		var err error
		runCfg, err = config.LoadFile(*path)
		if err != nil {
			log.Fatal(err)
		}
	}
	runGatewayForeground(runCfg)
}

func runGatewayInstall(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway install", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway install [-file path] [-workdir dir] [-json]")
	}
	installCfg := cfg
	if strings.TrimSpace(*path) != "" {
		var err error
		installCfg, err = config.LoadFile(*path)
		if err != nil {
			log.Fatal(err)
		}
	}
	if strings.TrimSpace(*workdir) != "" {
		installCfg.Workdir = strings.TrimSpace(*workdir)
	}
	result, err := installGatewayScripts(installCfg, *path)
	if err != nil {
		log.Fatal(err)
	}
	if *jsonOutput {
		printJSON(result)
		return
	}
	fmt.Printf("installed gateway scripts in %s\n", result.InstallDir)
	fmt.Println("manifest=" + result.ManifestPath)
	fmt.Println("scripts=" + strings.Join(result.Scripts, ","))
}

func runGatewayUninstall(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway uninstall", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
	stopFirst := fs.Bool("stop", false, "stop running gateway before uninstall")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway uninstall [-file path] [-workdir dir] [-stop] [-json]")
	}
	uninstallCfg := cfg
	if strings.TrimSpace(*path) != "" {
		var err error
		uninstallCfg, err = config.LoadFile(*path)
		if err != nil {
			log.Fatal(err)
		}
	}
	if strings.TrimSpace(*workdir) != "" {
		uninstallCfg.Workdir = strings.TrimSpace(*workdir)
	}
	if *stopFirst {
		stopArgs := make([]string, 0, 3)
		if strings.TrimSpace(*path) != "" {
			stopArgs = append(stopArgs, "-file", *path)
		}
		_ = stopArgs
		runGatewayStop(uninstallCfg, stopArgs)
	}
	result, err := uninstallGatewayScripts(uninstallCfg)
	if err != nil {
		log.Fatal(err)
	}
	if *jsonOutput {
		printJSON(result)
		return
	}
	fmt.Printf("uninstalled gateway scripts from %s\n", result.InstallDir)
	if len(result.Removed) > 0 {
		fmt.Println("removed=" + strings.Join(result.Removed, ","))
	} else {
		fmt.Println("removed=")
	}
}

func runGatewayStart(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway start", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway start [-file path] [-json]")
	}
	startCfg := cfg
	if strings.TrimSpace(*path) != "" {
		var err error
		startCfg, err = config.LoadFile(*path)
		if err != nil {
			log.Fatal(err)
		}
	}
	pidPath := gatewayPIDPath(startCfg)
	logPath := gatewayLogPath(startCfg)
	if running, pid := gatewayProcessStatus(startCfg); running {
		result := map[string]any{"success": true, "started": false, "running": true, "pid": pid, "pid_path": pidPath, "log_path": logPath}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Printf("gateway already running pid=%d\n", pid)
		fmt.Println("log_path=" + logPath)
		return
	}
	lockPath := gatewayLockPath(startCfg)
	if cleanupStaleGatewayLock(lockPath) {
		log.Printf("gateway start: removed stale runtime lock %s", lockPath)
	}
	if lockState := readGatewayLockState(lockPath); lockState.Alive {
		result := map[string]any{"success": true, "started": false, "running": true, "locked": true, "lock_pid": lockState.PID, "lock_path": gatewayLockPath(startCfg), "pid_path": pidPath, "log_path": logPath}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Printf("gateway already locked pid=%d\n", lockState.PID)
		fmt.Println("log_path=" + logPath)
		return
	}
	if tokenLockPath := gatewayTokenLockPath(startCfg); tokenLockPath != "" {
		if cleanupStaleGatewayLock(tokenLockPath) {
			log.Printf("gateway start: removed stale token lock %s", tokenLockPath)
		}
		if lockState := readGatewayLockState(tokenLockPath); lockState.Alive {
			result := map[string]any{"success": true, "started": false, "running": true, "token_locked": true, "token_lock_pid": lockState.PID, "token_lock_path": tokenLockPath, "pid_path": pidPath, "log_path": logPath}
			if *jsonOutput {
				printJSON(result)
				return
			}
			fmt.Printf("gateway token already locked pid=%d\n", lockState.PID)
			fmt.Println("log_path=" + logPath)
			return
		}
	}
	_ = os.Remove(pidPath)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		log.Fatal(err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	cmdArgs := []string{"gateway", "run"}
	if strings.TrimSpace(*path) != "" {
		cmdArgs = append(cmdArgs, "-file", *path)
	}
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = startCfg.Workdir
	cmd.Env = os.Environ()
	if strings.TrimSpace(*path) != "" {
		cmd.Env = append(cmd.Env, "AGENT_CONFIG_FILE="+*path)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	pid := cmd.Process.Pid
	running := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if alive, currentPID := gatewayProcessStatus(startCfg); alive {
			pid = currentPID
			running = true
			break
		}
		if !processAlive(pid) {
			break
		}
	}
	if !running && processAlive(pid) {
		running = true
	}
	result := map[string]any{"success": running, "started": running, "running": running, "pid": pid, "pid_path": pidPath, "log_path": logPath}
	if *jsonOutput {
		printJSON(result)
		return
	}
	if !running {
		log.Fatalf("gateway failed to start; inspect %s", logPath)
	}
	fmt.Printf("gateway started pid=%d\n", pid)
	fmt.Println("log_path=" + logPath)
}

func runGatewayStop(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway stop", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway stop [-file path] [-json]")
	}
	stopCfg := cfg
	if strings.TrimSpace(*path) != "" {
		var err error
		stopCfg, err = config.LoadFile(*path)
		if err != nil {
			log.Fatal(err)
		}
	}
	pidPath := gatewayPIDPath(stopCfg)
	running, pid := gatewayProcessStatus(stopCfg)
	if !running {
		_ = os.Remove(pidPath)
		result := map[string]any{"success": true, "stopped": false, "running": false, "pid": 0, "pid_path": pidPath}
		if *jsonOutput {
			printJSON(result)
			return
		}
		fmt.Println("gateway not running")
		return
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		if !processAlive(pid) {
			break
		}
	}
	_ = os.Remove(pidPath)
	stopped := !processAlive(pid)
	result := map[string]any{"success": stopped, "stopped": stopped, "running": !stopped, "pid": pid, "pid_path": pidPath}
	if *jsonOutput {
		printJSON(result)
		return
	}
	if !stopped {
		log.Fatalf("gateway stop timeout pid=%d", pid)
	}
	fmt.Printf("gateway stopped pid=%d\n", pid)
}

func runGatewayRestart(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway restart", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway restart [-file path] [-json]")
	}
	restartArgs := make([]string, 0, 3)
	if strings.TrimSpace(*path) != "" {
		restartArgs = append(restartArgs, "-file", *path)
	}
	_ = jsonOutput
	runGatewayStop(cfg, restartArgs)
	if *jsonOutput {
		restartArgs = append(restartArgs, "-json")
	}
	runGatewayStart(cfg, restartArgs)
}

type gatewayInstallInfo struct {
	Success      bool     `json:"success"`
	Installed    bool     `json:"installed"`
	InstallDir   string   `json:"install_dir"`
	ManifestPath string   `json:"manifest_path"`
	Scripts      []string `json:"scripts,omitempty"`
	Removed      []string `json:"removed,omitempty"`
}

type gatewayRuntimeLock struct {
	file *os.File
	path string
	pid  int
}

type gatewayLockSet struct {
	runtime *gatewayRuntimeLock
	token   *gatewayRuntimeLock
}

func installGatewayScripts(cfg config.Config, configPath string) (gatewayInstallInfo, error) {
	installDir := gatewayInstallDir(cfg)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return gatewayInstallInfo{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return gatewayInstallInfo{}, err
	}
	scriptSpecs := []struct {
		name string
		args string
	}{
		{name: "gateway-start.sh", args: "gateway start"},
		{name: "gateway-stop.sh", args: "gateway stop"},
		{name: "gateway-restart.sh", args: "gateway restart"},
		{name: "gateway-status.sh", args: "gateway status"},
	}
	scripts := make([]string, 0, len(scriptSpecs))
	configArg := ""
	if strings.TrimSpace(configPath) != "" {
		configArg = " -file " + shellQuote(config.ConfigFilePath(configPath))
	}
	for _, spec := range scriptSpecs {
		target := filepath.Join(installDir, spec.name)
		content := "#!/usr/bin/env bash\nset -euo pipefail\nexec " + shellQuote(exe) + " " + spec.args + configArg + " \"$@\"\n"
		if err := os.WriteFile(target, []byte(content), 0o755); err != nil {
			return gatewayInstallInfo{}, err
		}
		scripts = append(scripts, target)
	}
	manifest := map[string]any{
		"installed":    true,
		"installed_at": time.Now().Format(time.RFC3339Nano),
		"executable":   exe,
		"config_path":  config.ConfigFilePath(configPath),
		"workdir":      cfg.Workdir,
		"scripts":      scripts,
	}
	manifestPath := gatewayManifestPath(cfg)
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return gatewayInstallInfo{}, err
	}
	return gatewayInstallInfo{
		Success:      true,
		Installed:    true,
		InstallDir:   installDir,
		ManifestPath: manifestPath,
		Scripts:      scripts,
	}, nil
}

func uninstallGatewayScripts(cfg config.Config) (gatewayInstallInfo, error) {
	removed := make([]string, 0, 5)
	for _, name := range []string{"gateway-start.sh", "gateway-stop.sh", "gateway-restart.sh", "gateway-status.sh"} {
		target := filepath.Join(gatewayInstallDir(cfg), name)
		if err := os.Remove(target); err == nil {
			removed = append(removed, target)
		} else if err != nil && !os.IsNotExist(err) {
			return gatewayInstallInfo{}, err
		}
	}
	manifestPath := gatewayManifestPath(cfg)
	if err := os.Remove(manifestPath); err == nil {
		removed = append(removed, manifestPath)
	} else if err != nil && !os.IsNotExist(err) {
		return gatewayInstallInfo{}, err
	}
	return gatewayInstallInfo{
		Success:      true,
		Installed:    false,
		InstallDir:   gatewayInstallDir(cfg),
		ManifestPath: manifestPath,
		Removed:      removed,
	}, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func runGatewayForeground(cfg config.Config) {
	cfg.ModelUseStreaming = true
	eng, _ := mustBuildEngine(cfg)
	adapters := buildGatewayAdapters(cfg)
	if len(adapters) == 0 {
		log.Fatal("gateway run requires at least one configured platform adapter")
	}
	if !cfg.GatewayEnabled {
		log.Printf("gateway.enabled=false; continuing because gateway run was requested explicitly")
	}
	lock, err := acquireGatewayLocks(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Release()
	runner := gateway.NewRunner(adapters, eng, func(platform string) string {
		switch platform {
		case "telegram":
			return cfg.TelegramAllowed
		case "discord":
			return cfg.DiscordAllowed
		case "slack":
			return cfg.SlackAllowed
		case "yuanbao":
			return cfg.YuanbaoAllowed
		}
		return ""
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := runner.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(gatewayPIDPath(cfg)), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(gatewayPIDPath(cfg), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		log.Fatal(err)
	}
	defer removePIDFileIfOwned(gatewayPIDPath(cfg), os.Getpid())
	log.Printf("gateway running adapters=%s pid=%d", strings.Join(gatewayAdapterNames(adapters), ","), os.Getpid())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
	runner.Stop()
}

func acquireGatewayLocks(cfg config.Config) (*gatewayLockSet, error) {
	runtimeLock, err := acquireGatewayRuntimeLock(cfg)
	if err != nil {
		return nil, err
	}
	tokenLock, err := acquireGatewayTokenLock(cfg)
	if err != nil {
		runtimeLock.Release()
		return nil, err
	}
	return &gatewayLockSet{runtime: runtimeLock, token: tokenLock}, nil
}

func (s *gatewayLockSet) Release() {
	if s == nil {
		return
	}
	if s.token != nil {
		s.token.Release()
	}
	if s.runtime != nil {
		s.runtime.Release()
	}
}

func acquireGatewayRuntimeLock(cfg config.Config) (*gatewayRuntimeLock, error) {
	lockPath := gatewayLockPath(cfg)
	return acquirePIDFileLock(lockPath, "gateway lock")
}

func acquireGatewayTokenLock(cfg config.Config) (*gatewayRuntimeLock, error) {
	lockPath := gatewayTokenLockPath(cfg)
	if lockPath == "" {
		return nil, nil
	}
	return acquirePIDFileLock(lockPath, "gateway token lock")
}

func acquirePIDFileLock(lockPath, label string) (*gatewayRuntimeLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lockPID := readGatewayLockPID(lockPath)
		_ = file.Close()
		if lockPID > 0 {
			return nil, fmt.Errorf("%s is already held by pid=%d (%s)", label, lockPID, lockPath)
		}
		return nil, fmt.Errorf("%s is already held (%s)", label, lockPath)
	}
	pid := os.Getpid()
	if err := file.Truncate(0); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, err
	}
	if _, err := file.WriteString(strconv.Itoa(pid) + "\n"); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, err
	}
	return &gatewayRuntimeLock{file: file, path: lockPath, pid: pid}, nil
}

func (l *gatewayRuntimeLock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = l.file.Truncate(0)
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
	removePIDFileIfOwned(l.path, l.pid)
}

func runGatewayHookDoctor(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("gateway hooks doctor", flag.ExitOnError)
	workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
	path := fs.String("path", "", "base spool path")
	strict := fs.Bool("strict", false, "exit non-zero on warn/error (CI mode)")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway hooks doctor [-workdir dir] [-path file] [-strict]")
	}
	spoolPath := strings.TrimSpace(*path)
	if spoolPath == "" {
		spoolPath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
	}
	hookURL := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_URL"))
	enabled := hookURL != ""
	spoolEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL")), "true")
	secret := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SECRET"))
	verbose := strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_VERBOSE")), "true")
	delivery := strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_DELIVERY")), "true")
	timeout := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS", 4)
	retries := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_RETRIES", 2)
	backoff := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_BACKOFF_MS", 250)
	spoolReplay := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS", 10)
	spoolMaxLines := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES", 2000)
	spoolMaxBytes := parseIntEnvWithDefault("AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES", 5<<20)
	issues := make([]map[string]any, 0, 8)
	nextActions := make([]string, 0, 8)
	addIssue := func(level, key, detail string) {
		issues = append(issues, map[string]any{"level": level, "key": key, "detail": detail})
	}
	if !enabled {
		addIssue("warn", "AGENT_GATEWAY_HOOK_URL", "hooks disabled (url not set)")
		nextActions = append(nextActions, "设置 `AGENT_GATEWAY_HOOK_URL` 后可启用 webhook hooks。")
	}
	if enabled && timeout <= 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS", "must be > 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS` 调整为大于 0 的值。")
	}
	if enabled && retries < 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_RETRIES", "must be >= 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_RETRIES` 调整为大于等于 0。")
	}
	if enabled && backoff < 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_BACKOFF_MS", "must be >= 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_BACKOFF_MS` 调整为大于等于 0。")
	}
	if spoolEnabled && !enabled {
		addIssue("warn", "AGENT_GATEWAY_HOOK_SPOOL", "spool enabled but hook url missing")
		nextActions = append(nextActions, "若保留 spool，请先配置 `AGENT_GATEWAY_HOOK_URL`。")
	}
	if spoolEnabled && spoolReplay <= 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS", "must be > 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS` 调整为大于 0。")
	}
	if spoolEnabled && spoolMaxLines <= 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES", "must be > 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES` 调整为大于 0。")
	}
	if spoolEnabled && spoolMaxBytes < 0 {
		addIssue("error", "AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES", "must be >= 0")
		nextActions = append(nextActions, "将 `AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES` 调整为大于等于 0。")
	}
	if info, err := os.Stat(spoolPath); err == nil {
		if info.Size() > int64(20<<20) {
			addIssue("warn", "spool_size", "spool exceeds 20MB; consider replay/compact")
			nextActions = append(nextActions, "执行 `agentd gateway hooks spool replay -all` 与 `agentd gateway hooks spool compact -all` 清理积压。")
		}
	}
	status := "ok"
	for _, it := range issues {
		if it["level"] == "error" {
			status = "error"
			break
		}
		if it["level"] == "warn" && status == "ok" {
			status = "warn"
		}
	}
	printJSON(map[string]any{
		"status": status,
		"env": map[string]any{
			"hook_enabled":      enabled,
			"hook_url_set":      enabled,
			"has_secret":        secret != "",
			"verbose":           verbose,
			"delivery_events":   delivery,
			"timeout_seconds":   timeout,
			"retries":           retries,
			"backoff_ms":        backoff,
			"spool_enabled":     spoolEnabled,
			"spool_path":        spoolPath,
			"spool_replay_secs": spoolReplay,
			"spool_max_lines":   spoolMaxLines,
			"spool_max_bytes":   spoolMaxBytes,
		},
		"issues":       issues,
		"next_actions": nextActions,
	})
	if *strict && status != "ok" {
		os.Exit(1)
	}
}

func runGatewayHookPing(_ config.Config, args []string) {
	fs := flag.NewFlagSet("gateway hooks ping", flag.ExitOnError)
	urlFlag := fs.String("url", "", "hook URL (defaults to env AGENT_GATEWAY_HOOK_URL)")
	secret := fs.String("secret", "", "optional secret for signature (defaults to env AGENT_GATEWAY_HOOK_SECRET)")
	timeoutSeconds := fs.Int("timeout", 4, "timeout seconds")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway hooks ping [-url hook] [-secret s] [-timeout S]")
	}
	hookURL := strings.TrimSpace(*urlFlag)
	if hookURL == "" {
		hookURL = strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_URL"))
	}
	if hookURL == "" {
		log.Fatal("hook url required (set env AGENT_GATEWAY_HOOK_URL or pass -url)")
	}
	hookSecret := strings.TrimSpace(*secret)
	if hookSecret == "" {
		hookSecret = strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SECRET"))
	}
	if *timeoutSeconds <= 0 {
		*timeoutSeconds = 4
	}
	env := map[string]any{"hook_url": hookURL, "has_secret": hookSecret != "", "timeout_seconds": *timeoutSeconds}
	payload := map[string]any{
		"id":   uuid.NewString(),
		"type": "gateway.ping",
		"at":   time.Now().Format(time.RFC3339Nano),
		"data": map[string]any{"ok": true},
	}
	bs, _ := json.Marshal(payload)
	client := &http.Client{Timeout: time.Duration(*timeoutSeconds) * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSeconds)*time.Second)
	defer cancel()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hookURL, bytes.NewReader(bs))
	if err != nil {
		printJSON(map[string]any{"success": false, "error": err.Error(), "env": env})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Event", "gateway.ping")
	req.Header.Set("X-Agent-Event-Id", payload["id"].(string))
	req.Header.Set("X-Agent-Timestamp", ts)
	if hookSecret != "" {
		req.Header.Set("X-Agent-Signature", signHook(hookSecret, ts, bs))
	}
	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		printJSON(map[string]any{"success": false, "error": err.Error(), "env": env})
		return
	}
	printJSON(map[string]any{"success": resp.StatusCode >= 200 && resp.StatusCode < 300, "status": resp.StatusCode, "env": env})
}

func runGatewayHookSpool(cfg config.Config, args []string) {
	if len(args) == 0 {
		printGatewayHookSpoolUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("gateway hooks spool list", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path (default <workdir>/.agent-daemon/gateway_hooks_spool.jsonl)")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool list [-workdir dir] [-path file]")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		files := listSpoolFiles(basePath)
		printJSON(map[string]any{"base": basePath, "files": files, "count": len(files)})
	case "status":
		fs := flag.NewFlagSet("gateway hooks spool status", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "override spool path")
		all := fs.Bool("all", false, "aggregate status across base + rotated spool files")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool status [-workdir dir] [-path file] [-all]")
		}
		spoolPath := strings.TrimSpace(*path)
		if spoolPath == "" {
			spoolPath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{spoolPath}
		if *all {
			paths = listSpoolFiles(spoolPath)
		}
		stats := spoolStats(paths)
		printJSON(stats)
	case "stats":
		fs := flag.NewFlagSet("gateway hooks spool stats", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path")
		all := fs.Bool("all", false, "aggregate across base + rotated files")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool stats [-workdir dir] [-path file] [-all]")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{basePath}
		if *all {
			paths = listSpoolFiles(basePath)
		}
		stats := spoolStats(paths)
		printJSON(stats)
	case "clear":
		fs := flag.NewFlagSet("gateway hooks spool clear", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "override spool path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool clear [-workdir dir] [-path file]")
		}
		spoolPath := strings.TrimSpace(*path)
		if spoolPath == "" {
			spoolPath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		_ = os.WriteFile(spoolPath, []byte(""), 0o644)
		printJSON(map[string]any{"cleared": true, "path": spoolPath})
	case "replay":
		fs := flag.NewFlagSet("gateway hooks spool replay", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "override spool path")
		urlFlag := fs.String("url", "", "override hook URL (defaults to env AGENT_GATEWAY_HOOK_URL)")
		secret := fs.String("secret", "", "optional hook secret for signing (defaults to env AGENT_GATEWAY_HOOK_SECRET)")
		all := fs.Bool("all", false, "replay base spool and rotated spool files")
		typeFilter := fs.String("type", "", "optional event type filter (e.g. gateway.delivery.media)")
		idFilter := fs.String("id", "", "optional event id filter")
		limit := fs.Int("limit", 200, "max events to attempt in this run")
		timeoutSeconds := fs.Int("timeout", 4, "per-request timeout seconds")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool replay [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-url hook] [-secret s] [-limit N] [-timeout S]")
		}
		spoolPath := strings.TrimSpace(*path)
		if spoolPath == "" {
			spoolPath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		hookURL := strings.TrimSpace(*urlFlag)
		if hookURL == "" {
			hookURL = strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_URL"))
		}
		if hookURL == "" {
			log.Fatal("hook url required (set env AGENT_GATEWAY_HOOK_URL or pass -url)")
		}
		hookSecret := strings.TrimSpace(*secret)
		if hookSecret == "" {
			hookSecret = strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SECRET"))
		}
		if *timeoutSeconds <= 0 {
			*timeoutSeconds = 4
		}
		if *limit <= 0 {
			*limit = 200
		}
		totalSent := 0
		totalRemaining := 0
		var paths []string
		if *all {
			paths = listSpoolFiles(spoolPath)
		} else {
			paths = []string{spoolPath}
		}
		for _, p := range paths {
			if totalSent >= *limit {
				break
			}
			sent, remaining, err := replaySpoolOnce(p, hookURL, hookSecret, *timeoutSeconds, *limit-totalSent, strings.TrimSpace(*typeFilter), strings.TrimSpace(*idFilter))
			totalSent += sent
			totalRemaining += remaining
			if err != nil {
				printJSON(map[string]any{"success": false, "error": err.Error(), "path": p, "sent": totalSent, "remaining": totalRemaining})
				return
			}
		}
		printJSON(map[string]any{
			"success":   true,
			"paths":     paths,
			"sent":      totalSent,
			"remaining": totalRemaining,
			"filter": map[string]any{
				"type": strings.TrimSpace(*typeFilter),
				"id":   strings.TrimSpace(*idFilter),
			},
		})
	case "export":
		fs := flag.NewFlagSet("gateway hooks spool export", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path")
		out := fs.String("out", "", "output file path (required)")
		all := fs.Bool("all", false, "include rotated spool files")
		typeFilter := fs.String("type", "", "optional event type filter")
		idFilter := fs.String("id", "", "optional event id filter")
		before := fs.String("before", "", "optional RFC3339/RFC3339Nano cutoff (created_at < before)")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool export -out file [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-before ts]")
		}
		if strings.TrimSpace(*out) == "" {
			log.Fatal("out is required")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{basePath}
		if *all {
			paths = listSpoolFiles(basePath)
		}
		cutoff, err := parseCutoff(strings.TrimSpace(*before))
		if err != nil {
			log.Fatal(err)
		}
		lines, matched, err := collectSpoolLines(paths, strings.TrimSpace(*typeFilter), strings.TrimSpace(*idFilter), cutoff)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(strings.TrimSpace(*out)), 0o755); err != nil {
			log.Fatal(err)
		}
		data := strings.Join(lines, "\n")
		if len(lines) > 0 && !strings.HasSuffix(data, "\n") {
			data += "\n"
		}
		if err := os.WriteFile(strings.TrimSpace(*out), []byte(data), 0o644); err != nil {
			log.Fatal(err)
		}
		printJSON(map[string]any{
			"success": true,
			"out":     strings.TrimSpace(*out),
			"paths":   paths,
			"matched": matched,
			"filter": map[string]any{
				"type":   strings.TrimSpace(*typeFilter),
				"id":     strings.TrimSpace(*idFilter),
				"before": strings.TrimSpace(*before),
			},
		})
	case "prune":
		fs := flag.NewFlagSet("gateway hooks spool prune", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path")
		all := fs.Bool("all", false, "prune rotated spool files too")
		typeFilter := fs.String("type", "", "optional event type filter")
		idFilter := fs.String("id", "", "optional event id filter")
		before := fs.String("before", "", "optional RFC3339/RFC3339Nano cutoff (created_at < before)")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool prune [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-before ts]")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{basePath}
		if *all {
			paths = listSpoolFiles(basePath)
		}
		cutoff, err := parseCutoff(strings.TrimSpace(*before))
		if err != nil {
			log.Fatal(err)
		}
		totalRemoved := 0
		totalRemain := 0
		perFile := make([]map[string]any, 0, len(paths))
		for _, p := range paths {
			removed, remain, err := pruneSpoolFile(p, strings.TrimSpace(*typeFilter), strings.TrimSpace(*idFilter), cutoff)
			if err != nil {
				log.Fatal(err)
			}
			totalRemoved += removed
			totalRemain += remain
			perFile = append(perFile, map[string]any{"path": p, "removed": removed, "remaining": remain})
		}
		printJSON(map[string]any{
			"success":         true,
			"paths":           paths,
			"removed_total":   totalRemoved,
			"remaining_total": totalRemain,
			"files":           perFile,
			"filter": map[string]any{
				"type":   strings.TrimSpace(*typeFilter),
				"id":     strings.TrimSpace(*idFilter),
				"before": strings.TrimSpace(*before),
			},
		})
	case "compact":
		fs := flag.NewFlagSet("gateway hooks spool compact", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path")
		all := fs.Bool("all", false, "compact rotated spool files too")
		maxLines := fs.Int("max-lines", 2000, "max lines kept per file after compact")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool compact [-workdir dir] [-path file] [-all] [-max-lines N]")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{basePath}
		if *all {
			paths = listSpoolFiles(basePath)
		}
		if *maxLines <= 0 {
			*maxLines = 2000
		}
		perFile := make([]map[string]any, 0, len(paths))
		totalBefore, totalAfter := 0, 0
		for _, p := range paths {
			beforeCount, afterCount, err := compactSpoolFile(p, *maxLines)
			if err != nil {
				log.Fatal(err)
			}
			totalBefore += beforeCount
			totalAfter += afterCount
			perFile = append(perFile, map[string]any{
				"path":    p,
				"before":  beforeCount,
				"after":   afterCount,
				"removed": beforeCount - afterCount,
			})
		}
		printJSON(map[string]any{
			"success":       true,
			"paths":         paths,
			"before_total":  totalBefore,
			"after_total":   totalAfter,
			"removed_total": totalBefore - totalAfter,
			"max_lines":     *maxLines,
			"files":         perFile,
		})
	case "verify":
		fs := flag.NewFlagSet("gateway hooks spool verify", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "base spool path")
		all := fs.Bool("all", false, "verify rotated spool files too")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool verify [-workdir dir] [-path file] [-all]")
		}
		basePath := strings.TrimSpace(*path)
		if basePath == "" {
			basePath = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		paths := []string{basePath}
		if *all {
			paths = listSpoolFiles(basePath)
		}
		files := make([]map[string]any, 0, len(paths))
		totalLines, totalValid, totalInvalid := 0, 0, 0
		for _, p := range paths {
			lines, valid, invalid, samples, err := verifySpoolFile(p)
			if err != nil {
				files = append(files, map[string]any{"path": p, "error": err.Error()})
				continue
			}
			totalLines += lines
			totalValid += valid
			totalInvalid += invalid
			files = append(files, map[string]any{
				"path":            p,
				"lines":           lines,
				"valid":           valid,
				"invalid":         invalid,
				"invalid_samples": samples,
			})
		}
		status := "ok"
		if totalInvalid > 0 {
			status = "warn"
		}
		printJSON(map[string]any{
			"status":        status,
			"paths":         paths,
			"lines_total":   totalLines,
			"valid_total":   totalValid,
			"invalid_total": totalInvalid,
			"files":         files,
		})
	case "import":
		fs := flag.NewFlagSet("gateway hooks spool import", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		path := fs.String("path", "", "target base spool path")
		in := fs.String("in", "", "input JSONL file path (required)")
		all := fs.Bool("all", false, "import all JSONL files under input directory matching spool-like names")
		typeFilter := fs.String("type", "", "optional event type filter")
		idFilter := fs.String("id", "", "optional event id filter")
		before := fs.String("before", "", "optional RFC3339/RFC3339Nano cutoff (created_at < before)")
		appendMode := fs.Bool("append", true, "append into target (default true). set false to overwrite target before import")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway hooks spool import -in file [-all] [-type t] [-id eid] [-before ts] [-workdir dir] [-path file] [-append=true|false]")
		}
		if strings.TrimSpace(*in) == "" {
			log.Fatal("in is required")
		}
		target := strings.TrimSpace(*path)
		if target == "" {
			target = filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_hooks_spool.jsonl")
		}
		inputs := []string{strings.TrimSpace(*in)}
		if *all {
			allInputs, err := listImportFiles(strings.TrimSpace(*in))
			if err != nil {
				log.Fatal(err)
			}
			inputs = allInputs
		}
		cutoff, err := parseCutoff(strings.TrimSpace(*before))
		if err != nil {
			log.Fatal(err)
		}
		totalImported, totalSkipped, totalLines := 0, 0, 0
		perFile := make([]map[string]any, 0, len(inputs))
		first := true
		for _, inputPath := range inputs {
			appendThis := *appendMode
			if !*appendMode && !first {
				appendThis = true
			}
			imported, skipped, total, err := importSpoolFile(inputPath, target, appendThis, strings.TrimSpace(*typeFilter), strings.TrimSpace(*idFilter), cutoff)
			if err != nil {
				log.Fatal(err)
			}
			first = false
			totalImported += imported
			totalSkipped += skipped
			totalLines += total
			perFile = append(perFile, map[string]any{
				"input":    inputPath,
				"total":    total,
				"imported": imported,
				"skipped":  skipped,
			})
		}
		printJSON(map[string]any{
			"success":  true,
			"input":    strings.TrimSpace(*in),
			"inputs":   inputs,
			"target":   target,
			"total":    totalLines,
			"imported": totalImported,
			"skipped":  totalSkipped,
			"append":   *appendMode,
			"all":      *all,
			"files":    perFile,
			"filter": map[string]any{
				"type":   strings.TrimSpace(*typeFilter),
				"id":     strings.TrimSpace(*idFilter),
				"before": strings.TrimSpace(*before),
			},
		})
	default:
		printGatewayHookSpoolUsage()
		os.Exit(2)
	}
}

func runGatewayPairs(cfg config.Config, args []string) {
	if len(args) == 0 {
		printGatewayPairsUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("gateway pairs list", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir (contains .agent-daemon/gateway_pairs.json)")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway pairs list [-workdir dir] [-json]")
		}
		pairsPath := filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_pairs.json")
		pairs := map[string][]string{}
		bs, err := os.ReadFile(pairsPath)
		if err == nil && len(bs) > 0 {
			_ = json.Unmarshal(bs, &pairs)
		}
		if *jsonOutput {
			printJSON(map[string]any{"path": pairsPath, "pairs": pairs})
			return
		}
		fmt.Println("path=" + pairsPath)
		platforms := make([]string, 0, len(pairs))
		for p := range pairs {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)
		for _, p := range platforms {
			ids := pairs[p]
			sort.Strings(ids)
			fmt.Printf("%s=%s\n", p, strings.Join(ids, ","))
		}
	case "revoke":
		fs := flag.NewFlagSet("gateway pairs revoke", flag.ExitOnError)
		workdir := fs.String("workdir", cfg.Workdir, "agent workdir")
		platformName := fs.String("platform", "", "platform name (telegram/discord/slack/yuanbao)")
		userID := fs.String("user", "", "user id to revoke")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway pairs revoke -platform <p> -user <id> [-workdir dir]")
		}
		if strings.TrimSpace(*platformName) == "" || strings.TrimSpace(*userID) == "" {
			log.Fatal("platform and user are required")
		}
		pairsPath := filepath.Join(strings.TrimSpace(*workdir), ".agent-daemon", "gateway_pairs.json")
		pairs := map[string][]string{}
		bs, err := os.ReadFile(pairsPath)
		if err == nil && len(bs) > 0 {
			_ = json.Unmarshal(bs, &pairs)
		}
		p := strings.ToLower(strings.TrimSpace(*platformName))
		uid := strings.TrimSpace(*userID)
		ids := pairs[p]
		out := make([]string, 0, len(ids))
		removed := false
		for _, id := range ids {
			if strings.TrimSpace(id) == uid {
				removed = true
				continue
			}
			out = append(out, id)
		}
		if removed {
			if len(out) == 0 {
				delete(pairs, p)
			} else {
				pairs[p] = out
			}
			bs, _ := json.MarshalIndent(pairs, "", "  ")
			if err := os.MkdirAll(filepath.Dir(pairsPath), 0o755); err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(pairsPath, bs, 0o644); err != nil {
				log.Fatal(err)
			}
			fmt.Println("revoked=true")
		} else {
			fmt.Println("revoked=false")
		}
	default:
		printGatewayPairsUsage()
		os.Exit(2)
	}
}

func parseGatewayConfigPath(args []string, name string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatalf("usage: agentd %s [-file path]", name)
	}
	return *path
}

func runGatewaySetup(args []string) {
	fs := flag.NewFlagSet("gateway setup", flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	platformName := fs.String("platform", "", "platform name (telegram/discord/slack/yuanbao)")
	token := fs.String("token", "", "shared token field (telegram/discord/yuanbao)")
	botToken := fs.String("bot-token", "", "slack bot token")
	appToken := fs.String("app-token", "", "slack app token")
	appID := fs.String("app-id", "", "yuanbao app id")
	appSecret := fs.String("app-secret", "", "yuanbao app secret")
	allowedUsers := fs.String("allowed-users", "", "comma-separated allowed users")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway setup -platform <telegram|discord|slack|yuanbao> [platform flags] [-allowed-users ids] [-file path] [-json]")
	}
	platformKey := strings.ToLower(strings.TrimSpace(*platformName))
	if platformKey == "" {
		log.Fatal("platform is required")
	}
	targetPath := config.ConfigFilePath(*path)
	written, err := setupGatewayConfig(targetPath, platformKey, strings.TrimSpace(*token), strings.TrimSpace(*botToken), strings.TrimSpace(*appToken), strings.TrimSpace(*appID), strings.TrimSpace(*appSecret), strings.TrimSpace(*allowedUsers))
	if err != nil {
		log.Fatal(err)
	}
	if *jsonOutput {
		printJSON(map[string]any{
			"success":  true,
			"path":     targetPath,
			"platform": platformKey,
			"enabled":  true,
			"written":  written,
		})
		return
	}
	fmt.Printf("configured gateway platform %s in %s\n", platformKey, targetPath)
	fmt.Printf("written=%s\n", strings.Join(written, ","))
}

func setupGatewayConfig(path, platformKey, token, botToken, appToken, appID, appSecret, allowedUsers string) ([]string, error) {
	values := map[string]string{
		"gateway.enabled": "true",
	}
	written := []string{"gateway.enabled"}
	switch platformKey {
	case "telegram":
		if token == "" {
			return nil, fmt.Errorf("telegram setup requires -token")
		}
		values["gateway.telegram.bot_token"] = token
		written = append(written, "gateway.telegram.bot_token")
		if allowedUsers != "" {
			values["gateway.telegram.allowed_users"] = allowedUsers
			written = append(written, "gateway.telegram.allowed_users")
		}
	case "discord":
		if token == "" {
			return nil, fmt.Errorf("discord setup requires -token")
		}
		values["gateway.discord.bot_token"] = token
		written = append(written, "gateway.discord.bot_token")
		if allowedUsers != "" {
			values["gateway.discord.allowed_users"] = allowedUsers
			written = append(written, "gateway.discord.allowed_users")
		}
	case "slack":
		if botToken == "" || appToken == "" {
			return nil, fmt.Errorf("slack setup requires -bot-token and -app-token")
		}
		values["gateway.slack.bot_token"] = botToken
		values["gateway.slack.app_token"] = appToken
		written = append(written, "gateway.slack.bot_token", "gateway.slack.app_token")
		if allowedUsers != "" {
			values["gateway.slack.allowed_users"] = allowedUsers
			written = append(written, "gateway.slack.allowed_users")
		}
	case "yuanbao":
		if token == "" && (appID == "" || appSecret == "") {
			return nil, fmt.Errorf("yuanbao setup requires -token or both -app-id and -app-secret")
		}
		if token != "" {
			values["gateway.yuanbao.token"] = token
			written = append(written, "gateway.yuanbao.token")
		}
		if appID != "" {
			values["gateway.yuanbao.app_id"] = appID
			written = append(written, "gateway.yuanbao.app_id")
		}
		if appSecret != "" {
			values["gateway.yuanbao.app_secret"] = appSecret
			written = append(written, "gateway.yuanbao.app_secret")
		}
		if allowedUsers != "" {
			values["gateway.yuanbao.allowed_users"] = allowedUsers
			written = append(written, "gateway.yuanbao.allowed_users")
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platformKey)
	}
	for _, key := range written {
		if err := config.SaveConfigValue(path, key, values[key]); err != nil {
			return nil, err
		}
	}
	return uniqueSortedNames(written), nil
}

func containsName(names []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, name := range names {
		if strings.TrimSpace(name) == target {
			return true
		}
	}
	return false
}

func uniqueSortedNames(names []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func printGatewayUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd gateway run [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd gateway start [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway stop [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway restart [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway install [-file path] [-workdir dir] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway uninstall [-file path] [-workdir dir] [-stop] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway manifest -platform <slack|discord|telegram|yuanbao> [-command /agent] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway status [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway platforms")
	fmt.Fprintln(os.Stderr, "  agentd gateway enable [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd gateway disable [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd gateway setup -platform <telegram|discord|slack|yuanbao> [platform flags] [-allowed-users ids] [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway pairs list [-workdir dir] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway pairs revoke -platform <p> -user <id> [-workdir dir]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool status [-workdir dir] [-path file] [-all]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool clear [-workdir dir] [-path file]")
}

func printGatewayPairsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd gateway pairs list [-workdir dir] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway pairs revoke -platform <p> -user <id> [-workdir dir]")
}

type slackManifestExport struct {
	Platform      string              `json:"platform"`
	Command       string              `json:"command"`
	Commands      []map[string]string `json:"commands"`
	AppManifest   map[string]any      `json:"app_manifest"`
	NextActions   []string            `json:"next_actions"`
	CommandRoutes map[string]string   `json:"command_routes,omitempty"`
}

func runGatewayManifest(args []string) {
	fs := flag.NewFlagSet("gateway manifest", flag.ExitOnError)
	platformName := fs.String("platform", "", "platform name")
	command := fs.String("command", "/agent", "slash command entrypoint for slack")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd gateway manifest -platform <slack|discord|telegram|yuanbao> [-command /agent] [-json]")
	}
	switch strings.ToLower(strings.TrimSpace(*platformName)) {
	case "slack":
		export := buildSlackManifestExport(*command)
		if *jsonOutput {
			printJSON(export)
			return
		}
		printJSON(export)
	case "discord":
		export := buildDiscordManifestExport()
		if *jsonOutput {
			printJSON(export)
			return
		}
		printJSON(export)
	case "telegram":
		export := buildTelegramManifestExport()
		if *jsonOutput {
			printJSON(export)
			return
		}
		printJSON(export)
	case "yuanbao":
		export := buildYuanbaoManifestExport()
		if *jsonOutput {
			printJSON(export)
			return
		}
		printJSON(export)
	default:
		log.Fatal("supported platforms: slack, discord, telegram, yuanbao")
	}
}

func buildSlackManifestExport(command string) slackManifestExport {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "/agent"
	}
	if !strings.HasPrefix(command, "/") {
		command = "/" + command
	}
	commands := []map[string]string{
		{"command": command, "description": "Gateway command entrypoint"},
		{"command": "/pair", "description": "Pair with gateway using code"},
		{"command": "/unpair", "description": "Remove current gateway pairing"},
		{"command": "/cancel", "description": "Cancel the running task"},
		{"command": "/queue", "description": "Show queued task count"},
		{"command": "/status", "description": "Show current session status"},
		{"command": "/pending", "description": "Show latest pending approval"},
		{"command": "/approvals", "description": "Show active approvals"},
		{"command": "/grant", "description": "Grant session or pattern approval"},
		{"command": "/revoke", "description": "Revoke session or pattern approval"},
		{"command": "/approve", "description": "Approve a pending approval id"},
		{"command": "/deny", "description": "Deny a pending approval id"},
		{"command": "/help", "description": "Show supported commands"},
	}
	manifest := map[string]any{
		"display_information": map[string]any{
			"name": "agent-daemon gateway",
		},
		"features": map[string]any{
			"bot_user": map[string]any{
				"display_name":  "agent-daemon",
				"always_online": false,
			},
			"slash_commands": commands,
		},
		"oauth_config": map[string]any{
			"scopes": map[string]any{
				"bot": []string{"app_mentions:read", "channels:history", "chat:write", "commands", "groups:history", "im:history", "mpim:history"},
			},
		},
		"settings": map[string]any{
			"interactivity": map[string]any{
				"is_enabled": true,
			},
			"socket_mode_enabled": true,
		},
	}
	return slackManifestExport{
		Platform:    "slack",
		Command:     command,
		Commands:    commands,
		AppManifest: manifest,
		NextActions: []string{
			"在 Slack app 配置中启用 Socket Mode 与 Interactivity",
			"至少保留一个 slash command；若使用通用入口，可配置 " + command + " 并通过 `/agent status` 等命令转发",
			"把生成的 scopes 同步到 Slack app，并重新安装应用",
		},
		CommandRoutes: map[string]string{
			command + " status":    "/status",
			command + " pending":   "/pending",
			command + " approvals": "/approvals",
			command + " grant 300": "/grant 300",
			command + " revoke":    "/revoke",
		},
	}
}

func buildDiscordManifestExport() map[string]any {
	commands := make([]map[string]any, 0, len(platforms.DiscordApplicationCommands()))
	for _, cmd := range platforms.DiscordApplicationCommands() {
		item := map[string]any{
			"name":        cmd.Name,
			"description": cmd.Description,
			"type":        cmd.Type,
		}
		if len(cmd.Options) > 0 {
			opts := make([]map[string]any, 0, len(cmd.Options))
			for _, opt := range cmd.Options {
				if opt == nil {
					continue
				}
				opts = append(opts, map[string]any{
					"name":        opt.Name,
					"description": opt.Description,
					"type":        opt.Type,
					"required":    opt.Required,
				})
			}
			item["options"] = opts
		}
		commands = append(commands, item)
	}
	return map[string]any{
		"platform": "discord",
		"commands": commands,
		"permissions": []string{
			"applications.commands",
			"bot",
		},
		"bot_permissions": []string{
			"SendMessages",
			"ReadMessageHistory",
			"UseApplicationCommands",
			"AttachFiles",
		},
		"install_url_hint": "https://discord.com/oauth2/authorize?scope=bot%20applications.commands&permissions=379968&client_id=<APP_ID>",
		"next_actions": []string{
			"为 Discord 应用勾选 bot 与 applications.commands scopes",
			"确保 bot 拥有 Send Messages、Read Message History、Use Application Commands、Attach Files 权限",
			"启动 gateway 后会自动 bulk overwrite 注册这些全局 slash 命令",
		},
	}
}

func buildTelegramManifestExport() map[string]any {
	commands := make([]map[string]string, 0, len(platforms.TelegramCommands()))
	botFatherCommands := make([]string, 0, len(platforms.TelegramCommands()))
	for _, cmd := range platforms.TelegramCommands() {
		commands = append(commands, map[string]string{
			"command":     cmd.Command,
			"description": cmd.Description,
		})
		botFatherCommands = append(botFatherCommands, "/"+cmd.Command+" - "+cmd.Description)
	}
	return map[string]any{
		"platform":           "telegram",
		"commands":           commands,
		"set_my_commands":    commands,
		"botfather_commands": botFatherCommands,
		"install_requirements": []string{
			"TELEGRAM_TOKEN",
		},
		"next_actions": []string{
			"在 BotFather 创建 bot 并配置 TELEGRAM_TOKEN",
			"启动 gateway 后会自动调用 setMyCommands 注册这些命令",
			"若需手工核对，可把 botfather_commands 逐行粘贴到 BotFather 命令菜单配置",
		},
	}
}

func buildYuanbaoManifestExport() map[string]any {
	commands := []map[string]string{
		{"command": "/pair", "description": "pair with gateway using code"},
		{"command": "/unpair", "description": "remove current pairing"},
		{"command": "/cancel", "description": "cancel the running task"},
		{"command": "/queue", "description": "show queued task count"},
		{"command": "/status", "description": "show current session status"},
		{"command": "/pending", "description": "show latest pending approval"},
		{"command": "/approvals", "description": "show active approvals"},
		{"command": "/grant [ttl]", "description": "grant session approval"},
		{"command": "/grant pattern <name> [ttl]", "description": "grant pattern approval"},
		{"command": "/revoke", "description": "revoke session approval"},
		{"command": "/revoke pattern <name>", "description": "revoke pattern approval"},
		{"command": "/approve [id]", "description": "approve pending approval"},
		{"command": "/deny [id]", "description": "deny pending approval"},
		{"command": "/help", "description": "show supported commands"},
	}
	quickReplies := []map[string]string{
		{"text": "状态", "route": "/status"},
		{"text": "待审批", "route": "/pending"},
		{"text": "审批", "route": "/approvals"},
		{"text": "批准", "route": "/approve"},
		{"text": "同意", "route": "/approve"},
		{"text": "通过", "route": "/approve"},
		{"text": "拒绝", "route": "/deny"},
		{"text": "驳回", "route": "/deny"},
		{"text": "帮助", "route": "/help"},
	}
	return map[string]any{
		"platform":      "yuanbao",
		"commands":      commands,
		"quick_replies": quickReplies,
		"install_requirements": []string{
			"YUANBAO_TOKEN or (YUANBAO_APP_ID + YUANBAO_APP_SECRET)",
			"YUANBAO_BOT_ID when sign-token response does not include bot_id",
		},
		"optional_env": []string{
			"YUANBAO_API_DOMAIN",
			"YUANBAO_WS_URL",
			"YUANBAO_ROUTE_ENV",
		},
		"next_actions": []string{
			"配置 YUANBAO_TOKEN，或配置 YUANBAO_APP_ID 与 YUANBAO_APP_SECRET 以换取 sign-token",
			"如 sign-token 响应不返回 bot_id，则额外配置 YUANBAO_BOT_ID",
			"聊天侧可直接发送“状态”“待审批”“批准”“拒绝”等快捷回复，Gateway 会归一到现有命令内核",
		},
	}
}

func printGatewayHooksUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool <subcommand>")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks ping [-url hook] [-secret s] [-timeout S]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks doctor [-workdir dir] [-path file] [-strict]")
	fmt.Fprintln(os.Stderr, "  spool subcommands: status, clear, replay")
}

func printGatewayHookSpoolUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool list [-workdir dir] [-path file]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool status [-workdir dir] [-path file] [-all]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool stats [-workdir dir] [-path file] [-all]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool clear [-workdir dir] [-path file]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool replay [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-url hook] [-secret s] [-limit N] [-timeout S]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool export -out file [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-before ts]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool prune [-workdir dir] [-path file] [-all] [-type t] [-id eid] [-before ts]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool compact [-workdir dir] [-path file] [-all] [-max-lines N]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool verify [-workdir dir] [-path file] [-all]")
	fmt.Fprintln(os.Stderr, "  agentd gateway hooks spool import -in file [-all] [-type t] [-id eid] [-before ts] [-workdir dir] [-path file] [-append=true|false]")
}

type gatewayStatusInfo struct {
	Enabled             bool     `json:"enabled"`
	ConfiguredPlatforms []string `json:"configured_platforms"`
	SupportedPlatforms  []string `json:"supported_platforms"`
	Running             bool     `json:"running"`
	PID                 int      `json:"pid,omitempty"`
	PIDPath             string   `json:"pid_path,omitempty"`
	LogPath             string   `json:"log_path,omitempty"`
	Locked              bool     `json:"locked"`
	LockPID             int      `json:"lock_pid,omitempty"`
	LockPath            string   `json:"lock_path,omitempty"`
	TokenLocked         bool     `json:"token_locked"`
	TokenLockPID        int      `json:"token_lock_pid,omitempty"`
	TokenLockPath       string   `json:"token_lock_path,omitempty"`
	StaleLock           bool     `json:"stale_lock,omitempty"`
	StaleTokenLock      bool     `json:"stale_token_lock,omitempty"`
	Installed           bool     `json:"installed"`
	InstallDir          string   `json:"install_dir,omitempty"`
	ManifestPath        string   `json:"manifest_path,omitempty"`
}

func gatewayStatus(cfg config.Config) gatewayStatusInfo {
	running, pid := gatewayProcessStatus(cfg)
	lockState := readGatewayLockState(gatewayLockPath(cfg))
	tokenLockPath := gatewayTokenLockPath(cfg)
	tokenLockState := readGatewayLockState(tokenLockPath)
	return gatewayStatusInfo{
		Enabled:             cfg.GatewayEnabled,
		ConfiguredPlatforms: configuredGatewayPlatforms(cfg),
		SupportedPlatforms:  supportedGatewayPlatforms(),
		Running:             running,
		PID:                 pid,
		PIDPath:             gatewayPIDPath(cfg),
		LogPath:             gatewayLogPath(cfg),
		Locked:              lockState.Alive,
		LockPID:             lockState.PID,
		LockPath:            gatewayLockPath(cfg),
		TokenLocked:         tokenLockState.Alive,
		TokenLockPID:        tokenLockState.PID,
		TokenLockPath:       tokenLockPath,
		StaleLock:           lockState.Stale,
		StaleTokenLock:      tokenLockState.Stale,
		Installed:           fileExists(gatewayManifestPath(cfg)),
		InstallDir:          gatewayInstallDir(cfg),
		ManifestPath:        gatewayManifestPath(cfg),
	}
}

type gatewayLockState struct {
	PID   int
	Alive bool
	Stale bool
}

func readGatewayLockState(path string) gatewayLockState {
	pid := readGatewayLockPID(path)
	if pid <= 0 {
		return gatewayLockState{}
	}
	if processAlive(pid) {
		return gatewayLockState{PID: pid, Alive: true}
	}
	return gatewayLockState{PID: pid, Alive: false, Stale: true}
}

func cleanupStaleGatewayLock(path string) bool {
	st := readGatewayLockState(path)
	if !st.Stale {
		return false
	}
	_ = os.Remove(path)
	return true
}

func gatewayPIDPath(cfg config.Config) string {
	return filepath.Join(strings.TrimSpace(cfg.Workdir), ".agent-daemon", "gateway.pid")
}

func gatewayLogPath(cfg config.Config) string {
	return filepath.Join(strings.TrimSpace(cfg.Workdir), ".agent-daemon", "gateway.log")
}

func gatewayLockPath(cfg config.Config) string {
	return filepath.Join(strings.TrimSpace(cfg.Workdir), ".agent-daemon", "gateway.lock")
}

func gatewayTokenLockPath(cfg config.Config) string {
	fingerprint := gatewayTokenFingerprint(cfg)
	if fingerprint == "" {
		return ""
	}
	return filepath.Join(os.TempDir(), "agent-daemon-gateway-locks", fingerprint+".lock")
}

func gatewayTokenFingerprint(cfg config.Config) string {
	parts := make([]string, 0, 4)
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		parts = append(parts, "telegram:"+strings.TrimSpace(cfg.TelegramToken))
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		parts = append(parts, "discord:"+strings.TrimSpace(cfg.DiscordToken))
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" || strings.TrimSpace(cfg.SlackAppToken) != "" {
		parts = append(parts, "slack:"+strings.TrimSpace(cfg.SlackBotToken)+":"+strings.TrimSpace(cfg.SlackAppToken))
	}
	if strings.TrimSpace(cfg.YuanbaoToken) != "" || strings.TrimSpace(cfg.YuanbaoAppID) != "" {
		parts = append(parts, "yuanbao:"+strings.TrimSpace(cfg.YuanbaoToken)+":"+strings.TrimSpace(cfg.YuanbaoAppID))
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:16])
}

func gatewayInstallDir(cfg config.Config) string {
	return filepath.Join(strings.TrimSpace(cfg.Workdir), ".agent-daemon", "bin")
}

func gatewayManifestPath(cfg config.Config) string {
	return filepath.Join(gatewayInstallDir(cfg), "gateway-install.json")
}

func gatewayProcessStatus(cfg config.Config) (bool, int) {
	pidPath := gatewayPIDPath(cfg)
	bs, err := os.ReadFile(pidPath)
	if err != nil {
		return false, 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(bs)))
	if err != nil || pid <= 0 {
		return false, 0
	}
	if !processAlive(pid) {
		return false, 0
	}
	return true, pid
}

func readGatewayLockPID(path string) int {
	bs, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(bs)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func removePIDFileIfOwned(path string, pid int) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return
	}
	currentPID, err := strconv.Atoi(strings.TrimSpace(string(bs)))
	if err != nil || currentPID != pid {
		return
	}
	_ = os.Remove(path)
}

func gatewayAdapterNames(adapters []gateway.PlatformAdapter) []string {
	names := make([]string, 0, len(adapters))
	for _, adapter := range adapters {
		names = append(names, adapter.Name())
	}
	sort.Strings(names)
	return names
}

func supportedGatewayPlatforms() []string {
	return []string{"telegram", "discord", "slack", "yuanbao"}
}

func configuredGatewayPlatforms(cfg config.Config) []string {
	out := make([]string, 0, 3)
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		out = append(out, "telegram")
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		out = append(out, "discord")
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		out = append(out, "slack")
	}
	if strings.TrimSpace(cfg.YuanbaoToken) != "" || strings.TrimSpace(cfg.YuanbaoAppID) != "" {
		out = append(out, "yuanbao")
	}
	return out
}

func runSessions(cfg config.Config, args []string) {
	if len(args) == 0 {
		printSessionsUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("sessions list", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		limit := fs.Int("limit", 20, "result limit")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd sessions list [-data-dir dir] [-limit N]")
		}
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		rows, err := ss.ListRecentSessions(*limit)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(rows)
	case "search":
		fs := flag.NewFlagSet("sessions search", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		limit := fs.Int("limit", 20, "result limit")
		exclude := fs.String("exclude", "", "exclude session_id")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions search [-data-dir dir] [-limit N] [-exclude session_id] query")
		}
		query := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		rows, err := ss.Search(query, *limit, *exclude)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(rows)
	case "show":
		fs := flag.NewFlagSet("sessions show", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		offset := fs.Int("offset", 0, "message offset (0-based)")
		limit := fs.Int("limit", 200, "message limit")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions show [-data-dir dir] [-offset N] [-limit N] session_id")
		}
		sessionID := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		msgs, err := ss.LoadMessagesPage(sessionID, *offset, *limit)
		if err != nil {
			log.Fatal(err)
		}
		payload := map[string]any{
			"session_id": sessionID,
			"offset":     *offset,
			"limit":      *limit,
			"messages":   msgs,
		}
		printJSON(payload)
	case "stats":
		fs := flag.NewFlagSet("sessions stats", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions stats [-data-dir dir] session_id")
		}
		sessionID := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		stats, err := ss.SessionStats(sessionID)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(stats)
	default:
		printSessionsUsage()
		os.Exit(2)
	}
}

func printSessionsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd sessions list [-data-dir dir] [-limit N]")
	fmt.Fprintln(os.Stderr, "  agentd sessions search [-data-dir dir] [-limit N] [-exclude session_id] query")
	fmt.Fprintln(os.Stderr, "  agentd sessions show [-data-dir dir] [-offset N] [-limit N] session_id")
	fmt.Fprintln(os.Stderr, "  agentd sessions stats [-data-dir dir] session_id")
}

func supportedModelProviders() []string {
	return []string{"openai", "anthropic", "codex"}
}

func availableModelProviders(cfg config.Config) []string {
	out := append([]string{}, supportedModelProviders()...)
	items, err := loadConfiguredPlugins(cfg)
	if err != nil {
		return out
	}
	out = append(out, plugins.ProviderNames(items)...)
	return uniqueSortedNames(out)
}

func isProviderAvailable(cfg config.Config, provider string) bool {
	return containsName(availableModelProviders(cfg), strings.ToLower(strings.TrimSpace(provider)))
}

func currentModelConfig(cfg config.Config, provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return cfg.AnthropicModel, cfg.AnthropicBaseURL
	case "codex":
		return cfg.CodexModel, cfg.CodexBaseURL
	default:
		return cfg.ModelName, cfg.ModelBaseURL
	}
}

func parseModelSetArgs(args []string) (string, string, error) {
	if len(args) == 1 {
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("usage: agentd model set provider model or provider:model")
		}
		return normalizeModelSelection(parts[0], parts[1])
	}
	if len(args) == 2 {
		return normalizeModelSelection(args[0], args[1])
	}
	return "", "", fmt.Errorf("usage: agentd model set provider model or provider:model")
}

func normalizeModelSelection(provider, modelName string) (string, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", "", fmt.Errorf("model is required")
	}
	if provider == "" {
		return "", "", fmt.Errorf("provider is required")
	}
	for _, supported := range supportedModelProviders() {
		if provider == supported {
			return provider, modelName, nil
		}
	}
	// Plugin providers are allowed at config level; availability is validated at runtime.
	return provider, modelName, nil
}

func saveModelSelection(path, provider, modelName, baseURL string) error {
	provider, modelName, err := normalizeModelSelection(provider, modelName)
	if err != nil {
		return err
	}
	if err := config.SaveConfigValue(path, "api.type", provider); err != nil {
		return err
	}
	modelKey, baseURLKey := modelConfigKeys(provider)
	if err := config.SaveConfigValue(path, modelKey, modelName); err != nil {
		return err
	}
	if strings.TrimSpace(baseURL) != "" {
		if err := config.SaveConfigValue(path, baseURLKey, strings.TrimSpace(baseURL)); err != nil {
			return err
		}
	}
	return nil
}

func modelConfigKeys(provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "api.anthropic.model", "api.anthropic.base_url"
	case "codex":
		return "api.codex.model", "api.codex.base_url"
	default:
		return "api.model", "api.base_url"
	}
}

func providerAPIKeyConfigKey(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "api.anthropic.api_key"
	case "codex":
		return "api.codex.api_key"
	case "openai":
		return "api.api_key"
	default:
		return ""
	}
}

func runChat(cfg config.Config, first string, sessionID ...string) {
	eng, cronStore := mustBuildEngine(cfg)
	id := uuid.NewString()
	skills := ""
	if len(sessionID) > 0 && sessionID[0] != "" {
		id = sessionID[0]
	}
	if len(sessionID) > 1 && sessionID[1] != "" {
		skills = sessionID[1]
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.CronEnabled && cronStore != nil {
		s := &cronrunner.Scheduler{
			Store:          cronStore,
			Engine:         eng,
			Tick:           time.Duration(cfg.CronTickSeconds) * time.Second,
			MaxConcurrency: cfg.CronMaxConcurrency,
		}
		if err := s.Start(ctx); err != nil {
			log.Printf("cron scheduler start failed: %v", err)
		} else {
			log.Printf("cron scheduler enabled (tick=%ds max_concurrency=%d)", cfg.CronTickSeconds, cfg.CronMaxConcurrency)
		}
	}
	if err := cli.RunChat(ctx, eng, id, first, skills); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	if cfg.GatewayEnabled {
		cfg.ModelUseStreaming = true
	}
	eng, cronStore := mustBuildEngine(cfg)
	apiSrv := &api.Server{
		Engine: eng,
		ConfigSnapshotFn: func() map[string]any {
			return map[string]any{
				"model_provider":      cfg.ModelProvider,
				"model_name":          selectedModelName(cfg, cfg.ModelProvider),
				"listen_addr":         cfg.ListenAddr,
				"workdir":             cfg.Workdir,
				"data_dir":            cfg.DataDir,
				"gateway_enabled":     cfg.GatewayEnabled,
				"enabled_toolsets":    cfg.EnabledToolsets,
				"disabled_tools":      cfg.DisabledTools,
				"model_use_streaming": cfg.ModelUseStreaming,
			}
		},
		GatewayStatusFn: func() map[string]any {
			status := gatewayStatus(cfg)
			return map[string]any{
				"enabled":              status.Enabled,
				"configured_platforms": status.ConfiguredPlatforms,
				"supported_platforms":  status.SupportedPlatforms,
				"running":              status.Running,
				"pid":                  status.PID,
				"log_path":             status.LogPath,
				"locked":               status.Locked,
				"token_locked":         status.TokenLocked,
				"installed":            status.Installed,
			}
		},
		ConfigUpdateFn: func(key, value string) (map[string]any, error) {
			path := config.ConfigFilePath("")
			if err := config.SaveConfigValue(path, key, value); err != nil {
				return nil, err
			}
			return map[string]any{
				"success": true,
				"key":     key,
				"value":   value,
				"path":    path,
			}, nil
		},
		GatewayActionFn: func(action string) (map[string]any, error) {
			path := config.ConfigFilePath("")
			switch strings.ToLower(strings.TrimSpace(action)) {
			case "enable":
				if err := config.SaveConfigValue(path, "gateway.enabled", "true"); err != nil {
					return nil, err
				}
				cfg.GatewayEnabled = true
			case "disable":
				if err := config.SaveConfigValue(path, "gateway.enabled", "false"); err != nil {
					return nil, err
				}
				cfg.GatewayEnabled = false
			default:
				return nil, fmt.Errorf("unsupported gateway action: %s", action)
			}
			status := gatewayStatus(cfg)
			return map[string]any{
				"success": true,
				"action":  action,
				"status": map[string]any{
					"enabled":              status.Enabled,
					"configured_platforms": status.ConfiguredPlatforms,
					"supported_platforms":  status.SupportedPlatforms,
					"running":              status.Running,
					"pid":                  status.PID,
					"log_path":             status.LogPath,
				},
			}, nil
		},
	}
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: apiSrv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("agent-daemon listening on %s", cfg.ListenAddr)

	cronCtx, cronCancel := context.WithCancel(context.Background())
	defer cronCancel()
	if cfg.CronEnabled && cronStore != nil {
		s := &cronrunner.Scheduler{
			Store:          cronStore,
			Engine:         eng,
			Tick:           time.Duration(cfg.CronTickSeconds) * time.Second,
			MaxConcurrency: cfg.CronMaxConcurrency,
		}
		if err := s.Start(cronCtx); err != nil {
			log.Printf("cron scheduler start failed: %v", err)
		} else {
			log.Printf("cron scheduler enabled (tick=%ds max_concurrency=%d)", cfg.CronTickSeconds, cfg.CronMaxConcurrency)
		}
	}

	if cfg.GatewayEnabled {
		log.Printf("gateway enabled")
		gatewayCtx, gatewayCancel := context.WithCancel(context.Background())
		defer gatewayCancel()

		adapters := buildGatewayAdapters(cfg)
		if len(adapters) > 0 {
			lock, err := acquireGatewayLocks(cfg)
			if err != nil {
				log.Printf("gateway start skipped: %v", err)
			} else {
				defer lock.Release()
				runner := gateway.NewRunner(adapters, eng, func(platform string) string {
					switch platform {
					case "telegram":
						return cfg.TelegramAllowed
					case "discord":
						return cfg.DiscordAllowed
					case "slack":
						return cfg.SlackAllowed
					case "yuanbao":
						return cfg.YuanbaoAllowed
					}
					return ""
				})
				if err := runner.Start(gatewayCtx); err != nil {
					log.Printf("gateway start failed: %v", err)
				}
			}
		} else {
			log.Printf("gateway enabled but no platform adapters configured")
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			log.Printf("shutting down...")
			gatewayCancel()
			cronCancel()
			_ = srv.Shutdown(context.Background())
		}()
	}

	log.Fatal(srv.ListenAndServe())
}

func buildGatewayAdapters(cfg config.Config) []gateway.PlatformAdapter {
	var adapters []gateway.PlatformAdapter
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		ta, err := platforms.NewTelegramAdapter(cfg.TelegramToken)
		if err != nil {
			log.Printf("telegram adapter: %v", err)
		} else {
			adapters = append(adapters, ta)
			log.Printf("telegram adapter configured")
		}
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		da, err := platforms.NewDiscordAdapter(cfg.DiscordToken)
		if err != nil {
			log.Printf("discord adapter: %v", err)
		} else {
			adapters = append(adapters, da)
			log.Printf("discord adapter configured")
		}
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		sa, err := platforms.NewSlackAdapter(cfg.SlackBotToken, cfg.SlackAppToken)
		if err != nil {
			log.Printf("slack adapter: %v", err)
		} else {
			adapters = append(adapters, sa)
			log.Printf("slack adapter configured")
		}
	}
	if strings.TrimSpace(cfg.YuanbaoToken) != "" || strings.TrimSpace(cfg.YuanbaoAppID) != "" {
		ya, err := platforms.NewYuanbaoAdapterFromEnv()
		if err != nil {
			log.Printf("yuanbao adapter: %v", err)
		} else {
			adapters = append(adapters, ya)
			log.Printf("yuanbao adapter configured")
		}
	}
	return adapters
}

func applyDisabledTools(registry *tools.Registry, disabled string) {
	registry.Disable(parseNameList(disabled)...)
}

func applyEnabledToolsets(registry *tools.Registry, enabled string) error {
	names := parseNameList(enabled)
	if len(names) == 0 {
		return nil
	}
	allowed, err := tools.ResolveToolset(names)
	if err != nil {
		return err
	}
	all := registry.Names()
	disable := make([]string, 0, len(all))
	for _, toolName := range all {
		if _, ok := allowed[toolName]; !ok {
			disable = append(disable, toolName)
		}
	}
	registry.Disable(disable...)
	return nil
}

func mustBuildEngine(cfg config.Config) (*agent.Engine, *store.CronStore) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	sessionStore, err := store.NewSessionStore(filepath.Join(cfg.DataDir, "sessions.db"))
	if err != nil {
		log.Fatal(err)
	}
	cronStore, err := store.NewCronStore(sessionStore.DB())
	if err != nil {
		log.Printf("cron store init failed: %v", err)
		cronStore = nil
	}
	memoryStore, err := memory.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	registry := tools.NewRegistry()
	proc := tools.NewProcessRegistry(filepath.Join(cfg.DataDir, "processes"))
	tools.RegisterBuiltins(registry, proc)
	if cronStore != nil {
		registry.Register(tools.NewCronJobTool(cronStore))
	}
	registry.Register(tools.NewSendMessageTool())
	registry.Register(tools.NewExecuteCodeTool())
	approvalStore := tools.NewPersistentApprovalStore(time.Duration(cfg.ApprovalTTLSeconds)*time.Second, sessionStore)
	switch strings.ToLower(strings.TrimSpace(cfg.MCPTransport)) {
	case "stdio":
		if strings.TrimSpace(cfg.MCPStdioCommand) != "" {
			mcpClient := tools.NewMCPStdioClient(cfg.MCPStdioCommand, time.Duration(cfg.MCPTimeoutSeconds)*time.Second)
			if names, err := tools.RegisterMCPTools(context.Background(), registry, mcpClient); err != nil {
				log.Printf("mcp stdio discovery failed: %v", err)
			} else if len(names) > 0 {
				log.Printf("registered %d mcp tools via stdio command", len(names))
			}
		}
	default:
		if strings.TrimSpace(cfg.MCPEndpoint) != "" {
			mcpClient := tools.NewMCPClient(cfg.MCPEndpoint, time.Duration(cfg.MCPTimeoutSeconds)*time.Second)
			mcpClient.TokenStore = sessionStore
			if strings.TrimSpace(cfg.MCPOAuthTokenURL) != "" {
				grantType := strings.ToLower(strings.TrimSpace(cfg.MCPOAuthGrantType))
				if grantType == "authorization_code" {
					mcpClient.ConfigureOAuthAuthCode(tools.MCPOAuthConfig{
						TokenURL:     cfg.MCPOAuthTokenURL,
						AuthURL:      cfg.MCPOAuthAuthURL,
						RedirectURL:  cfg.MCPOAuthRedirectURL,
						ClientID:     cfg.MCPOAuthClientID,
						ClientSecret: cfg.MCPOAuthClientSecret,
						Scopes:       cfg.MCPOAuthScopes,
					})
					done := make(chan string, 1)
					if err := mcpClient.StartOAuthCallbackServer(cfg.MCPOAuthCallbackPort, done); err != nil {
						log.Printf("mcp oauth callback server failed: %v", err)
					} else {
						authURL := mcpClient.BuildAuthURL("mcp-auth")
						log.Printf("mcp oauth: open this URL to authorize: %s", authURL)
						select {
						case <-done:
							log.Printf("mcp oauth: authorization successful")
						case <-time.After(5 * time.Minute):
							log.Printf("mcp oauth: authorization timed out")
						}
					}
				} else {
					mcpClient.ConfigureOAuthClientCredentials(tools.MCPOAuthConfig{
						TokenURL:     cfg.MCPOAuthTokenURL,
						ClientID:     cfg.MCPOAuthClientID,
						ClientSecret: cfg.MCPOAuthClientSecret,
						Scopes:       cfg.MCPOAuthScopes,
					})
				}
			}
			if names, err := tools.RegisterMCPTools(context.Background(), registry, mcpClient); err != nil {
				log.Printf("mcp discovery failed: %v", err)
			} else if len(names) > 0 {
				log.Printf("registered %d mcp tools from %s", len(names), cfg.MCPEndpoint)
			}
		}
	}
	if err := applyEnabledToolsets(registry, cfg.EnabledToolsets); err != nil {
		log.Printf("enabled_toolsets ignored: %v", err)
	}
	applyDisabledTools(registry, cfg.DisabledTools)
	pluginManifests, err := loadConfiguredPlugins(cfg)
	if err != nil {
		log.Printf("plugin manifests ignored: %v", err)
	} else if n, err := plugins.RegisterToolPlugins(registry, pluginManifests); err != nil {
		log.Printf("plugin tools ignored: %v", err)
	} else if n > 0 {
		log.Printf("registered %d plugin tools", n)
	}
	client := buildModelClient(cfg)
	return &agent.Engine{
		Client:                  client,
		Registry:                registry,
		SessionStore:            sessionStore,
		SearchStore:             sessionStore,
		MemoryStore:             memoryStore,
		TodoStore:               tools.NewTodoStore(),
		ApprovalStore:           approvalStore,
		Workdir:                 cfg.Workdir,
		SystemPrompt:            agent.DefaultSystemPrompt(),
		MaxIterations:           cfg.MaxIterations,
		MaxContextChars:         cfg.MaxContextChars,
		CompressionTailMessages: cfg.CompressionTailMessages,
	}, cronStore
}

func buildModelClient(cfg config.Config) model.Client {
	if strings.TrimSpace(cfg.ModelCascade) != "" {
		return buildCascadeClient(cfg)
	}

	primaryProvider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	primary := buildProviderClient(cfg, primaryProvider)
	if primary == nil {
		log.Printf("primary provider %q unavailable; fallback to openai", primaryProvider)
		primaryProvider = "openai"
		primary = buildProviderClient(cfg, primaryProvider)
	}
	fallbackProvider := strings.ToLower(strings.TrimSpace(cfg.ModelFallbackProvider))
	if fallbackProvider == "" || fallbackProvider == primaryProvider {
		return primary
	}
	fallback := buildProviderClient(cfg, fallbackProvider)
	if fallback == nil {
		return primary
	}

	circuitThreshold := cfg.ModelCircuitThreshold
	circuitRecovery := time.Duration(cfg.ModelCircuitRecoverySec) * time.Second
	circuitHalfOpenMax := cfg.ModelCircuitHalfOpenMax

	if cfg.ModelRaceEnabled {
		log.Printf("model race enabled: primary=%s fallback=%s", primaryProvider, fallbackProvider)
		return model.NewRaceClient(primary, primaryProvider, fallback, fallbackProvider, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
	}

	log.Printf("model fallback enabled: primary=%s fallback=%s", primaryProvider, fallbackProvider)
	return model.NewFallbackClientWithCircuit(primary, primaryProvider, fallback, fallbackProvider, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
}

func buildProviderClient(cfg config.Config, provider string) model.Client {
	switch provider {
	case "anthropic":
		client := model.NewAnthropicClient(cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, cfg.AnthropicModel)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	case "codex":
		client := model.NewCodexClient(cfg.CodexBaseURL, cfg.CodexAPIKey, cfg.CodexModel)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	case "openai", "":
		client := model.NewOpenAIClient(cfg.ModelBaseURL, cfg.ModelAPIKey, cfg.ModelName)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	default:
		items, err := loadConfiguredPlugins(cfg)
		if err != nil {
			log.Printf("provider plugin discovery failed (%s): %v", provider, err)
			return nil
		}
		selectedModel, _ := currentModelConfig(cfg, provider)
		pc, ok, err := plugins.NewProviderClient(provider, selectedModel, items)
		if err != nil {
			log.Printf("provider plugin load failed (%s): %v", provider, err)
			return nil
		}
		if ok {
			return pc
		}
		log.Printf("unknown provider: %s", provider)
		return nil
	}
}

func buildCascadeClient(cfg config.Config) model.Client {
	builder := func(name string) (model.Client, string, error) {
		client := buildProviderClient(cfg, name)
		if client == nil {
			return nil, "", fmt.Errorf("unknown provider: %s", name)
		}
		return client, name, nil
	}
	entries, err := model.ParseCascadeProviders(cfg.ModelCascade, builder)
	if err != nil {
		log.Printf("cascade parse error: %v, falling back to single provider", err)
		return buildProviderClient(cfg, cfg.ModelProvider)
	}
	circuitThreshold := cfg.ModelCircuitThreshold
	circuitRecovery := time.Duration(cfg.ModelCircuitRecoverySec) * time.Second
	circuitHalfOpenMax := cfg.ModelCircuitHalfOpenMax

	if cfg.ModelCostAware {
		log.Printf("model cascade (cost-aware): %d providers", len(entries))
	} else {
		log.Printf("model cascade (ordered): %d providers", len(entries))
	}
	return model.NewCascadeClientWithCircuit(entries, cfg.ModelCostAware, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
}
