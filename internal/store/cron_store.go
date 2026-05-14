package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CronJob struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Prompt         string     `json:"prompt"`
	ScheduleKind   string     `json:"schedule_kind"` // "once" | "interval" | "cron"
	ScheduleExpr   string     `json:"schedule_expr,omitempty"`
	IntervalMins   int        `json:"interval_mins,omitempty"`
	RunAt          *time.Time `json:"run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	RepeatTimes    *int       `json:"repeat_times,omitempty"`     // nil = forever
	RepeatComplete int        `json:"repeat_completed,omitempty"` // runs completed
	DeliveryTarget string     `json:"delivery_target,omitempty"`
	DeliverOn      string     `json:"deliver_on,omitempty"`   // always|success|failure
	ContextMode    string     `json:"context_mode,omitempty"` // isolated|chained
	RunMode        string     `json:"run_mode,omitempty"`     // agent|script
	ScriptCommand  string     `json:"script_command,omitempty"`
	ScriptCWD      string     `json:"script_cwd,omitempty"`
	ScriptTimeout  int        `json:"script_timeout,omitempty"` // seconds, 0=default
	Paused         bool       `json:"paused"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CronRun struct {
	ID                string     `json:"id"`
	JobID             string     `json:"job_id"`
	SessionID         string     `json:"session_id"`
	Status            string     `json:"status"` // running|completed|failed
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	Output            string     `json:"output,omitempty"`
	Error             string     `json:"error,omitempty"`
	DeliveryTarget    string     `json:"delivery_target,omitempty"`
	DeliveryStatus    string     `json:"delivery_status,omitempty"` // sent|failed|skipped
	DeliveryMessageID string     `json:"delivery_message_id,omitempty"`
	DeliveryError     string     `json:"delivery_error,omitempty"`
}

type CronStore struct {
	db *sql.DB
}

