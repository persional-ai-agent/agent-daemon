package cronrunner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/store"
)

type Scheduler struct {
	Store          *store.CronStore
	Engine         *agent.Engine
	Tick           time.Duration
	MaxConcurrency int

	sem  chan struct{}
	once sync.Once
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
	sessionID := fmt.Sprintf("cron:%s:%s", job.ID, runID)
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
	eng.TodoStore = nil
	res, err := eng.Run(ctx, sessionID, job.Prompt, agent.DefaultSystemPrompt(), nil)
	out := ""
	if res != nil {
		out = res.FinalResponse
	}
	if err := s.Store.FinishRun(ctx, runID, "completed", out, err); err != nil {
		log.Printf("[cron] finish run failed: job=%s run=%s err=%v", job.ID, runID, err)
	}
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
		return nil, true
	default:
		return nil, true
	}
}

