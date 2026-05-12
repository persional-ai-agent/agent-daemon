package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	"github.com/dingjingmaster/agent-daemon/internal/store"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
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
	case "gateway":
		runGateway(cfg, os.Args[2:])
	case "sessions":
		runSessions(cfg, os.Args[2:])
	default:
		runChat(cfg, "", uuid.NewString())
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
		for _, provider := range supportedModelProviders() {
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
	fmt.Fprintln(os.Stderr, "  agentd model providers")
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
		checkToolsetsConfig(cfg),
		checkStubTools(cfg),
		checkRegisteredTools(),
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
	// These tools exist as interface-alignment stubs only.
	stubs := []string{
		"vision_analyze",
		"image_generate",
		"text_to_speech",
		"mixture_of_agents",
		"browser_navigate",
		"browser_snapshot",
		"browser_click",
		"browser_type",
		"browser_scroll",
		"browser_back",
		"browser_press",
		"browser_get_images",
		"browser_vision",
		"browser_console",
		"browser_cdp",
		"browser_dialog",
	}
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
	values := map[string]string{
		"gateway.enabled": "true",
	}
	written := []string{"gateway.enabled"}
	switch platformKey {
	case "telegram":
		if strings.TrimSpace(*token) == "" {
			log.Fatal("telegram setup requires -token")
		}
		values["gateway.telegram.bot_token"] = strings.TrimSpace(*token)
		written = append(written, "gateway.telegram.bot_token")
		if strings.TrimSpace(*allowedUsers) != "" {
			values["gateway.telegram.allowed_users"] = strings.TrimSpace(*allowedUsers)
			written = append(written, "gateway.telegram.allowed_users")
		}
	case "discord":
		if strings.TrimSpace(*token) == "" {
			log.Fatal("discord setup requires -token")
		}
		values["gateway.discord.bot_token"] = strings.TrimSpace(*token)
		written = append(written, "gateway.discord.bot_token")
		if strings.TrimSpace(*allowedUsers) != "" {
			values["gateway.discord.allowed_users"] = strings.TrimSpace(*allowedUsers)
			written = append(written, "gateway.discord.allowed_users")
		}
	case "slack":
		if strings.TrimSpace(*botToken) == "" || strings.TrimSpace(*appToken) == "" {
			log.Fatal("slack setup requires -bot-token and -app-token")
		}
		values["gateway.slack.bot_token"] = strings.TrimSpace(*botToken)
		values["gateway.slack.app_token"] = strings.TrimSpace(*appToken)
		written = append(written, "gateway.slack.bot_token", "gateway.slack.app_token")
		if strings.TrimSpace(*allowedUsers) != "" {
			values["gateway.slack.allowed_users"] = strings.TrimSpace(*allowedUsers)
			written = append(written, "gateway.slack.allowed_users")
		}
	case "yuanbao":
		if strings.TrimSpace(*token) == "" && (strings.TrimSpace(*appID) == "" || strings.TrimSpace(*appSecret) == "") {
			log.Fatal("yuanbao setup requires -token or both -app-id and -app-secret")
		}
		if strings.TrimSpace(*token) != "" {
			values["gateway.yuanbao.token"] = strings.TrimSpace(*token)
			written = append(written, "gateway.yuanbao.token")
		}
		if strings.TrimSpace(*appID) != "" {
			values["gateway.yuanbao.app_id"] = strings.TrimSpace(*appID)
			written = append(written, "gateway.yuanbao.app_id")
		}
		if strings.TrimSpace(*appSecret) != "" {
			values["gateway.yuanbao.app_secret"] = strings.TrimSpace(*appSecret)
			written = append(written, "gateway.yuanbao.app_secret")
		}
		if strings.TrimSpace(*allowedUsers) != "" {
			values["gateway.yuanbao.allowed_users"] = strings.TrimSpace(*allowedUsers)
			written = append(written, "gateway.yuanbao.allowed_users")
		}
	default:
		log.Fatalf("unsupported platform: %s", platformKey)
	}
	targetPath := config.ConfigFilePath(*path)
	for _, key := range written {
		if err := config.SaveConfigValue(targetPath, key, values[key]); err != nil {
			log.Fatal(err)
		}
	}
	sort.Strings(written)
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

func printGatewayUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
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
}

func gatewayStatus(cfg config.Config) gatewayStatusInfo {
	return gatewayStatusInfo{
		Enabled:             cfg.GatewayEnabled,
		ConfiguredPlatforms: configuredGatewayPlatforms(cfg),
		SupportedPlatforms:  supportedGatewayPlatforms(),
	}
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
	for _, supported := range supportedModelProviders() {
		if provider == supported {
			return provider, modelName, nil
		}
	}
	return "", "", fmt.Errorf("unsupported provider %q (supported: openai, anthropic, codex)", provider)
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
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: (&api.Server{Engine: eng}).Handler(), ReadHeaderTimeout: 10 * time.Second}
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
		client := model.NewOpenAIClient(cfg.ModelBaseURL, cfg.ModelAPIKey, cfg.ModelName)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
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