func NewCronStore(db *sql.DB) (*CronStore, error) {
	if db == nil {
		return nil, errors.New("cron store requires db")
	}
	s := &CronStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *CronStore) init() error {
	schema := `
CREATE TABLE IF NOT EXISTS cron_jobs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  prompt TEXT NOT NULL DEFAULT '',
  schedule_kind TEXT NOT NULL,
  schedule_expr TEXT NOT NULL DEFAULT '',
  interval_mins INTEGER NOT NULL DEFAULT 0,
  run_at TEXT NOT NULL DEFAULT '',
  next_run_at TEXT NOT NULL DEFAULT '',
  repeat_times INTEGER,
  repeat_completed INTEGER NOT NULL DEFAULT 0,
  delivery_target TEXT NOT NULL DEFAULT '',
  deliver_on TEXT NOT NULL DEFAULT 'always',
  context_mode TEXT NOT NULL DEFAULT 'isolated',
  run_mode TEXT NOT NULL DEFAULT 'agent',
  script_command TEXT NOT NULL DEFAULT '',
  script_cwd TEXT NOT NULL DEFAULT '',
  script_timeout INTEGER NOT NULL DEFAULT 0,
  paused INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_next_run_at ON cron_jobs(next_run_at);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_paused ON cron_jobs(paused);

CREATE TABLE IF NOT EXISTS cron_runs (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  output TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  delivery_target TEXT NOT NULL DEFAULT '',
  delivery_status TEXT NOT NULL DEFAULT '',
  delivery_message_id TEXT NOT NULL DEFAULT '',
  delivery_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_cron_runs_job_id ON cron_runs(job_id);
CREATE INDEX IF NOT EXISTS idx_cron_runs_started_at ON cron_runs(started_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	migrations := []struct {
		table string
		col   string
		def   string
	}{
		{"cron_jobs", "delivery_target", "TEXT NOT NULL DEFAULT ''"},
		{"cron_jobs", "deliver_on", "TEXT NOT NULL DEFAULT 'always'"},
		{"cron_jobs", "context_mode", "TEXT NOT NULL DEFAULT 'isolated'"},
		{"cron_jobs", "run_mode", "TEXT NOT NULL DEFAULT 'agent'"},
		{"cron_jobs", "script_command", "TEXT NOT NULL DEFAULT ''"},
		{"cron_jobs", "script_cwd", "TEXT NOT NULL DEFAULT ''"},
		{"cron_jobs", "script_timeout", "INTEGER NOT NULL DEFAULT 0"},
		{"cron_runs", "delivery_target", "TEXT NOT NULL DEFAULT ''"},
		{"cron_runs", "delivery_status", "TEXT NOT NULL DEFAULT ''"},
		{"cron_runs", "delivery_message_id", "TEXT NOT NULL DEFAULT ''"},
		{"cron_runs", "delivery_error", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, m := range migrations {
		if err := s.ensureColumn(m.table, m.col, m.def); err != nil {
			return err
		}
	}
	return nil
}

type CreateCronJobParams struct {
	ID             string
	Name           string
	Prompt         string
	ScheduleKind   string
	ScheduleExpr   string
	IntervalMins   int
	RunAt          *time.Time
	NextRunAt      *time.Time
	RepeatTimes    *int
	DeliveryTarget string
	DeliverOn      string
	ContextMode    string
	RunMode        string
	ScriptCommand  string
	ScriptCWD      string
	ScriptTimeout  int
}

type UpdateCronJobParams struct {
	Name            *string
	Prompt          *string
	ScheduleKind    *string
	ScheduleExpr    *string
	IntervalMins    *int
	RunAt           **time.Time
	NextRunAt       **time.Time
	RepeatTimes     **int
	DeliveryTarget  *string
	DeliverOn       *string
	ContextMode     *string
	RunMode         *string
	ScriptCommand   *string
	ScriptCWD       *string
	ScriptTimeout   *int
	Paused          *bool
	RepeatCompleted *int
}

func (s *CronStore) CreateJob(ctx context.Context, p CreateCronJobParams) (CronJob, error) {
	if strings.TrimSpace(p.ID) == "" {
		return CronJob{}, errors.New("id required")
	}
	if strings.TrimSpace(p.ScheduleKind) == "" {
		return CronJob{}, errors.New("schedule_kind required")
	}
	now := time.Now().UTC()
	job := CronJob{
		ID:             strings.TrimSpace(p.ID),
		Name:           strings.TrimSpace(p.Name),
		Prompt:         strings.TrimSpace(p.Prompt),
		ScheduleKind:   strings.TrimSpace(p.ScheduleKind),
		ScheduleExpr:   strings.TrimSpace(p.ScheduleExpr),
		IntervalMins:   p.IntervalMins,
		RunAt:          p.RunAt,
		NextRunAt:      p.NextRunAt,
		RepeatTimes:    p.RepeatTimes,
		DeliveryTarget: strings.TrimSpace(p.DeliveryTarget),
		DeliverOn:      normalizeDeliverOn(p.DeliverOn),
		ContextMode:    normalizeCronContextMode(p.ContextMode),
		RunMode:        normalizeCronRunMode(p.RunMode),
		ScriptCommand:  strings.TrimSpace(p.ScriptCommand),
		ScriptCWD:      strings.TrimSpace(p.ScriptCWD),
		ScriptTimeout:  p.ScriptTimeout,
		Paused:         false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if job.Name == "" {
		job.Name = job.ID
	}
	if job.RunMode == "script" {
		if job.ScriptCommand == "" {
			return CronJob{}, errors.New("script_command required for run_mode=script")
		}
	} else if job.Prompt == "" {
		return CronJob{}, errors.New("prompt required")
	}
	if job.NextRunAt == nil {
		return CronJob{}, errors.New("next_run_at required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO cron_jobs(
  id, name, prompt, schedule_kind, schedule_expr, interval_mins, run_at, next_run_at,
  repeat_times, repeat_completed, delivery_target, deliver_on, context_mode, run_mode, script_command, script_cwd, script_timeout, paused, created_at, updated_at
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
`,
		job.ID,
		job.Name,
		job.Prompt,
		job.ScheduleKind,
		job.ScheduleExpr,
		job.IntervalMins,
		formatTimePtr(job.RunAt),
		formatTimePtr(job.NextRunAt),
		job.RepeatTimes,
		job.DeliveryTarget,
		job.DeliverOn,
		job.ContextMode,
		job.RunMode,
		job.ScriptCommand,
		job.ScriptCWD,
		job.ScriptTimeout,
		job.CreatedAt.Format(time.RFC3339),
		job.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return CronJob{}, err
	}
	return job, nil
}

