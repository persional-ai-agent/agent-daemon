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

