package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/config"
)

type migratePlanItem struct {
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Exists    bool   `json:"exists"`
	Overwrite bool   `json:"overwrite"`
}

func runSetupMigrate(cfg config.Config, args []string) {
	mode := "apply"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		mode = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch mode {
	case "apply":
		runSetupMigrateApply(cfg, args)
	case "rollback":
		runSetupMigrateRollback(cfg, args)
	default:
		log.Fatal("usage: agentd setup migrate [apply|rollback] ...")
	}
}

func runSetupMigrateApply(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("setup migrate apply", flag.ExitOnError)
	from := fs.String("from", "hermes", "source preset: hermes|openclaw|path")
	fromPath := fs.String("from-path", "", "custom source root path")
	preset := fs.String("preset", "minimal", "migration preset: minimal|full")
	overwrite := fs.Bool("overwrite", false, "overwrite existing target files")
	dryRun := fs.Bool("dry-run", true, "preview migration plan only")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd setup migrate apply [-from hermes|openclaw|path] [-from-path dir] [-preset minimal|full] [-overwrite] [-dry-run=true|false] [-json]")
	}
	srcRoot, err := resolveMigrationSource(strings.TrimSpace(*from), strings.TrimSpace(*fromPath))
	if err != nil {
		log.Fatal(err)
	}
	plan, err := buildMigrationPlan(srcRoot, cfg.Workdir, cfg.DataDir, strings.TrimSpace(*preset), *overwrite)
	if err != nil {
		log.Fatal(err)
	}
	result := map[string]any{
		"success":   true,
		"mode":      "dry-run",
		"source":    srcRoot,
		"preset":    strings.TrimSpace(*preset),
		"overwrite": *overwrite,
		"count":     len(plan),
		"plan":      plan,
	}
	if !*dryRun {
		checkpoint, copied, skipped, err := applyMigrationPlan(plan, *overwrite, cfg.DataDir)
		if err != nil {
			log.Fatal(err)
		}
		result["mode"] = "apply"
		result["checkpoint"] = checkpoint
		result["copied"] = copied
		result["skipped"] = skipped
	}
	if *jsonOutput {
		printJSON(result)
		return
	}
	fmt.Printf("mode=%v\n", result["mode"])
	fmt.Printf("source=%v\n", result["source"])
	fmt.Printf("preset=%v\n", result["preset"])
	fmt.Printf("overwrite=%v\n", result["overwrite"])
	fmt.Printf("plan_count=%v\n", result["count"])
	if cp, ok := result["checkpoint"].(string); ok && strings.TrimSpace(cp) != "" {
		fmt.Printf("checkpoint=%s\n", cp)
	}
	if copied, ok := result["copied"].(int); ok {
		fmt.Printf("copied=%d\n", copied)
	}
	if skipped, ok := result["skipped"].(int); ok {
		fmt.Printf("skipped=%d\n", skipped)
	}
}

func runSetupMigrateRollback(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("setup migrate rollback", flag.ExitOnError)
	checkpoint := fs.String("checkpoint", "", "migration checkpoint tar.gz")
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 || strings.TrimSpace(*checkpoint) == "" {
		log.Fatal("usage: agentd setup migrate rollback -checkpoint file.tar.gz [-json]")
	}
	restored, err := restoreMigrationCheckpoint(strings.TrimSpace(*checkpoint))
	if err != nil {
		log.Fatal(err)
	}
	out := map[string]any{
		"success":    true,
		"checkpoint": strings.TrimSpace(*checkpoint),
		"restored":   restored,
	}
	if *jsonOutput {
		printJSON(out)
		return
	}
	fmt.Printf("checkpoint=%s\n", strings.TrimSpace(*checkpoint))
	fmt.Printf("restored=%d\n", restored)
}

func resolveMigrationSource(from, fromPath string) (string, error) {
	if strings.TrimSpace(fromPath) != "" {
		return filepath.Abs(strings.TrimSpace(fromPath))
	}
	home, _ := os.UserHomeDir()
	switch strings.ToLower(strings.TrimSpace(from)) {
	case "hermes":
		return filepath.Join(home, ".hermes"), nil
	case "openclaw":
		return filepath.Join(home, ".openclaw"), nil
	case "path":
		return "", fmt.Errorf("from-path is required when -from path")
	default:
		return "", fmt.Errorf("unsupported migration source: %s", from)
	}
}

