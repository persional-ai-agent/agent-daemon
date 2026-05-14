package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/store"
)

func TestCronjobDefaultActionList(t *testing.T) {
	ctx := context.Background()
	ss, err := store.NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	cs, err := store.NewCronStore(ss.DB())
	if err != nil {
		t.Fatal(err)
	}

	next := time.Now().UTC().Add(5 * time.Minute)
	_, err = cs.CreateJob(ctx, store.CreateCronJobParams{
		ID:           "job-default-list",
		Name:         "default-list",
		Prompt:       "hello",
		ScheduleKind: "interval",
		IntervalMins: 10,
		NextRunAt:    &next,
	})
	if err != nil {
		t.Fatal(err)
	}

	tool := NewCronJobTool(cs)
	res, err := tool.Call(ctx, map[string]any{}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := res["success"].(bool); !ok {
		t.Fatalf("expected success response: %+v", res)
	}
	if count, _ := res["count"].(int); count < 1 {
		t.Fatalf("expected at least one job in default list response: %+v", res)
	}
}

func TestCronjobSchemaDescriptionIncludesRunActions(t *testing.T) {
	schema := NewCronJobTool(nil).Schema()
	desc := schema.Function.Description
	if !strings.Contains(desc, "runs") || !strings.Contains(desc, "run_get") {
		t.Fatalf("unexpected cronjob description: %q", desc)
	}
}

func TestCronjobSchemaActionDescriptionIncludesDefault(t *testing.T) {
	schema := NewCronJobTool(nil).Schema()
	props, _ := schema.Function.Parameters["properties"].(map[string]any)
	action, _ := props["action"].(map[string]any)
	desc, _ := action["description"].(string)
	if !strings.Contains(desc, "default: list") {
		t.Fatalf("unexpected cronjob action description: %q", desc)
	}
}

func TestCronjobSchemaDocumentsRunsLimitBounds(t *testing.T) {
	schema := NewCronJobTool(nil).Schema()
	props, _ := schema.Function.Parameters["properties"].(map[string]any)
	limit, _ := props["limit"].(map[string]any)
	desc, _ := limit["description"].(string)
	if !strings.Contains(desc, "default 50") || !strings.Contains(desc, "max 200") {
		t.Fatalf("unexpected cronjob limit description: %q", desc)
	}
}

func TestCronjobCreateAcceptsCronExpression(t *testing.T) {
	ctx := context.Background()
	ss, err := store.NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	cs, err := store.NewCronStore(ss.DB())
	if err != nil {
		t.Fatal(err)
	}

	tool := NewCronJobTool(cs)
	res, err := tool.Call(ctx, map[string]any{
		"action":          "create",
		"name":            "weekday",
		"prompt":          "send report",
		"schedule":        "*/15 9-17 * * 1-5",
		"delivery_target": "telegram:123",
		"deliver_on":      "success",
		"context_mode":    "chained",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := res["success"].(bool); !ok {
		t.Fatalf("expected success response: %+v", res)
	}
	job, _ := res["job"].(store.CronJob)
	if job.ScheduleKind != "cron" || job.ScheduleExpr != "*/15 9-17 * * 1-5" || job.NextRunAt == nil {
		t.Fatalf("unexpected cron job: %+v", job)
	}
	if job.DeliveryTarget != "telegram:123" || job.DeliverOn != "success" {
		t.Fatalf("unexpected delivery fields: %+v", job)
	}
	if job.ContextMode != "chained" {
		t.Fatalf("unexpected context mode: %+v", job)
	}
}

func TestCronjobCreateScriptMode(t *testing.T) {
	ctx := context.Background()
	ss, err := store.NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	cs, err := store.NewCronStore(ss.DB())
	if err != nil {
		t.Fatal(err)
	}

	tool := NewCronJobTool(cs)
	res, err := tool.Call(ctx, map[string]any{
		"action":         "create",
		"name":           "script-job",
		"run_mode":       "script",
		"script_command": "echo test",
		"script_cwd":     "/tmp",
		"script_timeout": 25,
		"schedule":       "every 5m",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := res["success"].(bool); !ok {
		t.Fatalf("expected success response: %+v", res)
	}
	job, _ := res["job"].(store.CronJob)
	if job.RunMode != "script" || job.ScriptCommand != "echo test" || job.ScriptCWD != "/tmp" || job.ScriptTimeout != 25 {
		t.Fatalf("unexpected script fields: %+v", job)
	}
}
