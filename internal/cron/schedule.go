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

	// cron-like -> scheduled by NextRun.
	if cronExprLike.MatchString(s) {
		return Schedule{Kind: "cron", Expr: s, Display: s}, nil
	}

	return Schedule{}, fmt.Errorf("invalid schedule %q (use \"every 30m\", \"30m\", or RFC3339 timestamp)", schedule)
}

type cronField struct {
	allowed  map[int]struct{}
	wildcard bool
}

type cronSpec struct {
	second cronField
	minute cronField
	hour   cronField
	dom    cronField
	month  cronField
	dow    cronField
	hasSec bool
}

func NextRun(expr string, after time.Time) (time.Time, error) {
	spec, err := parseCronSpec(expr)
	if err != nil {
		return time.Time{}, err
	}
	step := time.Minute
	next := after.UTC().Truncate(time.Minute).Add(time.Minute)
	if spec.hasSec {
		step = time.Second
		next = after.UTC().Truncate(time.Second).Add(time.Second)
	}
	deadline := after.UTC().AddDate(5, 0, 0)
	for !next.After(deadline) {
		if spec.matches(next) {
			return next, nil
		}
		next = next.Add(step)
	}
	return time.Time{}, fmt.Errorf("cron expression has no next run within 5 years: %q", expr)
}

func parseCronSpec(expr string) (cronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 && len(fields) != 6 {
		return cronSpec{}, fmt.Errorf("cron expression must have 5 or 6 fields")
	}
	spec := cronSpec{}
	var err error
	if len(fields) == 6 {
		spec.hasSec = true
		spec.second, err = parseCronField(fields[0], 0, 59, false)
		if err != nil {
			return cronSpec{}, fmt.Errorf("second: %w", err)
		}
		fields = fields[1:]
	} else {
		spec.second = cronField{allowed: map[int]struct{}{0: {}}, wildcard: false}
	}
	spec.minute, err = parseCronField(fields[0], 0, 59, false)
	if err != nil {
		return cronSpec{}, fmt.Errorf("minute: %w", err)
	}
	spec.hour, err = parseCronField(fields[1], 0, 23, false)
	if err != nil {
		return cronSpec{}, fmt.Errorf("hour: %w", err)
	}
	spec.dom, err = parseCronField(fields[2], 1, 31, false)
	if err != nil {
		return cronSpec{}, fmt.Errorf("day-of-month: %w", err)
	}
	spec.month, err = parseCronField(fields[3], 1, 12, false)
	if err != nil {
		return cronSpec{}, fmt.Errorf("month: %w", err)
	}
	spec.dow, err = parseCronField(fields[4], 0, 7, true)
	if err != nil {
		return cronSpec{}, fmt.Errorf("day-of-week: %w", err)
	}
	return spec, nil
}

func parseCronField(raw string, minVal, maxVal int, sundayAlias bool) (cronField, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cronField{}, errors.New("empty field")
	}
	out := cronField{allowed: map[int]struct{}{}, wildcard: raw == "*"}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, errors.New("empty list segment")
		}
		base, step := part, 1
		if strings.Contains(part, "/") {
			pieces := strings.SplitN(part, "/", 2)
			base = strings.TrimSpace(pieces[0])
			n, err := strconv.Atoi(strings.TrimSpace(pieces[1]))
			if err != nil || n <= 0 {
				return cronField{}, fmt.Errorf("invalid step %q", pieces[1])
			}
			step = n
		}
		start, end, err := cronFieldRange(base, minVal, maxVal)
		if err != nil {
			return cronField{}, err
		}
		for i := start; i <= end; i += step {
			v := i
			if sundayAlias && v == 7 {
				v = 0
			}
			out.allowed[v] = struct{}{}
		}
	}
	if len(out.allowed) == 0 {
		return cronField{}, errors.New("field matches nothing")
	}
	return out, nil
}

func cronFieldRange(base string, minVal, maxVal int) (int, int, error) {
	if base == "*" {
		return minVal, maxVal, nil
	}
	if strings.Contains(base, "-") {
		pieces := strings.SplitN(base, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(pieces[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range start %q", pieces[0])
		}
		end, err := strconv.Atoi(strings.TrimSpace(pieces[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range end %q", pieces[1])
		}
		if start > end {
			return 0, 0, fmt.Errorf("range start greater than end: %q", base)
		}
		if start < minVal || end > maxVal {
			return 0, 0, fmt.Errorf("range %q outside %d-%d", base, minVal, maxVal)
		}
		return start, end, nil
	}
	v, err := strconv.Atoi(base)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value %q", base)
	}
	if v < minVal || v > maxVal {
		return 0, 0, fmt.Errorf("value %d outside %d-%d", v, minVal, maxVal)
	}
	return v, v, nil
}

func (s cronSpec) matches(t time.Time) bool {
	if !s.second.has(t.Second()) || !s.minute.has(t.Minute()) || !s.hour.has(t.Hour()) || !s.month.has(int(t.Month())) {
		return false
	}
	domMatch := s.dom.has(t.Day())
	dowMatch := s.dow.has(int(t.Weekday()))
	if !s.dom.wildcard && !s.dow.wildcard {
		return domMatch || dowMatch
	}
	return domMatch && dowMatch
}

func (f cronField) has(v int) bool {
	_, ok := f.allowed[v]
	return ok
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
