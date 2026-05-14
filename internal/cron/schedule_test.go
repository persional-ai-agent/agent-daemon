package cron

import (
	"testing"
	"time"
)

func TestParseScheduleInterval(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	s, err := ParseSchedule(now, "every 2h")
	if err != nil {
		t.Fatal(err)
	}
	if s.Kind != "interval" || s.IntervalMins != 120 {
		t.Fatalf("got kind=%s mins=%d", s.Kind, s.IntervalMins)
	}
}

func TestParseScheduleOnceDuration(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	s, err := ParseSchedule(now, "30m")
	if err != nil {
		t.Fatal(err)
	}
	if s.Kind != "once" || s.RunAt == nil {
		t.Fatalf("got kind=%s runAt=%v", s.Kind, s.RunAt)
	}
	if !s.RunAt.Equal(now.Add(30 * time.Minute)) {
		t.Fatalf("runAt=%s want=%s", s.RunAt.Format(time.RFC3339), now.Add(30*time.Minute).Format(time.RFC3339))
	}
}

func TestParseScheduleCronExpression(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	s, err := ParseSchedule(now, "*/15 9-17 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}
	if s.Kind != "cron" || s.Expr != "*/15 9-17 * * 1-5" {
		t.Fatalf("got kind=%s expr=%q", s.Kind, s.Expr)
	}
}

func TestNextRunCronFiveField(t *testing.T) {
	after := time.Date(2026, 5, 11, 9, 7, 30, 0, time.UTC)
	next, err := NextRun("*/15 9-17 * * 1-5", after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 11, 9, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s want=%s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestNextRunCronSixField(t *testing.T) {
	after := time.Date(2026, 5, 11, 9, 0, 4, 0, time.UTC)
	next, err := NextRun("*/10 * * * * *", after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 11, 9, 0, 10, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s want=%s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestNextRunCronSundayAlias(t *testing.T) {
	after := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC) // Monday
	next, err := NextRun("0 8 * * 7", after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 17, 8, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s want=%s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