func buildMigrationPlan(srcRoot, workdir, dataDir, preset string, overwrite bool) ([]migratePlanItem, error) {
	srcRoot = strings.TrimSpace(srcRoot)
	if srcRoot == "" {
		return nil, fmt.Errorf("migration source required")
	}
	if _, err := os.Stat(srcRoot); err != nil {
		return nil, fmt.Errorf("migration source not found: %s", srcRoot)
	}
	configTarget := config.ConfigFilePath("")
	if strings.TrimSpace(workdir) != "" {
		configTarget = filepath.Join(workdir, "config.ini")
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = filepath.Join(workdir, ".agent-daemon")
	}
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = "minimal"
	}
	plan := make([]migratePlanItem, 0, 8)
	appendIfExists := func(kind, src, dst string) {
		if !fileExists(src) {
			return
		}
		plan = append(plan, migratePlanItem{
			Kind:      kind,
			Source:    src,
			Target:    dst,
			Exists:    fileExists(dst),
			Overwrite: overwrite,
		})
	}
	appendIfExists("config", filepath.Join(srcRoot, "config.ini"), configTarget)
	appendIfExists("memory", filepath.Join(srcRoot, "MEMORY.md"), filepath.Join(dataDir, "MEMORY.md"))
	appendIfExists("user", filepath.Join(srcRoot, "USER.md"), filepath.Join(dataDir, "USER.md"))
	appendIfExists("sessions_db", filepath.Join(srcRoot, "sessions.db"), filepath.Join(dataDir, "sessions.db"))

	legacyData := filepath.Join(srcRoot, ".agent-daemon")
	appendIfExists("memory", filepath.Join(legacyData, "MEMORY.md"), filepath.Join(dataDir, "MEMORY.md"))
	appendIfExists("user", filepath.Join(legacyData, "USER.md"), filepath.Join(dataDir, "USER.md"))
	appendIfExists("sessions_db", filepath.Join(legacyData, "sessions.db"), filepath.Join(dataDir, "sessions.db"))

	if preset == "full" {
		appendIfExists("spool", filepath.Join(legacyData, "gateway-hooks-spool.jsonl"), filepath.Join(dataDir, "gateway-hooks-spool.jsonl"))
		appendIfExists("cron_runs", filepath.Join(legacyData, "cron_runs.db"), filepath.Join(dataDir, "cron_runs.db"))
	}
	if len(plan) == 0 {
		return nil, fmt.Errorf("no migratable files found under %s", srcRoot)
	}
	sort.SliceStable(plan, func(i, j int) bool { return plan[i].Kind < plan[j].Kind })
	return plan, nil
}

func applyMigrationPlan(plan []migratePlanItem, overwrite bool, dataDir string) (checkpoint string, copied int, skipped int, err error) {
	targets := make([]string, 0, len(plan))
	for _, item := range plan {
		targets = append(targets, item.Target)
	}
	checkpoint, err = createMigrationCheckpoint(targets, dataDir)
	if err != nil {
		return "", 0, 0, err
	}
	for _, item := range plan {
		if item.Exists && !overwrite {
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(item.Target), 0o755); err != nil {
			return checkpoint, copied, skipped, err
		}
		if err := copyFile(item.Source, item.Target); err != nil {
			return checkpoint, copied, skipped, err
		}
		copied++
	}
	return checkpoint, copied, skipped, nil
}

func createMigrationCheckpoint(targets []string, dataDir string) (string, error) {
	existing := make([]string, 0, len(targets))
	for _, target := range targets {
		if fileExists(target) {
			existing = append(existing, target)
		}
	}
	if len(existing) == 0 {
		return "", nil
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = "."
	}
	backupDir := filepath.Join(dataDir, "migration-backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(backupDir, fmt.Sprintf("%d-migration-checkpoint.tar.gz", time.Now().UnixNano()))
	outFile, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	gz := gzip.NewWriter(outFile)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	manifest := map[string]any{
		"created_at": time.Now().Format(time.RFC3339Nano),
		"files":      existing,
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeTarBytes(tw, "manifest.json", manifestBytes, 0o644); err != nil {
		return "", err
	}
	for _, target := range existing {
		abs, _ := filepath.Abs(target)
		bs, err := os.ReadFile(abs)
		if err != nil {
			return "", err
		}
		name := "files" + filepath.ToSlash(abs)
		if err := writeTarBytes(tw, name, bs, 0o644); err != nil {
			return "", err
		}
	}
	return outPath, nil
}

func restoreMigrationCheckpoint(checkpoint string) (int, error) {
	f, err := os.Open(checkpoint)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	restored := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return restored, err
		}
		if h.FileInfo().IsDir() || h.Name == "manifest.json" {
			continue
		}
		if !strings.HasPrefix(h.Name, "files/") {
			continue
		}
		target := strings.TrimPrefix(h.Name, "files")
		if strings.TrimSpace(target) == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return restored, err
		}
		out, err := os.Create(target)
		if err != nil {
			return restored, err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return restored, err
		}
		_ = out.Close()
		restored++
	}
	return restored, nil
}

