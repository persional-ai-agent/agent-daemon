package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCronStoreCreateListGet(t *testing.T) {
	ctx := context.Background()
	ss, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	cs, err := NewCronStore(ss.DB())
	if err != nil {
		t.Fatal(err)
	}

	next := time.Now().UTC().Add(5 * time.Minute)
	job, err := cs.CreateJob(ctx, CreateCronJobParams{
		ID:           "job123",
		Name:         "test",
		Prompt:       "hello",
		ScheduleKind: "interval",
		IntervalMins: 10,
		NextRunAt:    &next,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != "job123" {
		t.Fatalf("id=%q", job.ID)
	}

	jobs, err := cs.ListJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs=%d", len(jobs))
	}

	got, ok, err := cs.GetJob(ctx, "job123")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.ID != "job123" {
		t.Fatalf("ok=%v id=%q", ok, got.ID)
	}
}

