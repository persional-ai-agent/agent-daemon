package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return strings.TrimSpace(buf.String())
}

type testSessionStore struct {
	msgs      []core.Message
	bySession map[string][]core.Message
}

func (s *testSessionStore) AppendMessage(sessionID string, msg core.Message) error {
	s.msgs = append(s.msgs, msg)
	if s.bySession == nil {
		s.bySession = map[string][]core.Message{}
	}
	s.bySession[sessionID] = append(s.bySession[sessionID], msg)
	return nil
}

func (s *testSessionStore) LoadMessages(sessionID string, _ int) ([]core.Message, error) {
	if s.bySession != nil {
		if msgs, ok := s.bySession[sessionID]; ok {
			out := make([]core.Message, len(msgs))
			copy(out, msgs)
			return out, nil
		}
	}
	out := make([]core.Message, len(s.msgs))
	copy(out, s.msgs)
	return out, nil
}

func (s *testSessionStore) ListRecentSessions(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	return []map[string]any{
		{"session_id": "s1", "last_seen": "2026-05-13T00:00:00Z"},
	}, nil
}

func (s *testSessionStore) LoadMessagesPage(_ string, offset, limit int) ([]core.Message, error) {
	if limit <= 0 {
		limit = 20
	}
	start := offset
	if start > len(s.msgs) {
		start = len(s.msgs)
	}
	end := start + limit
	if end > len(s.msgs) {
		end = len(s.msgs)
	}
	out := make([]core.Message, end-start)
	copy(out, s.msgs[start:end])
	return out, nil
}

func (s *testSessionStore) SessionStats(sessionID string) (map[string]any, error) {
	return map[string]any{"session_id": sessionID, "message_count": len(s.msgs)}, nil
}

func makeEngineForSlashTests(msgs []core.Message) *agent.Engine {
	reg := tools.NewRegistry()
	reg.Register(tools.NewSendMessageTool())
	store := &testSessionStore{msgs: msgs}
	return &agent.Engine{
		Registry:     reg,
		SessionStore: store,
	}
}

type scriptedClient struct {
	response string
	calls    int
}

func (c *scriptedClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	c.calls++
	return core.Message{Role: "assistant", Content: c.response}, nil
}

func TestHandleSlashCommandClear(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "hello"}})
	history := []core.Message{{Role: "assistant", Content: "hi"}}
	next, prompt, handled, err := handleSlashCommand("/clear", "s1", "sp", history, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	if next != nil {
		t.Fatalf("expected nil history after clear, got %v", next)
	}
	if prompt != "sp" {
		t.Fatalf("prompt changed unexpectedly: %q", prompt)
	}
}

func TestHandleSlashCommandReload(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}})
	next, _, handled, err := handleSlashCommand("/reload", "s1", "sp", nil, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	if len(next) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(next))
	}
}

func TestHandleSlashCommandStats(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "a"}})
	_, _, handled, err := handleSlashCommand("/stats", "s1", "sp", nil, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
}

func TestHandleSlashCommandNewAndResume(t *testing.T) {
	store := &testSessionStore{bySession: map[string][]core.Message{
		"old": {{Role: "user", Content: "stored"}},
	}}
	reg := tools.NewRegistry()
	eng := &agent.Engine{Registry: reg, SessionStore: store}
	state := &chatState{SessionID: "s1", SystemPrompt: "sp", History: []core.Message{{Role: "user", Content: "live"}}}
	handled, err := handleSlashCommandState(context.Background(), "/new next", state, eng)
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if state.SessionID != "next" || len(state.History) != 0 {
		t.Fatalf("new state = %+v", state)
	}
	handled, err = handleSlashCommandState(context.Background(), "/resume old", state, eng)
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if state.SessionID != "old" || len(state.History) != 1 || state.History[0].Content != "stored" {
		t.Fatalf("resume state = %+v", state)
	}
}

func TestHandleSlashCommandUndoAndCompress(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	state := &chatState{SessionID: "s1", SystemPrompt: "sp", History: []core.Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}}
	_, err := handleSlashCommandState(context.Background(), "/undo", state, eng)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.History) != 2 || state.History[1].Content != "two" {
		t.Fatalf("undo history = %+v", state.History)
	}
	state.History = []core.Message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
		{Role: "user", Content: "c"},
		{Role: "assistant", Content: "d"},
	}
	_, err = handleSlashCommandState(context.Background(), "/compress 2", state, eng)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.History) != 3 || !strings.Contains(state.History[0].Content, "Context summary") {
		t.Fatalf("compress history = %+v", state.History)
	}
}

func TestHandleSlashCommandToolsShow(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	line := captureStdout(t, func() {
		_, _ = handleSlashCommandState(context.Background(), "/TOOLS SHOW send_message", &chatState{SessionID: "s1"}, eng)
	})
	if !strings.Contains(line, `"send_message"`) {
		t.Fatalf("expected send_message schema, got %s", line)
	}
}

func TestHandleSlashCommandCaseInsensitiveRootCommands(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}

	if handled, err := handleSlashCommandState(context.Background(), "/SESSION", state, eng); err != nil || !handled {
		t.Fatalf("session handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/NEW next", state, eng); err != nil || !handled {
		t.Fatalf("new handled=%v err=%v", handled, err)
	}
	if state.SessionID != "next" {
		t.Fatalf("sessionID=%q", state.SessionID)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/MODEL", state, eng); err != nil || !handled {
		t.Fatalf("model handled=%v err=%v", handled, err)
	}
}

func TestHandleSlashCommandRetry(t *testing.T) {
	client := &scriptedClient{response: "retried"}
	eng := makeEngineForSlashTests(nil)
	eng.Client = client
	state := &chatState{SessionID: "s1", SystemPrompt: "sp", History: []core.Message{
		{Role: "user", Content: "try me"},
		{Role: "assistant", Content: "old"},
	}}
	_, err := handleSlashCommandState(context.Background(), "/retry", state, eng)
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	if got := state.History[len(state.History)-1].Content; got != "retried" {
		t.Fatalf("last content=%q", got)
	}
}

func TestPrintCLIEnvelopeSuccess(t *testing.T) {
	line := captureStdout(t, func() {
		printCLIEnvelope(true, map[string]any{"session_id": "s1"}, "", "")
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["ok"] != true || out["api_version"] != "v1" || out["compat"] == "" {
		t.Fatalf("unexpected envelope: %+v", out)
	}
	if out["session_id"] != "s1" {
		t.Fatalf("missing payload: %+v", out)
	}
}

func TestPrintCLIEnvelopeError(t *testing.T) {
	line := captureStdout(t, func() {
		printCLIEnvelope(false, nil, "invalid_argument", "bad")
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["ok"] != false || out["api_version"] != "v1" || out["compat"] == "" {
		t.Fatalf("unexpected envelope: %+v", out)
	}
	errObj, _ := out["error"].(map[string]any)
	if errObj["code"] != "invalid_argument" || errObj["message"] != "bad" {
		t.Fatalf("unexpected error payload: %+v", out)
	}
}
