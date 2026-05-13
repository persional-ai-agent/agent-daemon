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