func (s *CronStore) UpdateJob(ctx context.Context, id string, p UpdateCronJobParams) (CronJob, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CronJob{}, false, errors.New("id required")
	}
	cur, ok, err := s.GetJob(ctx, id)
	if err != nil || !ok {
		return CronJob{}, ok, err
	}

	next := cur
	if p.Name != nil {
		next.Name = strings.TrimSpace(*p.Name)
	}
	if p.Prompt != nil {
		next.Prompt = strings.TrimSpace(*p.Prompt)
	}
	if p.ScheduleKind != nil {
		next.ScheduleKind = strings.TrimSpace(*p.ScheduleKind)
	}
	if p.ScheduleExpr != nil {
		next.ScheduleExpr = strings.TrimSpace(*p.ScheduleExpr)
	}
	if p.IntervalMins != nil {
		next.IntervalMins = *p.IntervalMins
	}
	if p.RunAt != nil {
		next.RunAt = *p.RunAt
	}
	if p.NextRunAt != nil {
		next.NextRunAt = *p.NextRunAt
	}
	if p.RepeatTimes != nil {
		next.RepeatTimes = *p.RepeatTimes
	}
	if p.DeliveryTarget != nil {
		next.DeliveryTarget = strings.TrimSpace(*p.DeliveryTarget)
	}
	if p.DeliverOn != nil {
		next.DeliverOn = normalizeDeliverOn(*p.DeliverOn)
	}
	if p.ContextMode != nil {
		next.ContextMode = normalizeCronContextMode(*p.ContextMode)
	}
	if p.RunMode != nil {
		next.RunMode = normalizeCronRunMode(*p.RunMode)
	}
	if p.ScriptCommand != nil {
		next.ScriptCommand = strings.TrimSpace(*p.ScriptCommand)
	}
	if p.ScriptCWD != nil {
		next.ScriptCWD = strings.TrimSpace(*p.ScriptCWD)
	}
	if p.ScriptTimeout != nil {
		next.ScriptTimeout = *p.ScriptTimeout
	}
	if p.Paused != nil {
		next.Paused = *p.Paused
	}
	if p.RepeatCompleted != nil {
		next.RepeatComplete = *p.RepeatCompleted
	}

	if strings.TrimSpace(next.Name) == "" {
		next.Name = next.ID
	}
	if next.RunMode == "script" {
		if strings.TrimSpace(next.ScriptCommand) == "" {
			return CronJob{}, true, errors.New("script_command required for run_mode=script")
		}
	} else if strings.TrimSpace(next.Prompt) == "" {
		return CronJob{}, true, errors.New("prompt required")
	}
	if strings.TrimSpace(next.ScheduleKind) == "" {
		return CronJob{}, true, errors.New("schedule_kind required")
	}

	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE cron_jobs
SET name = ?, prompt = ?, schedule_kind = ?, schedule_expr = ?, interval_mins = ?, run_at = ?, next_run_at = ?,
    repeat_times = ?, repeat_completed = ?, delivery_target = ?, deliver_on = ?, context_mode = ?, run_mode = ?, script_command = ?, script_cwd = ?, script_timeout = ?, paused = ?, updated_at = ?
