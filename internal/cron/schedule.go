package cron

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	Kind         string     // once|interval|cron
	IntervalMins int        // interval only
	RunAt        *time.Time // once only
	Expr         string     // cron only (not executed yet)
	Display      string
}

var cronExprLike = regexp.MustCompile(`^[\d\*\-,/]+\s+[\d\*\-,/]+\s+[\d\*\-,/]+\s+[\d\*\-,/]+\s+[\d\*\-,/]+(?:\s+[\d\*\-,/]+)?$`)

func ParseSchedule(now time.Time, schedule string) (Schedule, error) {
	s := strings.TrimSpace(schedule)
	if s == "" {
		return Schedule{}, errors.New("schedule required")
	}
	low := strings.ToLower(s)

	if strings.HasPrefix(low, "every ") {
		mins, err := parseDurationMinutes(strings.TrimSpace(s[6:]))
		if err != nil {
			return Schedule{}, err
		}
		return Schedule{Kind: "interval", IntervalMins: mins, Display: fmt.Sprintf("every %dm", mins)}, nil
	}

	// duration -> one-shot
	if mins, err := parseDurationMinutes(s); err == nil {
		runAt := now.UTC().Add(time.Duration(mins) * time.Minute)
		return Schedule{Kind: "once", RunAt: &runAt, Display: fmt.Sprintf("in %dm", mins)}, nil
	}

	// timestamp -> one-shot
	if t, ok := parseTimestampUTC(s); ok {
		return Schedule{Kind: "once", RunAt: &t, Display: t.Format(time.RFC3339)}, nil
	}

	// cron-like -> store only (scheduler currently does not evaluate cron expressions)
	if cronExprLike.MatchString(s) {
		return Schedule{Kind: "cron", Expr: s, Display: s}, nil
	}

	return Schedule{}, fmt.Errorf("invalid schedule %q (use \"every 30m\", \"30m\", or RFC3339 timestamp)", schedule)
}

func parseDurationMinutes(s string) (int, error) {
	raw := strings.TrimSpace(strings.ToLower(s))
	m := regexp.MustCompile(`^(\d+)\s*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)$`).FindStringSubmatch(raw)
	if m == nil {
		return 0, errors.New("invalid duration")
	}
	n, _ := strconv.Atoi(m[1])
	if n <= 0 {
		return 0, errors.New("duration must be positive")
	}
	unit := m[2]
	switch unit[:1] {
	case "m":
		return n, nil
	case "h":
		return n * 60, nil
	case "d":
		return n * 1440, nil
	default:
		return 0, errors.New("invalid duration unit")
	}
}

func parseTimestampUTC(s string) (time.Time, bool) {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04", // Hermes docs accept this commonly
		"2006-01-02 15:04",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