func writeTarBytes(tw *tar.Writer, name string, body []byte, mode int64) error {
	h := &tar.Header{
		Name:    name,
		Mode:    mode,
		Size:    int64(len(body)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	_, err := tw.Write(body)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func runSetupCompletion(args []string) {
	if len(args) == 0 {
		printSetupCompletionUsage()
		os.Exit(2)
	}
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	args = args[1:]
	switch mode {
	case "install":
		fs := flag.NewFlagSet("setup completion install", flag.ExitOnError)
		shell := fs.String("shell", "bash", "shell type: bash|zsh")
		force := fs.Bool("force", false, "overwrite existing completion file")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd setup completion install [-shell bash|zsh] [-force] [-json]")
		}
		path, err := installShellCompletion(strings.TrimSpace(*shell), *force)
		if err != nil {
			log.Fatal(err)
		}
		out := map[string]any{"success": true, "shell": *shell, "path": path}
		if *jsonOutput {
			printJSON(out)
			return
		}
		fmt.Printf("installed completion: shell=%s path=%s\n", *shell, path)
	case "status":
		fs := flag.NewFlagSet("setup completion status", flag.ExitOnError)
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd setup completion status [-json]")
		}
		status := map[string]any{
			"bash_path":      completionPath("bash"),
			"bash_installed": fileExists(completionPath("bash")),
			"zsh_path":       completionPath("zsh"),
			"zsh_installed":  fileExists(completionPath("zsh")),
		}
		if *jsonOutput {
			printJSON(status)
			return
		}
		fmt.Printf("bash_installed=%v path=%v\n", status["bash_installed"], status["bash_path"])
		fmt.Printf("zsh_installed=%v path=%v\n", status["zsh_installed"], status["zsh_path"])
	case "uninstall":
		fs := flag.NewFlagSet("setup completion uninstall", flag.ExitOnError)
		shell := fs.String("shell", "all", "shell type: bash|zsh|all")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args)
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd setup completion uninstall [-shell bash|zsh|all] [-json]")
		}
		removed, err := uninstallShellCompletion(strings.TrimSpace(*shell))
		if err != nil {
			log.Fatal(err)
		}
		out := map[string]any{"success": true, "shell": *shell, "removed": removed}
		if *jsonOutput {
			printJSON(out)
			return
		}
		fmt.Printf("removed completion files=%d\n", removed)
	default:
		printSetupCompletionUsage()
		os.Exit(2)
	}
}

func printSetupCompletionUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd setup completion install [-shell bash|zsh] [-force] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd setup completion status [-json]")
	fmt.Fprintln(os.Stderr, "  agentd setup completion uninstall [-shell bash|zsh|all] [-json]")
}

func installShellCompletion(shell string, force bool) (string, error) {
	path := completionPath(shell)
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
	if fileExists(path) && !force {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	script := completionScript(shell)
	return path, os.WriteFile(path, []byte(script), 0o644)
}

func uninstallShellCompletion(shell string) (int, error) {
	targets := []string{}
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		targets = append(targets, completionPath("bash"))
	case "zsh":
		targets = append(targets, completionPath("zsh"))
	case "all", "":
		targets = append(targets, completionPath("bash"), completionPath("zsh"))
	default:
		return 0, fmt.Errorf("unsupported shell: %s", shell)
	}
	removed := 0
	for _, target := range targets {
		if strings.TrimSpace(target) == "" || !fileExists(target) {
			continue
		}
		if err := os.Remove(target); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func completionPath(shell string) string {
	home, _ := os.UserHomeDir()
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "agentd")
	case "zsh":
		return filepath.Join(home, ".zsh", "completions", "_agentd")
	default:
		return ""
	}
}

func completionScript(shell string) string {
	commands := []string{
		"chat tui web serve setup bootstrap update version gateway sessions plugins research config model doctor",
	}
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh":
		return "#compdef agentd\n_arguments '1: :(" + commands[0] + ")'\n"
	default:
		return "_agentd_completions(){\n  COMPREPLY=( $(compgen -W \"" + commands[0] + "\" -- \"${COMP_WORDS[1]}\") )\n}\ncomplete -F _agentd_completions agentd\n"
	}
}
