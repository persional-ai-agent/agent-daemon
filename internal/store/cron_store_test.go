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
		ID:             "job123",
		Name:           "test",
		Prompt:         "hello",
		ScheduleKind:   "interval",
		IntervalMins:   10,
		NextRunAt:      &next,
		DeliveryTarget: "telegram:123",
		DeliverOn:      "success",
		ContextMode:    "chained",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != "job123" {
		t.Fatalf("id=%q", job.ID)
	}
	if job.DeliveryTarget != "telegram:123" || job.DeliverOn != "success" {
		t.Fatalf("delivery fields not preserved: %+v", job)
	}
	if job.ContextMode != "chained" {
		t.Fatalf("context mode not preserved: %+v", job)
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
	if got.DeliveryTarget != "telegram:123" || got.DeliverOn != "success" {
		t.Fatalf("stored delivery fields not preserved: %+v", got)
	}
	if got.ContextMode != "chained" {
		t.Fatalf("stored context mode not preserved: %+v", got)
	}
}

func TestCronStoreCreateScriptRunMode(t *testing.T) {
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
	next := time.Now().UTC().Add(2 * time.Minute)
	job, err := cs.CreateJob(ctx, CreateCronJobParams{
		ID:            "job-script",
		Name:          "script",
		ScheduleKind:  "interval",
		IntervalMins:  5,
		NextRunAt:     &next,
		RunMode:       "script",
		ScriptCommand: "echo hi",
		ScriptCWD:     "/tmp",
		ScriptTimeout: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.RunMode != "script" || job.ScriptCommand != "echo hi" || job.ScriptCWD != "/tmp" || job.ScriptTimeout != 30 {
		t.Fatalf("script fields not preserved: %+v", job)
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
	if err := cs.SetRunDelivery(ctx, "run1", "telegram:123", "sent", "m1", ""); err != nil {
		t.Fatal(err)
	}

	runs, err := cs.ListRuns(ctx, "job123", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ID != "run1" {
		t.Fatalf("runs=%v", runs)
	}
	if runs[0].DeliveryStatus != "sent" || runs[0].DeliveryMessageID != "m1" {
		t.Fatalf("delivery fields missing from list: %+v", runs[0])
	}

	got, ok, err := cs.GetRun(ctx, "run1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.JobID != "job123" {
		t.Fatalf("ok=%v got=%v", ok, got)
	}
	if got.DeliveryTarget != "telegram:123" || got.DeliveryStatus != "sent" || got.DeliveryMessageID != "m1" {
		t.Fatalf("delivery fields missing from get: %+v", got)
	}
}
