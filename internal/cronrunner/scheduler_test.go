package cronrunner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"github.com/dingjingmaster/agent-daemon/internal/store"
)

func TestComputeNextCronExpression(t *testing.T) {
	now := time.Date(2026, 5, 11, 9, 7, 30, 0, time.UTC)
	next, pause := computeNext(store.CronJob{
		ID:           "cron-job",
		ScheduleKind: "cron",
		ScheduleExpr: "*/15 9-17 * * 1-5",
	}, now)
	if pause {
		t.Fatal("cron job should stay active")
	}
	if next == nil {
		t.Fatal("next run is nil")
	}
	want := time.Date(2026, 5, 11, 9, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next=%s want=%s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestComputeNextInvalidCronPauses(t *testing.T) {
	next, pause := computeNext(store.CronJob{
		ID:           "bad-cron",
		ScheduleKind: "cron",
		ScheduleExpr: "bad",
	}, time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC))
	if !pause || next != nil {
		t.Fatalf("next=%v pause=%v, want nil/true", next, pause)
	}
}

func TestCronRunSessionIDUsesStableSessionForChainedContext(t *testing.T) {
	runID := "run1"
	if got := cronRunSessionID(store.CronJob{ID: "job1", ContextMode: "chained"}, runID); got != "cron:job1" {
		t.Fatalf("chained session id=%q", got)
	}
	if got := cronRunSessionID(store.CronJob{ID: "job1", ContextMode: "isolated"}, runID); got != "cron:job1:run1" {
		t.Fatalf("isolated session id=%q", got)
	}
}

type fakeCronSessionStore struct {
	requested string
	msgs      []core.Message
}

func (s *fakeCronSessionStore) AppendMessage(string, core.Message) error { return nil }
func (s *fakeCronSessionStore) LoadMessages(sessionID string, _ int) ([]core.Message, error) {
	s.requested = sessionID
	return core.CloneMessages(s.msgs), nil
}

func TestLoadCronRunHistoryOnlyForChainedContext(t *testing.T) {
	ss := &fakeCronSessionStore{msgs: []core.Message{{Role: "assistant", Content: "previous"}}}
	history := loadCronRunHistory(&agent.Engine{SessionStore: ss}, store.CronJob{ID: "job1", ContextMode: "chained"}, "cron:job1")
	if ss.requested != "cron:job1" || len(history) != 1 || history[0].Content != "previous" {
		t.Fatalf("history=%+v requested=%q", history, ss.requested)
	}

	ss.requested = ""
	history = loadCronRunHistory(&agent.Engine{SessionStore: ss}, store.CronJob{ID: "job1", ContextMode: "isolated"}, "cron:job1:run1")
	if history != nil || ss.requested != "" {
		t.Fatalf("isolated history=%+v requested=%q", history, ss.requested)
	}
}

func TestExecuteJobScriptMode(t *testing.T) {
	s := &Scheduler{Engine: &agent.Engine{Workdir: t.TempDir()}}
	out, err := s.executeJob(context.Background(), store.CronJob{
		RunMode:       "script",
		ScriptCommand: "printf cron-script-ok",
		ScriptTimeout: 10,
	}, "cron:job1:run1", &agent.Engine{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(out, "exit_code=0", "cron-script-ok") {
		t.Fatalf("unexpected script output: %q", out)
	}
}

type fakeDeliveryAdapter struct {
	name    string
	chatID  string
	content string
}

func (f *fakeDeliveryAdapter) Name() string                     { return f.name }
func (f *fakeDeliveryAdapter) Connect(context.Context) error    { return nil }
func (f *fakeDeliveryAdapter) Disconnect(context.Context) error { return nil }
func (f *fakeDeliveryAdapter) Send(_ context.Context, chatID, content, _ string) (platform.SendResult, error) {
	f.chatID = chatID
	f.content = content
	return platform.SendResult{Success: true, MessageID: "m1"}, nil
}
func (f *fakeDeliveryAdapter) EditMessage(context.Context, string, string, string) error { return nil }
func (f *fakeDeliveryAdapter) SendTyping(context.Context, string) error                  { return nil }
func (f *fakeDeliveryAdapter) OnMessage(context.Context, platform.MessageHandler)        {}

func TestDeliverRunResultSendsOutput(t *testing.T) {
	adapter := &fakeDeliveryAdapter{name: "cronfake"}
	platform.Register(adapter)
	defer platform.Unregister("cronfake")

	target, status, messageID, deliveryErr := deliverRunResult(context.Background(), store.CronJob{
		ID:             "job1",
		Name:           "Daily",
		DeliveryTarget: "cronfake:chat-1",
		DeliverOn:      "always",
	}, "run1", "report body", nil, "")
	if target != "cronfake:chat-1" || status != "sent" || messageID != "m1" || deliveryErr != "" {
		t.Fatalf("unexpected delivery result target=%q status=%q messageID=%q err=%q", target, status, messageID, deliveryErr)
	}
	if adapter.chatID != "chat-1" {
		t.Fatalf("chatID=%q", adapter.chatID)
	}
	if adapter.content == "" || !containsAll(adapter.content, "Cron job Daily completed", "report body", "run_id=run1") {
		t.Fatalf("unexpected content: %q", adapter.content)
	}
}

func TestDeliverRunResultHonorsDeliverOn(t *testing.T) {
	target, status, messageID, deliveryErr := deliverRunResult(context.Background(), store.CronJob{
		ID:             "job1",
		DeliveryTarget: "cronfake:chat-1",
		DeliverOn:      "failure",
	}, "run1", "ok", nil, "")
	if target != "cronfake:chat-1" || status != "skipped" || messageID != "" || deliveryErr != "" {
		t.Fatalf("unexpected delivery result target=%q status=%q messageID=%q err=%q", target, status, messageID, deliveryErr)
	}
}

func containsAll(s string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}
