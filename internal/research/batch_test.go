package research

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func TestLoadTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.jsonl")
	body := `{"input":"task1"}
{"id":"t2","session_id":"s2","input":"task2"}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tasks, err := LoadTasks(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len=%d", len(tasks))
	}
	if tasks[0].ID == "" || tasks[0].SessionID == "" {
		t.Fatalf("auto ids not generated: %#v", tasks[0])
	}
}

func TestCompressAndStatsTrajectories(t *testing.T) {
	in := filepath.Join(t.TempDir(), "traj.jsonl")
	out := filepath.Join(t.TempDir(), "traj.compact.jsonl.gz")
	content := `{"id":"1","session_id":"s1","input":"a","started_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:01Z","duration_ms":1000,"success":true,"result":{"session_id":"s1","final_response":"ok","turns_used":1}}
{"id":"2","session_id":"s2","input":"b","started_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:02Z","duration_ms":2000,"success":false,"error":"bad"}
`
	if err := os.WriteFile(in, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	meta, err := CompressTrajectories(in, out, 200)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := meta["success"].(bool); !ok {
		t.Fatalf("meta=%#v", meta)
	}
	stats, err := StatsTrajectories(in)
	if err != nil {
		t.Fatal(err)
	}
	if stats["total"].(int) != 2 {
		t.Fatalf("stats=%#v", stats)
	}
}

type staticResearchClient struct{}

func (staticResearchClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	return core.Message{Role: "assistant", Content: "ok"}, nil
}

func TestRunBatchWithOptionsAndExportFilter(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "traj.jsonl")
	eng := &agent.Engine{
		Client:       staticResearchClient{},
		Registry:     tools.NewRegistry(),
		SystemPrompt: agent.DefaultSystemPrompt(),
	}
	tasks := []Task{
		{ID: "t1", SessionID: "s1", Input: "hello", Metadata: map[string]any{"env": "bench-a"}},
		{ID: "t2", SessionID: "s2", Input: "world", Metadata: map[string]any{"env": "bench-b"}},
	}
	report, err := RunBatchWithOptions(context.Background(), eng, tasks, out, RunOptions{Concurrency: 2, TimeoutSec: 5})
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.Succeeded != 2 {
		t.Fatalf("unexpected report: %#v", report)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"reward"`) || !strings.Contains(string(raw), `"outcome"`) {
		t.Fatalf("trajectory missing reward/outcome: %s", string(raw))
	}

	stats, err := StatsTrajectoriesWithFilter(out, FilterOptions{Success: boolPtr(true), Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if stats["total"].(int) < 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}

	exportOut := filepath.Join(tmp, "export.jsonl")
	meta, err := ExportTrajectories(out, exportOut, FilterOptions{Success: boolPtr(true), Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if meta["exported"].(int) != 1 {
		t.Fatalf("unexpected export meta: %#v", meta)
	}
	bs, err := os.ReadFile(exportOut)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(bs))
	var row map[string]any
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(anyToString(row["label"])) != "success" {
		t.Fatalf("unexpected row: %#v", row)
	}
}

func boolPtr(v bool) *bool { return &v }

func anyToString(v any) string {
	s, _ := v.(string)
	return s
}
