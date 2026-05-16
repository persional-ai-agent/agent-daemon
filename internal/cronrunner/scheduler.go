package cronrunner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/cron"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"github.com/dingjingmaster/agent-daemon/internal/store"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type Scheduler struct {
	Store          *store.CronStore
	Engine         *agent.Engine
	Tick           time.Duration
	MaxConcurrency int

	sem   chan struct{}
	once  sync.Once
	mu    sync.Mutex
	byJob map[string]int
}

func (s *Scheduler) Start(ctx context.Context) error {
	if s.Store == nil {
		return errors.New("cron scheduler requires store")
	}
	if s.Engine == nil {
		return errors.New("cron scheduler requires engine")
	}
	if s.Tick <= 0 {
		s.Tick = 5 * time.Second
	}
	if s.MaxConcurrency <= 0 {
		s.MaxConcurrency = 1
	}
	s.once.Do(func() {
		s.sem = make(chan struct{}, s.MaxConcurrency)
		s.byJob = map[string]int{}
	})

	go func() {
		t := time.NewTicker(s.Tick)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.tick(ctx)
			}
		}
	}()
	return nil
}

func (s *Scheduler) tick(ctx context.Context) {
	now := time.Now().UTC()
	jobs, err := s.Store.DueJobs(ctx, now, 50)
	if err != nil {
		log.Printf("[cron] due jobs query failed: %v", err)
		return
	}
	for _, job := range jobs {
		select {
		case s.sem <- struct{}{}:
			go func(j store.CronJob) {
				defer func() { <-s.sem }()
				if !s.acquireJobSlot(j.ID, j.MaxConcurrency) {
					return
				}
				defer s.releaseJobSlot(j.ID)
				s.runJob(ctx, j)
			}(job)
		default:
			return
		}
	}
}

func (s *Scheduler) runJob(ctx context.Context, job store.CronJob) {
	if job.NextRunAt == nil {
		_ = s.Store.SetPaused(ctx, job.ID, true)
		return
	}

	nextRunAt, pauseAfter := computeNext(job, time.Now().UTC())
	repeatCompleted := job.RepeatComplete + 1
	paused := job.Paused || pauseAfter
	if job.RepeatTimes != nil && repeatCompleted >= *job.RepeatTimes {
		paused = true
		nextRunAt = nil
	}
	if err := s.Store.MarkJobScheduled(ctx, job.ID, nextRunAt, repeatCompleted, paused); err != nil {
		log.Printf("[cron] mark scheduled failed: job=%s err=%v", job.ID, err)
		return
	}

	runID := uuid.NewString()
	sessionID := cronRunSessionID(job, runID)
	run := store.CronRun{
		ID:        runID,
		JobID:     job.ID,
		SessionID: sessionID,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateRun(ctx, run); err != nil {
		log.Printf("[cron] create run failed: job=%s err=%v", job.ID, err)
		return
	}

	eng := *s.Engine
	out, runErr := s.executeJobWithRetry(ctx, job, sessionID, &eng)
	if err := s.Store.FinishRun(ctx, runID, "completed", out, runErr); err != nil {
		log.Printf("[cron] finish run failed: job=%s run=%s err=%v", job.ID, runID, err)
	}
	target, status, messageID, deliveryErr := deliverRunResult(ctx, job, runID, out, runErr, s.Engine.Workdir)
	if status != "" {
		if err := s.Store.SetRunDelivery(ctx, runID, target, status, messageID, deliveryErr); err != nil {
			log.Printf("[cron] delivery status update failed: job=%s run=%s err=%v", job.ID, runID, err)
		}
	}
}

func (s *Scheduler) executeJobWithRetry(ctx context.Context, job store.CronJob, sessionID string, eng *agent.Engine) (string, error) {
	out, err := s.executeJob(ctx, job, sessionID, eng)
	if err == nil || job.RetryMax <= 0 {
		return out, err
	}
	delay := time.Duration(job.RetryDelaySec) * time.Second
	for attempt := 1; attempt <= job.RetryMax; attempt++ {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(delay):
			}
		}
		retryOut, retryErr := s.executeJob(ctx, job, sessionID, eng)
		out = strings.TrimSpace(out + "\n\n[retry_attempt=" + fmt.Sprint(attempt) + "]\n" + strings.TrimSpace(retryOut))
		err = retryErr
		if retryErr == nil {
			return out, nil
		}
	}
	return out, err
}

