package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/cron"
	"github.com/dingjingmaster/agent-daemon/internal/store"
)

type CronJobTool struct {
	Store *store.CronStore
}

func NewCronJobTool(store *store.CronStore) *CronJobTool {
	return &CronJobTool{Store: store}
}

func (t *CronJobTool) Name() string { return "cronjob" }

func (t *CronJobTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.Name(),
			Description: "Manage scheduled agent runs (cron jobs). Actions: create, list, get, update, pause, resume, remove, trigger, runs, run_get, replay.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Action to perform (default: list)",
						"enum":        []string{"create", "list", "get", "update", "pause", "resume", "remove", "trigger", "runs", "run_get", "replay"},
					},
					"job_id": map[string]any{"type": "string", "description": "Cron job id"},
					"run_id": map[string]any{"type": "string", "description": "Cron run id (for run_get)"},
					"name":   map[string]any{"type": "string", "description": "Optional job name"},
					"prompt": map[string]any{"type": "string", "description": "Prompt to run when the job fires (required for run_mode=agent)"},
					"run_mode": map[string]any{
						"type":        "string",
						"description": "Execution mode (default: agent)",
						"enum":        []string{"agent", "script", "no_agent"},
					},
					"script_command": map[string]any{
						"type":        "string",
						"description": "Shell command to execute when run_mode=script",
					},
					"script_cwd": map[string]any{
						"type":        "string",
						"description": "Optional working directory for run_mode=script; defaults to engine workdir",
					},
					"script_timeout": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "Optional timeout seconds for run_mode=script (default 120)",
					},
					"run_timeout_sec": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "Optional run timeout seconds for both agent/script execution.",
					},
					"retry_max": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "Automatic retry count on failure (default 0).",
					},
					"retry_delay_sec": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "Delay seconds between retry attempts (default 0).",
					},
					"max_concurrency": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "Per-job concurrency cap (default 1).",
					},
					"schedule": map[string]any{
						"type":        "string",
						"description": "Schedule: \"every 30m\", \"30m\" (one-shot), RFC3339 timestamp like 2026-02-03T14:00:00Z, or 5/6-field cron expression like \"*/15 9-17 * * 1-5\"",
					},
					"repeat": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "How many times to run (omit for forever). For one-shot schedules, default is 1.",
					},
					"delivery_target": map[string]any{
						"type":        "string",
						"description": "Optional result delivery target like telegram:123, discord:channel_id, slack:channel_id, or yuanbao:group:123. When set, the scheduler sends the final result after each run.",
					},
					"deliver_on": map[string]any{
						"type":        "string",
						"description": "When to deliver results (default: always)",
						"enum":        []string{"always", "success", "failure"},
					},
					"context_mode": map[string]any{
						"type":        "string",
						"description": "Cron run context mode (default: isolated). Set chained to reuse the job session and load previous run history.",
						"enum":        []string{"isolated", "chained"},
					},
					"chain_context": map[string]any{
						"type":        "boolean",
						"description": "Alias for context_mode=chained when true.",
					},
					"paused": map[string]any{"type": "boolean", "description": "Pause/resume the job (update only)"},
					"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "description": "Limit for runs listing (default 50, max 200)."},
				},
			},
		},
	}
}

