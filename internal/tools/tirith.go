package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

type tirithResult struct {
	Action   string `json:"action"`   // allow|warn|block
	Summary  string `json:"summary"`  // optional
	Findings any    `json:"findings"` // optional
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n := 0
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}

func tirithConfig() (enabled bool, bin string, timeoutSec int, failOpen bool) {
	bin = strings.TrimSpace(os.Getenv("TIRITH_BIN"))
	if bin == "" {
		bin = "tirith"
	}
	timeoutSec = envInt("TIRITH_TIMEOUT", 5)
	failOpen = envBool("TIRITH_FAIL_OPEN", true)

	// Default: enable iff binary is available.
	if _, ok := os.LookupEnv("TIRITH_ENABLED"); ok {
		enabled = envBool("TIRITH_ENABLED", true)
	} else {
		_, err := exec.LookPath(bin)
		enabled = err == nil
	}
	return enabled, bin, timeoutSec, failOpen
}

func checkCommandSecurityWithTirith(ctx context.Context, command string) tirithResult {
	enabled, bin, timeoutSec, failOpen := tirithConfig()
	if !enabled {
		return tirithResult{Action: "allow"}
	}
	if _, err := exec.LookPath(bin); err != nil {
		if failOpen {
			return tirithResult{Action: "allow", Summary: "tirith unavailable (fail-open)"}
		}
		return tirithResult{Action: "block", Summary: "tirith unavailable (fail-closed)"}
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Hermes invocation:
	// tirith check --json --non-interactive --shell posix -- <command>
	cmd := exec.CommandContext(runCtx, bin, "check", "--json", "--non-interactive", "--shell", "posix", "--", command)
	out, err := cmd.CombinedOutput()

	if runCtx.Err() == context.DeadlineExceeded {
		if failOpen {
			return tirithResult{Action: "allow", Summary: "tirith timed out (fail-open)"}
		}
		return tirithResult{Action: "block", Summary: "tirith timed out (fail-closed)"}
	}

	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			if failOpen {
				return tirithResult{Action: "allow", Summary: "tirith spawn failed (fail-open)"}
			}
			return tirithResult{Action: "block", Summary: "tirith spawn failed (fail-closed)"}
		}
	}

	action := "allow"
	switch exitCode {
	case 0:
		action = "allow"
	case 1:
		action = "block"
	case 2:
		action = "warn"
	default:
		if failOpen {
			return tirithResult{Action: "allow", Summary: "tirith unknown exit code (fail-open)"}
		}
		return tirithResult{Action: "block", Summary: "tirith unknown exit code (fail-closed)"}
	}

	var payload map[string]any
	if strings.TrimSpace(string(out)) != "" {
		_ = json.Unmarshal(out, &payload)
	}
	summary, _ := payload["summary"].(string)
	findings := payload["findings"]
	return tirithResult{Action: action, Summary: strings.TrimSpace(summary), Findings: findings}
}