func (s *Scheduler) executeJob(ctx context.Context, job store.CronJob, sessionID string, eng *agent.Engine) (string, error) {
	runCtx := ctx
	cancel := func() {}
	if job.RunTimeoutSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(job.RunTimeoutSec)*time.Second)
	}
	defer cancel()
	if normalizeRunMode(job.RunMode) == "script" {
		cwd := strings.TrimSpace(job.ScriptCWD)
		if cwd == "" && s.Engine != nil {
			cwd = strings.TrimSpace(s.Engine.Workdir)
		}
		timeout := job.ScriptTimeout
		if timeout <= 0 {
			timeout = 120
		}
		out, code, err := tools.RunForeground(runCtx, job.ScriptCommand, cwd, timeout)
		if err != nil {
			return fmt.Sprintf("exit_code=%d\n%s", code, strings.TrimSpace(out)), err
		}
		if code != 0 {
			return fmt.Sprintf("exit_code=%d\n%s", code, strings.TrimSpace(out)), fmt.Errorf("script exited with code %d", code)
		}
		return fmt.Sprintf("exit_code=%d\n%s", code, strings.TrimSpace(out)), nil
	}
	eng.TodoStore = nil
	history := loadCronRunHistory(s.Engine, job, sessionID)
	res, runErr := eng.Run(runCtx, sessionID, job.Prompt, agent.DefaultSystemPrompt(), history)
	out := ""
	if res != nil {
		out = res.FinalResponse
	}
	return out, runErr
}

