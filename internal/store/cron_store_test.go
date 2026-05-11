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

func TestCronStoreUpdateJob(t *testing.T) {
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
	_, err = cs.CreateJob(ctx, CreateCronJobParams{
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

	newPrompt := "updated"
	job, ok, err := cs.UpdateJob(ctx, "job123", UpdateCronJobParams{Prompt: &newPrompt})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || job.Prompt != "updated" {
		t.Fatalf("ok=%v prompt=%q", ok, job.Prompt)
	}
}

func TestCronStoreListRunsAndGetRun(t *testing.T) {
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
	_, err = cs.CreateJob(ctx, CreateCronJobParams{
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

	run := CronRun{
		ID:        "run1",
		JobID:     "job123",
		SessionID: "cron:job123:run1",
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := cs.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := cs.FinishRun(ctx, "run1", "completed", "ok", nil); err != nil {
		t.Fatal(err)
	}

	runs, err := cs.ListRuns(ctx, "job123", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ID != "run1" {
		t.Fatalf("runs=%v", runs)
	}

	got, ok, err := cs.GetRun(ctx, "run1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.JobID != "job123" {
		t.Fatalf("ok=%v got=%v", ok, got)
	}
}