WHERE id = ?
`,
		next.Name,
		next.Prompt,
		next.ScheduleKind,
		next.ScheduleExpr,
		next.IntervalMins,
		formatTimePtr(next.RunAt),
		formatTimePtr(next.NextRunAt),
		next.RepeatTimes,
		next.RepeatComplete,
		next.DeliveryTarget,
		next.DeliverOn,
		next.ContextMode,
		next.RunMode,
		next.ScriptCommand,
		next.ScriptCWD,
		next.ScriptTimeout,
		boolToInt(next.Paused),
		now.Format(time.RFC3339),
		next.ID,
	)
	if err != nil {
		return CronJob{}, true, err
	}
	updated, ok2, err := s.GetJob(ctx, id)
	return updated, ok2, err
}

func (s *CronStore) ListJobs(ctx context.Context) ([]CronJob, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, prompt, schedule_kind, schedule_expr, interval_mins, run_at, next_run_at, repeat_times, repeat_completed, delivery_target, deliver_on, context_mode, run_mode, script_command, script_cwd, script_timeout, paused, created_at, updated_at
FROM cron_jobs
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CronJob
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *CronStore) GetJob(ctx context.Context, id string) (CronJob, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, prompt, schedule_kind, schedule_expr, interval_mins, run_at, next_run_at, repeat_times, repeat_completed, delivery_target, deliver_on, context_mode, run_mode, script_command, script_cwd, script_timeout, paused, created_at, updated_at
FROM cron_jobs WHERE id = ?`, strings.TrimSpace(id))
	j, err := scanCronJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CronJob{}, false, nil
		}
		return CronJob{}, false, err
	}
	return j, true, nil
}

func (s *CronStore) SetPaused(ctx context.Context, id string, paused bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE cron_jobs SET paused = ?, updated_at = ? WHERE id = ?`, boolToInt(paused), time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(id))
	return err
}

func (s *CronStore) RemoveJob(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cron_jobs WHERE id = ?`, strings.TrimSpace(id))
	return err
}

func (s *CronStore) TriggerJob(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE cron_jobs SET next_run_at = ?, updated_at = ? WHERE id = ?`, now, now, strings.TrimSpace(id))
	return err
}

func (s *CronStore) DueJobs(ctx context.Context, now time.Time, limit int) ([]CronJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, prompt, schedule_kind, schedule_expr, interval_mins, run_at, next_run_at, repeat_times, repeat_completed, delivery_target, deliver_on, context_mode, run_mode, script_command, script_cwd, script_timeout, paused, created_at, updated_at
FROM cron_jobs
WHERE paused = 0 AND next_run_at != '' AND next_run_at <= ?
ORDER BY next_run_at ASC
LIMIT ?`, now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CronJob
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *CronStore) MarkJobScheduled(ctx context.Context, jobID string, nextRunAt *time.Time, repeatCompleted int, paused bool) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE cron_jobs
SET next_run_at = ?, repeat_completed = ?, paused = ?, updated_at = ?
WHERE id = ?
`, formatTimePtr(nextRunAt), repeatCompleted, boolToInt(paused), time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(jobID))
	return err
}

func (s *CronStore) CreateRun(ctx context.Context, run CronRun) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO cron_runs(id, job_id, session_id, status, started_at, finished_at, output, error, delivery_target, delivery_status, delivery_message_id, delivery_error)
VALUES(?, ?, ?, ?, ?, '', '', '', '', '', '', '')
`, run.ID, run.JobID, run.SessionID, run.Status, run.StartedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *CronStore) FinishRun(ctx context.Context, runID string, status string, output string, runErr error) error {
	finishedAt := time.Now().UTC()
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
		status = "failed"
	}
	if strings.TrimSpace(status) == "" {
		status = "completed"
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE cron_runs
SET status = ?, finished_at = ?, output = ?, error = ?
WHERE id = ?
`, status, finishedAt.Format(time.RFC3339), output, errText, strings.TrimSpace(runID))
	return err
}

func (s *CronStore) SetRunDelivery(ctx context.Context, runID, target, status, messageID, deliveryErr string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE cron_runs
SET delivery_target = ?, delivery_status = ?, delivery_message_id = ?, delivery_error = ?
WHERE id = ?
`, strings.TrimSpace(target), strings.TrimSpace(status), strings.TrimSpace(messageID), strings.TrimSpace(deliveryErr), strings.TrimSpace(runID))
	return err
}