func (t *CronJobTool) Call(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	if t.Store == nil {
		return nil, errors.New("cron store not configured")
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		action = "list"
	}
	switch action {
	case "create":
		runMode := normalizeCronRunModeForTool(strArg(args, "run_mode"))
		prompt := strings.TrimSpace(strArg(args, "prompt"))
		scriptCommand := strings.TrimSpace(strArg(args, "script_command"))
		scriptCWD := strings.TrimSpace(strArg(args, "script_cwd"))
		scriptTimeout := intArg(args, "script_timeout", 0)
		if scriptTimeout < 0 {
			scriptTimeout = 0
		}
		if runMode == "script" {
			if scriptCommand == "" {
				return map[string]any{"success": false, "error": "script_command required for run_mode=script"}, nil
			}
		} else if prompt == "" {
			return map[string]any{"success": false, "error": "prompt required"}, nil
		}
		schedRaw := strings.TrimSpace(strArg(args, "schedule"))
		if schedRaw == "" {
			return map[string]any{"success": false, "error": "schedule required"}, nil
		}
		now := time.Now().UTC()
		sched, err := cron.ParseSchedule(now, schedRaw)
		if err != nil {
			return nil, err
		}

		repeat := intArg(args, "repeat", 0)
		var repeatPtr *int
		if repeat > 0 {
			repeatPtr = &repeat
		}
		if sched.Kind == "once" && repeatPtr == nil {
			one := 1
			repeatPtr = &one
		}
		deliveryTarget := strings.TrimSpace(strArg(args, "delivery_target"))
		if deliveryTarget == "" {
			deliveryTarget = strings.TrimSpace(strArg(args, "target"))
		}
		deliverOn, ok := normalizeCronDeliverOn(strArg(args, "deliver_on"))
		if !ok {
			return map[string]any{"success": false, "error": "deliver_on must be one of always, success, failure"}, nil
		}
		contextMode, ok := normalizeCronContextMode(strArg(args, "context_mode"), boolArg(args, "chain_context", false))
		if !ok {
			return map[string]any{"success": false, "error": "context_mode must be one of isolated, chained"}, nil
		}

		id := uuid.NewString()[:12]
		name := strings.TrimSpace(strArg(args, "name"))
		if name == "" {
			name = id
		}

		nextRunAt := sched.RunAt
		if sched.Kind == "interval" {
			tm := now.Add(time.Duration(sched.IntervalMins) * time.Minute).UTC()
			nextRunAt = &tm
		} else if sched.Kind == "cron" {
			tm, err := cron.NextRun(sched.Expr, now)
			if err != nil {
				return nil, err
			}
			nextRunAt = &tm
		}

		job, err := t.Store.CreateJob(ctx, store.CreateCronJobParams{
			ID:             id,
			Name:           name,
			Prompt:         prompt,
			ScheduleKind:   sched.Kind,
			ScheduleExpr:   sched.Expr,
			IntervalMins:   sched.IntervalMins,
			RunAt:          sched.RunAt,
			NextRunAt:      nextRunAt,
			RepeatTimes:    repeatPtr,
			DeliveryTarget: deliveryTarget,
			DeliverOn:      deliverOn,
			ContextMode:    contextMode,
			RunMode:        runMode,
			ScriptCommand:  scriptCommand,
			ScriptCWD:      scriptCWD,
			ScriptTimeout:  scriptTimeout,
			RunTimeoutSec:  intArg(args, "run_timeout_sec", 0),
			RetryMax:       intArg(args, "retry_max", 0),
			RetryDelaySec:  intArg(args, "retry_delay_sec", 0),
			MaxConcurrency: intArg(args, "max_concurrency", 1),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job": job}, nil
	case "list":
		jobs, err := t.Store.ListJobs(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "count": len(jobs), "jobs": jobs}, nil
	case "get":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		job, ok, err := t.Store.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}
		return map[string]any{"success": true, "job": job}, nil
	case "update":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		cur, ok, err := t.Store.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}

		var upd store.UpdateCronJobParams
		if name := strings.TrimSpace(strArg(args, "name")); name != "" {
			upd.Name = &name
		}
		if prompt := strings.TrimSpace(strArg(args, "prompt")); prompt != "" {
			upd.Prompt = &prompt
		}
		if _, ok := args["run_mode"]; ok {
			runMode := normalizeCronRunModeForTool(strArg(args, "run_mode"))
			upd.RunMode = &runMode
		}
		if _, ok := args["script_command"]; ok {
			v := strings.TrimSpace(strArg(args, "script_command"))
			upd.ScriptCommand = &v
		}
		if _, ok := args["script_cwd"]; ok {
			v := strings.TrimSpace(strArg(args, "script_cwd"))
			upd.ScriptCWD = &v
		}
		if _, ok := args["script_timeout"]; ok {
			v := intArg(args, "script_timeout", 0)
			if v < 0 {
				v = 0
			}
			upd.ScriptTimeout = &v
		}
		if _, ok := args["run_timeout_sec"]; ok {
			v := intArg(args, "run_timeout_sec", 0)
			if v < 0 {
				v = 0
			}
			upd.RunTimeoutSec = &v
		}
		if _, ok := args["retry_max"]; ok {
			v := intArg(args, "retry_max", 0)
			if v < 0 {
				v = 0
			}
			upd.RetryMax = &v
		}
		if _, ok := args["retry_delay_sec"]; ok {
			v := intArg(args, "retry_delay_sec", 0)
			if v < 0 {
				v = 0
			}
			upd.RetryDelaySec = &v
		}
		if _, ok := args["max_concurrency"]; ok {
			v := intArg(args, "max_concurrency", 1)
			if v <= 0 {
				v = 1
			}
			upd.MaxConcurrency = &v
		}

		if v, ok := args["paused"]; ok {
			if b, ok := v.(bool); ok {
				upd.Paused = &b
			}
		}

		if schedRaw := strings.TrimSpace(strArg(args, "schedule")); schedRaw != "" {
			now := time.Now().UTC()
			sched, err := cron.ParseSchedule(now, schedRaw)
			if err != nil {
				return nil, err
			}
			upd.ScheduleKind = &sched.Kind
			expr := sched.Expr
			upd.ScheduleExpr = &expr
			if sched.Kind == "interval" {
				mins := sched.IntervalMins
				upd.IntervalMins = &mins
			} else {
				zero := 0
				upd.IntervalMins = &zero
			}

			// reset run_at/next_run_at
			runAt := sched.RunAt
			nextRunAt := sched.RunAt
			if sched.Kind == "interval" {
				tm := now.Add(time.Duration(sched.IntervalMins) * time.Minute).UTC()
				nextRunAt = &tm
			} else if sched.Kind == "cron" {
				tm, err := cron.NextRun(sched.Expr, now)
				if err != nil {
					return nil, err
				}
				nextRunAt = &tm
			}
			upd.RunAt = &runAt
			upd.NextRunAt = &nextRunAt

			// schedule changed -> reset completion counter
			zero := 0
			upd.RepeatCompleted = &zero
		}

		if repeat := intArg(args, "repeat", -1); repeat >= 0 {
			var rptr *int
			if repeat > 0 {
				rptr = &repeat
			}
			// If existing is once and repeat omitted, keep existing; but here repeat was set explicitly.
			upd.RepeatTimes = &rptr
		}
		if _, ok := args["delivery_target"]; ok {
			v := strings.TrimSpace(strArg(args, "delivery_target"))
			upd.DeliveryTarget = &v
		} else if _, ok := args["target"]; ok {
			v := strings.TrimSpace(strArg(args, "target"))
			upd.DeliveryTarget = &v
		}
		if _, ok := args["deliver_on"]; ok {
			deliverOn, valid := normalizeCronDeliverOn(strArg(args, "deliver_on"))
			if !valid {
				return map[string]any{"success": false, "error": "deliver_on must be one of always, success, failure"}, nil
			}
			upd.DeliverOn = &deliverOn
		}
		if _, ok := args["context_mode"]; ok {
			contextMode, valid := normalizeCronContextMode(strArg(args, "context_mode"), false)
			if !valid {
				return map[string]any{"success": false, "error": "context_mode must be one of isolated, chained"}, nil
			}
			upd.ContextMode = &contextMode
		} else if _, ok := args["chain_context"]; ok {
			contextMode, _ := normalizeCronContextMode("", boolArg(args, "chain_context", false))
			upd.ContextMode = &contextMode
		}

		// Default: ensure next_run_at exists for interval jobs if missing.
		if cur.ScheduleKind == "interval" && cur.NextRunAt == nil {
			now := time.Now().UTC()
			tm := now.Add(time.Duration(cur.IntervalMins) * time.Minute).UTC()
			next := &tm
			upd.NextRunAt = &next
		}

		job, ok2, err := t.Store.UpdateJob(ctx, id, upd)
		if err != nil {
			return nil, err
		}
		if !ok2 {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}
		return map[string]any{"success": true, "job": job}, nil
	case "pause":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		if err := t.Store.SetPaused(ctx, id, true); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "paused": true}, nil
	case "resume":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		if err := t.Store.SetPaused(ctx, id, false); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "paused": false}, nil
	case "remove":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		if err := t.Store.RemoveJob(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "removed": true}, nil
	case "trigger":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return map[string]any{"success": false, "error": "job_id required"}, nil
		}
		if err := t.Store.TriggerJob(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "triggered": true}, nil
	case "runs":
		id := strings.TrimSpace(strArg(args, "job_id"))
		limit := intArg(args, "limit", 50)
		runs, err := t.Store.ListRuns(ctx, id, limit)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "count": len(runs), "runs": runs}, nil
	case "run_get":
		runID := strings.TrimSpace(strArg(args, "run_id"))
		if runID == "" {
			return map[string]any{"success": false, "error": "run_id required"}, nil
		}
		run, ok, err := t.Store.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}
		return map[string]any{"success": true, "run": run}, nil
	case "replay":
		runID := strings.TrimSpace(strArg(args, "run_id"))
		if runID == "" {
			return map[string]any{"success": false, "error": "run_id required"}, nil
		}
		run, ok, err := t.Store.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}
		return map[string]any{
			"success": true,
			"run_id":  run.ID,
			"job_id":  run.JobID,
			"status":  run.Status,
			"replay": map[string]any{
				"output":          run.Output,
				"error":           run.Error,
				"delivery_target": run.DeliveryTarget,
				"delivery_status": run.DeliveryStatus,
				"delivery_error":  run.DeliveryError,
			},
		}, nil
	default:
		return map[string]any{"success": false, "error": fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

func normalizeCronDeliverOn(v string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "always":
		return "always", true
	case "success":
		return "success", true
	case "failure":
		return "failure", true
	default:
		return "", false
	}
}

func normalizeCronContextMode(v string, chain bool) (string, bool) {
	if chain {
		return "chained", true
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "isolated":
		return "isolated", true
	case "chained", "chain", "stateful":
		return "chained", true
	default:
		return "", false
	}
}

func normalizeCronRunModeForTool(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "script", "no_agent":
		return "script"
	default:
		return "agent"
	}
}
