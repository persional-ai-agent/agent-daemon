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
			Description: "Manage scheduled agent runs (cron jobs). Actions: create, list, get, update, pause, resume, remove, trigger, runs, run_get.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Action to perform (default: list)",
						"enum":        []string{"create", "list", "get", "update", "pause", "resume", "remove", "trigger", "runs", "run_get"},
					},
					"job_id": map[string]any{"type": "string", "description": "Cron job id"},
					"run_id": map[string]any{"type": "string", "description": "Cron run id (for run_get)"},
					"name":   map[string]any{"type": "string", "description": "Optional job name"},
					"prompt": map[string]any{"type": "string", "description": "Prompt to run when the job fires"},
					"schedule": map[string]any{
						"type":        "string",
						"description": "Schedule: \"every 30m\", \"30m\" (one-shot), or RFC3339 timestamp like 2026-02-03T14:00:00Z",
					},
					"repeat": map[string]any{
						"type":        "integer",
						"description": "How many times to run (omit for forever). For one-shot schedules, default is 1.",
					},
					"paused": map[string]any{"type": "boolean", "description": "Pause/resume the job (update only)"},
					"limit":  map[string]any{"type": "integer", "description": "Limit for runs listing (default 50, max 200)."},
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
		prompt := strings.TrimSpace(strArg(args, "prompt"))
		if prompt == "" {
			return nil, errors.New("prompt required")
		}
		schedRaw := strings.TrimSpace(strArg(args, "schedule"))
		if schedRaw == "" {
			return nil, errors.New("schedule required")
		}
		now := time.Now().UTC()
		sched, err := cron.ParseSchedule(now, schedRaw)
		if err != nil {
			return nil, err
		}
		if sched.Kind == "cron" {
			return nil, fmt.Errorf("cron expressions are stored but not yet executed by agent-daemon scheduler: %q", schedRaw)
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

		id := uuid.NewString()[:12]
		name := strings.TrimSpace(strArg(args, "name"))
		if name == "" {
			name = id
		}

		nextRunAt := sched.RunAt
		if sched.Kind == "interval" {
			tm := now.Add(time.Duration(sched.IntervalMins) * time.Minute).UTC()
			nextRunAt = &tm
		}

		job, err := t.Store.CreateJob(ctx, store.CreateCronJobParams{
			ID:           id,
			Name:         name,
			Prompt:       prompt,
			ScheduleKind: sched.Kind,
			ScheduleExpr: sched.Expr,
			IntervalMins: sched.IntervalMins,
			RunAt:        sched.RunAt,
			NextRunAt:    nextRunAt,
			RepeatTimes:  repeatPtr,
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
			return nil, errors.New("job_id required")
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
			return nil, errors.New("job_id required")
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
			if sched.Kind == "cron" {
				return nil, fmt.Errorf("cron expressions are stored but not yet executed by agent-daemon scheduler: %q", schedRaw)
			}
			upd.ScheduleKind = &sched.Kind
			if sched.Expr != "" {
				expr := sched.Expr
				upd.ScheduleExpr = &expr
			}
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
			return nil, errors.New("job_id required")
		}
		if err := t.Store.SetPaused(ctx, id, true); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "paused": true}, nil
	case "resume":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return nil, errors.New("job_id required")
		}
		if err := t.Store.SetPaused(ctx, id, false); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "paused": false}, nil
	case "remove":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return nil, errors.New("job_id required")
		}
		if err := t.Store.RemoveJob(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "job_id": id, "removed": true}, nil
	case "trigger":
		id := strings.TrimSpace(strArg(args, "job_id"))
		if id == "" {
			return nil, errors.New("job_id required")
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
			return nil, errors.New("run_id required")
		}
		run, ok, err := t.Store.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return map[string]any{"success": false, "error": "not_found"}, nil
		}
		return map[string]any{"success": true, "run": run}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
