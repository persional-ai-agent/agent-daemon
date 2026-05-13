package research

import (
	"os"
	"path/filepath"
	"testing"
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
