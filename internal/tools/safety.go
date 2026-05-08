package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var hardlineCommandPatterns = []struct {
	re          *regexp.Regexp
	description string
}{
	{
		re:          regexp.MustCompile(`(?i)\brm\s+(-[^\s]*\s+)*(?:/|/\*|/home(?:/\*)?|/root(?:/\*)?|~(?:/\*)?)(?:\s|$)`),
		description: "recursive delete of a protected root path",
	},
	{
		re:          regexp.MustCompile(`(?i)\bmkfs(?:\.[a-z0-9]+)?\b`),
		description: "filesystem formatting command",
	},
	{
		re:          regexp.MustCompile(`(?i)\bdd\b[^\n]*\bof=/dev/(?:sd|nvme|hd|mmcblk|vd|xvd)[a-z0-9]*`),
		description: "raw block device overwrite",
	},
	{
		re:          regexp.MustCompile(`(?i)\bkill\s+(-[^\s]+\s+)*-1\b`),
		description: "kill all processes command",
	},
	{
		re:          regexp.MustCompile(`(?i)(^|[;&|\n` + "`" + `]\s*)(?:sudo\s+)?(?:shutdown|reboot|halt|poweroff)\b`),
		description: "system shutdown or reboot command",
	},
}

var dangerousCommandPatterns = []struct {
	category    string
	re          *regexp.Regexp
	description string
}{
	{
		category:    "recursive_delete",
		re:          regexp.MustCompile(`(?i)\brm\s+(-[^\s]*\s+)*(?:--recursive\b|-[^\s]*r[^\s]*\b)`),
		description: "recursive delete command",
	},
	{
		category:    "world_writable",
		re:          regexp.MustCompile(`(?i)\bchmod\s+(-[^\s]*\s+)*(?:777|666|a\+w|o\+w)\b`),
		description: "world writable permission change",
	},
	{
		category:    "root_ownership",
		re:          regexp.MustCompile(`(?i)\b(chown)\s+(-[^\s]*\s+)*(?:root|0:0)\b`),
		description: "ownership change to root",
	},
	{
		category:    "remote_pipe_shell",
		re:          regexp.MustCompile(`(?i)\b(curl|wget)\b[^\n]*\|\s*(?:bash|sh)\b`),
		description: "remote content piped to shell",
	},
	{
		category:    "service_lifecycle",
		re:          regexp.MustCompile(`(?i)\b(systemctl)\s+(-[^\s]*\s+)*(?:stop|restart|disable|mask)\b`),
		description: "system service lifecycle command",
	},
}

func detectHardlineCommand(command string) (string, bool) {
	for _, item := range hardlineCommandPatterns {
		if item.re.MatchString(command) {
			return item.description, true
		}
	}
	return "", false
}

func detectDangerousCommand(command string) (category string, description string, dangerous bool) {
	for _, item := range dangerousCommandPatterns {
		if item.re.MatchString(command) {
			return item.category, item.description, true
		}
	}
	return "", "", false
}

func resolvePathWithinWorkdir(workdir, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("path required")
	}
	root, err := normalizedWorkdir(workdir)
	if err != nil {
		return "", err
	}
	target := strings.TrimSpace(input)
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workdir: %s", input)
	}
	return target, nil
}

func normalizedWorkdir(workdir string) (string, error) {
	base := strings.TrimSpace(workdir)
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = wd
	}
	return filepath.Abs(filepath.Clean(base))
}