func (s *Scheduler) acquireJobSlot(jobID string, max int) bool {
	if max <= 0 {
		max = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byJob == nil {
		s.byJob = map[string]int{}
	}
	if s.byJob[jobID] >= max {
		return false
	}
	s.byJob[jobID]++
	return true
}

func (s *Scheduler) releaseJobSlot(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byJob == nil {
		return
	}
	s.byJob[jobID]--
	if s.byJob[jobID] <= 0 {
		delete(s.byJob, jobID)
	}
}

func cronRunSessionID(job store.CronJob, runID string) string {
	if strings.EqualFold(strings.TrimSpace(job.ContextMode), "chained") {
		return fmt.Sprintf("cron:%s", job.ID)
	}
	return fmt.Sprintf("cron:%s:%s", job.ID, runID)
}

func loadCronRunHistory(eng *agent.Engine, job store.CronJob, sessionID string) []core.Message {
	if !strings.EqualFold(strings.TrimSpace(job.ContextMode), "chained") || eng == nil || eng.SessionStore == nil {
		return nil
	}
	history, err := eng.SessionStore.LoadMessages(sessionID, 500)
	if err != nil {
		log.Printf("[cron] load chained context failed: job=%s session=%s err=%v", job.ID, sessionID, err)
		return nil
	}
	return history
}

func computeNext(job store.CronJob, now time.Time) (*time.Time, bool) {
	switch job.ScheduleKind {
	case "interval":
		if job.IntervalMins <= 0 {
			return nil, true
		}
		t := now.Add(time.Duration(job.IntervalMins) * time.Minute).UTC()
		return &t, false
	case "once":
		return nil, true
	case "cron":
		if job.ScheduleExpr == "" {
			return nil, true
		}
		t, err := cron.NextRun(job.ScheduleExpr, now)
		if err != nil {
			log.Printf("[cron] invalid cron expression: job=%s expr=%q err=%v", job.ID, job.ScheduleExpr, err)
			return nil, true
		}
		return &t, false
	default:
		return nil, true
	}
}

func deliverRunResult(ctx context.Context, job store.CronJob, runID, output string, runErr error, workdir string) (target, status, messageID, deliveryErr string) {
	target = strings.TrimSpace(job.DeliveryTarget)
	if target == "" {
		return "", "", "", ""
	}
	deliverOn := strings.ToLower(strings.TrimSpace(job.DeliverOn))
	if deliverOn == "" {
		deliverOn = "always"
	}
	success := runErr == nil
	switch deliverOn {
	case "success":
		if !success {
			return target, "skipped", "", ""
		}
	case "failure":
		if success {
			return target, "skipped", "", ""
		}
	case "always":
	default:
		deliverOn = "always"
	}

	platformName, chatID, ok := parseDeliveryTarget(target)
	if !ok {
		return target, "failed", "", "invalid delivery_target (expected platform:chat_id)"
	}
	a, ok := platform.Get(platformName)
	if !ok {
		return target, "failed", "", "platform adapter not connected: " + platformName
	}

	body := buildDeliveryMessage(job, runID, output, runErr)
	if mediaPath, ok := cronMediaPath(output, workdir); ok {
		ms, ok := a.(platform.MediaSender)
		if !ok {
			return target, "failed", "", "platform adapter does not support media delivery: " + platformName
		}
		res, err := ms.SendMedia(ctx, chatID, mediaPath, cronDeliveryHeader(job, runID, runErr), "")
		if err != nil {
			return target, "failed", "", err.Error()
		}
		if !res.Success && strings.TrimSpace(res.Error) != "" {
			return target, "failed", "", res.Error
		}
		return target, "sent", res.MessageID, ""
	}
	res, err := a.Send(ctx, chatID, body, "")
	if err != nil {
		return target, "failed", "", err.Error()
	}
	if !res.Success && strings.TrimSpace(res.Error) != "" {
		return target, "failed", "", res.Error
	}
	return target, "sent", res.MessageID, ""
}

func parseDeliveryTarget(target string) (platformName, chatID string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(target), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	platformName = strings.ToLower(strings.TrimSpace(parts[0]))
	chatID = strings.TrimSpace(parts[1])
	return platformName, chatID, platformName != "" && chatID != ""
}

func buildDeliveryMessage(job store.CronJob, runID, output string, runErr error) string {
	body := strings.TrimSpace(output)
	if body == "" && runErr != nil {
		body = runErr.Error()
	}
	if body == "" {
		body = "(no output)"
	}
	msg := cronDeliveryHeader(job, runID, runErr) + "\n\n" + body
	return truncateDeliveryMessage(msg, 3900)
}

func cronDeliveryHeader(job store.CronJob, runID string, runErr error) string {
	status := "completed"
	if runErr != nil {
		status = "failed"
	}
	name := strings.TrimSpace(job.Name)
	if name == "" {
		name = job.ID
	}
	return fmt.Sprintf("Cron job %s %s\njob_id=%s run_id=%s", name, status, job.ID, runID)
}

func truncateDeliveryMessage(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}
	return s[:limit-3] + "..."
}

func cronMediaPath(output, workdir string) (string, bool) {
	msg := strings.TrimSpace(output)
	if !strings.HasPrefix(strings.ToUpper(msg), "MEDIA:") {
		return "", false
	}
	path := strings.TrimSpace(msg[len("MEDIA:"):])
	if path == "" {
		return "", false
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "", false
	}
	if strings.HasPrefix(clean, "/tmp/") || clean == "/tmp" {
		return clean, true
	}
	if strings.TrimSpace(workdir) == "" {
		return "", false
	}
	wd, err := filepath.Abs(workdir)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(wd, clean)
	if err != nil {
		return "", false
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return clean, true
	}
	return "", false
}

func normalizeRunMode(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "script") {
		return "script"
	}
	return "agent"
}