func (s *CronStore) ListRuns(ctx context.Context, jobID string, limit int) ([]CronRun, error) {
	jobID = strings.TrimSpace(jobID)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	baseSQL := `SELECT id, job_id, session_id, status, started_at, finished_at, output, error, delivery_target, delivery_status, delivery_message_id, delivery_error FROM cron_runs`
	args := []any{}
	if jobID != "" {
		baseSQL += ` WHERE job_id = ?`
		args = append(args, jobID)
	}
	baseSQL += ` ORDER BY started_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, baseSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CronRun, 0)
	for rows.Next() {
		var r CronRun
		var startedAt, finishedAt string
		if err := rows.Scan(&r.ID, &r.JobID, &r.SessionID, &r.Status, &startedAt, &finishedAt, &r.Output, &r.Error, &r.DeliveryTarget, &r.DeliveryStatus, &r.DeliveryMessageID, &r.DeliveryError); err != nil {
			return nil, err
		}
		sa, err := time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return nil, err
		}
		r.StartedAt = sa
		if strings.TrimSpace(finishedAt) != "" {
			fa, err := time.Parse(time.RFC3339, finishedAt)
			if err != nil {
				return nil, err
			}
			r.FinishedAt = &fa
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *CronStore) GetRun(ctx context.Context, runID string) (CronRun, bool, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return CronRun{}, false, errors.New("run_id required")
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, job_id, session_id, status, started_at, finished_at, output, error, delivery_target, delivery_status, delivery_message_id, delivery_error FROM cron_runs WHERE id = ?`, runID)
	var r CronRun
	var startedAt, finishedAt string
	if err := row.Scan(&r.ID, &r.JobID, &r.SessionID, &r.Status, &startedAt, &finishedAt, &r.Output, &r.Error, &r.DeliveryTarget, &r.DeliveryStatus, &r.DeliveryMessageID, &r.DeliveryError); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CronRun{}, false, nil
		}
		return CronRun{}, false, err
	}
	sa, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return CronRun{}, false, err
	}
	r.StartedAt = sa
	if strings.TrimSpace(finishedAt) != "" {
		fa, err := time.Parse(time.RFC3339, finishedAt)
		if err != nil {
			return CronRun{}, false, err
		}
		r.FinishedAt = &fa
	}
	return r, true, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTimePtr(s string) (*time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &tt, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCronJob(scn scanner) (CronJob, error) {
	var j CronJob
	var runAt, nextRunAt string
	var repeat sql.NullInt64
	var paused int
	var createdAt, updatedAt string
	if err := scn.Scan(
		&j.ID, &j.Name, &j.Prompt,
		&j.ScheduleKind, &j.ScheduleExpr, &j.IntervalMins,
		&runAt, &nextRunAt,
		&repeat, &j.RepeatComplete,
		&j.DeliveryTarget, &j.DeliverOn, &j.ContextMode, &j.RunMode, &j.ScriptCommand, &j.ScriptCWD, &j.ScriptTimeout,
		&paused, &createdAt, &updatedAt,
	); err != nil {
		return CronJob{}, err
	}
	j.Paused = paused != 0
	if repeat.Valid {
		v := int(repeat.Int64)
		j.RepeatTimes = &v
	}
	if t, err := parseTimePtr(runAt); err == nil {
		j.RunAt = t
	} else {
		return CronJob{}, fmt.Errorf("parse run_at: %w", err)
	}
	if t, err := parseTimePtr(nextRunAt); err == nil {
		j.NextRunAt = t
	} else {
		return CronJob{}, fmt.Errorf("parse next_run_at: %w", err)
	}
	j.DeliverOn = normalizeDeliverOn(j.DeliverOn)
	j.ContextMode = normalizeCronContextMode(j.ContextMode)
	j.RunMode = normalizeCronRunMode(j.RunMode)
	ct, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return CronJob{}, fmt.Errorf("parse created_at: %w", err)
	}
	ut, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return CronJob{}, fmt.Errorf("parse updated_at: %w", err)
	}
	j.CreatedAt = ct
	j.UpdatedAt = ut
	return j, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *CronStore) ensureColumn(table, col, def string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == col {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + col + ` ` + def)
	return err
}

func normalizeDeliverOn(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "always":
		return "always"
	case "success":
		return "success"
	case "failure":
		return "failure"
	default:
		return "always"
	}
}

func normalizeCronContextMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "chained", "chain", "stateful":
		return "chained"
	default:
		return "isolated"
	}
}

func normalizeCronRunMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "script":
		return "script"
	default:
		return "agent"
	}
}
