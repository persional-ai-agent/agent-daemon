package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	compactN  int
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

func (s *testSessionStore) CompactSession(_ string, keepLastN int) (before int, after int, err error) {
	s.compactN++
	before = len(s.msgs)
	if keepLastN <= 0 || keepLastN >= before {
		return before, before, nil
	}
	after = keepLastN
	return before, after, nil
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

type sequenceClient struct {
	responses []core.Message
	errors    []error
	calls     int
}

func (c *sequenceClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	idx := c.calls
	c.calls++
	if idx < len(c.errors) && c.errors[idx] != nil {
		return core.Message{}, c.errors[idx]
	}
	if idx < len(c.responses) {
		return c.responses[idx], nil
	}
	return core.Message{Role: "assistant", Content: "ok"}, nil
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

func TestHandleSlashCommandUsageAndPersonality(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "a"}})
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	if handled, err := handleSlashCommandState(context.Background(), "/usage", state, eng); err != nil || !handled {
		t.Fatalf("usage handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/personality show", state, eng); err != nil || !handled {
		t.Fatalf("personality show handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/personality custom prompt", state, eng); err != nil || !handled {
		t.Fatalf("personality set handled=%v err=%v", handled, err)
	}
	if state.SystemPrompt != "custom prompt" {
		t.Fatalf("system prompt not updated: %q", state.SystemPrompt)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/personality reset", state, eng); err != nil || !handled {
		t.Fatalf("personality reset handled=%v err=%v", handled, err)
	}
	if strings.TrimSpace(state.SystemPrompt) == "custom prompt" {
		t.Fatalf("system prompt should be reset: %q", state.SystemPrompt)
	}
}

func TestHandleSlashCommandCancelAndStop(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	if handled, err := handleSlashCommandState(context.Background(), "/cancel", state, eng); err != nil || !handled {
		t.Fatalf("cancel handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/stop", state, eng); err != nil || !handled {
		t.Fatalf("stop handled=%v err=%v", handled, err)
	}
}

func TestHandleSlashCommandModelSetAndShow(t *testing.T) {
	workdir := t.TempDir()
	eng := makeEngineForSlashTests(nil)
	eng.Workdir = workdir
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}

	if handled, err := handleSlashCommandState(context.Background(), "/model openai:gpt-5-mini", state, eng); err != nil || !handled {
		t.Fatalf("model set handled=%v err=%v", handled, err)
	}
	provider, err := tools.GetGatewaySetting(workdir, "model_provider")
	if err != nil {
		t.Fatal(err)
	}
	modelName, err := tools.GetGatewaySetting(workdir, "model_name")
	if err != nil {
		t.Fatal(err)
	}
	if provider != "openai" || modelName != "gpt-5-mini" {
		t.Fatalf("unexpected model preference provider=%q model=%q", provider, modelName)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/model codex gpt-5-codex", state, eng); err != nil || !handled {
		t.Fatalf("model set pair handled=%v err=%v", handled, err)
	}
	provider, _ = tools.GetGatewaySetting(workdir, "model_provider")
	modelName, _ = tools.GetGatewaySetting(workdir, "model_name")
	if provider != "codex" || modelName != "gpt-5-codex" {
		t.Fatalf("unexpected model preference provider=%q model=%q", provider, modelName)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/model", state, eng); err != nil || !handled {
		t.Fatalf("model show handled=%v err=%v", handled, err)
	}
}

func TestHandleSlashCommandSetHomeAndTargets(t *testing.T) {
	workdir := t.TempDir()
	eng := makeEngineForSlashTests(nil)
	eng.Workdir = workdir
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	if handled, err := handleSlashCommandState(context.Background(), "/sethome telegram:100", state, eng); err != nil || !handled {
		t.Fatalf("sethome handled=%v err=%v", handled, err)
	}
	if v := os.Getenv("TELEGRAM_HOME_CHANNEL"); v != "100" {
		t.Fatalf("env TELEGRAM_HOME_CHANNEL=%q want=100", v)
	}
	if err := tools.UpsertChannelDirectory(workdir, tools.ChannelDirectoryEntry{
		Platform: "telegram",
		ChatID:   "100",
		UserID:   "u1",
	}); err != nil {
		t.Fatal(err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/targets telegram", state, eng); err != nil || !handled {
		t.Fatalf("targets handled=%v err=%v", handled, err)
	}
}

func TestHandleSlashCommandContinuityAndIdentity(t *testing.T) {
	workdir := t.TempDir()
	eng := makeEngineForSlashTests(nil)
	eng.Workdir = workdir
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}

	if handled, err := handleSlashCommandState(context.Background(), "/continuity user_name", state, eng); err != nil || !handled {
		t.Fatalf("continuity set handled=%v err=%v", handled, err)
	}
	mode, err := tools.GetGatewaySetting(workdir, "continuity_mode")
	if err != nil {
		t.Fatal(err)
	}
	if mode != "user_name" {
		t.Fatalf("continuity_mode=%q", mode)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/setid telegram u1 gid-1", state, eng); err != nil || !handled {
		t.Fatalf("setid handled=%v err=%v", handled, err)
	}
	globalID, err := tools.ResolveGatewayIdentity(workdir, "telegram", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if globalID != "gid-1" {
		t.Fatalf("globalID=%q", globalID)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/whoami telegram u1", state, eng); err != nil || !handled {
		t.Fatalf("whoami handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/unsetid telegram u1", state, eng); err != nil || !handled {
		t.Fatalf("unsetid handled=%v err=%v", handled, err)
	}
	globalID, err = tools.ResolveGatewayIdentity(workdir, "telegram", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if globalID != "" {
		t.Fatalf("expected empty globalID after unset, got %q", globalID)
	}
}

func TestHandleSlashCommandResolve(t *testing.T) {
	workdir := t.TempDir()
	eng := makeEngineForSlashTests(nil)
	eng.Workdir = workdir
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	if err := tools.SetGatewaySetting(workdir, "continuity_mode", "user_name"); err != nil {
		t.Fatal(err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/resolve telegram group 1001 u1 Alice", state, eng); err != nil || !handled {
		t.Fatalf("resolve handled=%v err=%v", handled, err)
	}
	if handled, err := handleSlashCommandState(context.Background(), "/resolve telegram group 1001", state, eng); err != nil || !handled {
		t.Fatalf("resolve invalid handled=%v err=%v", handled, err)
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

func TestHandleSlashCommandRetryNoMessage(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	line := captureStdout(t, func() {
		_, _ = handleSlashCommandState(context.Background(), "/retry", state, eng)
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["ok"] != false {
		t.Fatalf("expected error envelope, got: %+v", out)
	}
	errObj, _ := out["error"].(map[string]any)
	if errObj["code"] != "not_available" || errObj["message"] != tools.RetryNotAvailableZH() {
		t.Fatalf("unexpected retry no-message payload: %+v", out)
	}
}

func TestHandleSlashCommandStrictArgumentValidation(t *testing.T) {
	eng := makeEngineForSlashTests(nil)
	state := &chatState{SessionID: "s1", SystemPrompt: "sp"}
	cases := []struct {
		cmd  string
		want string
	}{
		{cmd: "/new a b", want: tools.UsageZH(tools.CommandNewResetUsage)},
		{cmd: "/reset a", want: tools.UsageZH(tools.CommandNewResetUsage)},
		{cmd: "/history 1 2", want: tools.UsageZHOptionalN("/history")},
		{cmd: "/sessions 1 2", want: tools.UsageZHOptionalN("/sessions")},
		{cmd: "/show s1 -1", want: tools.UsageZH(tools.CommandShowUsage)},
		{cmd: "/show s1 0 0", want: tools.UsageZH(tools.CommandShowUsage)},
		{cmd: "/show s1 0 1 x", want: tools.UsageZH(tools.CommandShowUsage)},
	}
	for _, tc := range cases {
		line := captureStdout(t, func() {
			_, _ = handleSlashCommandState(context.Background(), tc.cmd, state, eng)
		})
		var out map[string]any
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			t.Fatalf("invalid json for %q: %v", tc.cmd, err)
		}
		if out["ok"] != false {
			t.Fatalf("expected error envelope for %q: %+v", tc.cmd, out)
		}
		errObj, _ := out["error"].(map[string]any)
		if errObj["code"] != "invalid_argument" || errObj["message"] != tc.want {
			t.Fatalf("unexpected error for %q: %+v", tc.cmd, out)
		}
	}
}

func TestRunWithContextRecoveryCompressRetry(t *testing.T) {
	store := &testSessionStore{msgs: []core.Message{{Role: "user", Content: "u1"}, {Role: "assistant", Content: "a1"}}}
	reg := tools.NewRegistry()
	client := &sequenceClient{
		errors: []error{
			fmt.Errorf("openai api error (400): {\"error\":{\"type\":\"exceed_context_size_error\"}}"),
			fmt.Errorf("openai api error (400): {\"error\":{\"type\":\"exceed_context_size_error\"}}"),
			fmt.Errorf("openai api error (400): {\"error\":{\"type\":\"exceed_context_size_error\"}}"),
			nil,
		},
		responses: []core.Message{{}, {}, {}, {Role: "assistant", Content: "ok-after-compress"}},
	}
	eng := &agent.Engine{Registry: reg, SessionStore: store, Client: client}
	state := &chatState{SessionID: "s1", SystemPrompt: "sp", History: []core.Message{{Role: "user", Content: "x"}, {Role: "assistant", Content: "y"}}}
	res, err := runWithContextRecovery(context.Background(), eng, state, "ping")
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalResponse != "ok-after-compress" {
		t.Fatalf("unexpected final response: %q", res.FinalResponse)
	}
	if store.compactN == 0 {
		t.Fatal("expected compact session to be called")
	}
}

func TestRunWithContextRecoveryFallbackToNewSession(t *testing.T) {
	store := &testSessionStore{msgs: []core.Message{{Role: "user", Content: "u1"}, {Role: "assistant", Content: "a1"}}}
	reg := tools.NewRegistry()
	client := &sequenceClient{
		errors: []error{
			fmt.Errorf("request exceeds the available context size"),
			fmt.Errorf("request exceeds the available context size"),
			fmt.Errorf("request exceeds the available context size"),
			fmt.Errorf("request exceeds the available context size"),
			fmt.Errorf("request exceeds the available context size"),
			fmt.Errorf("request exceeds the available context size"),
			nil,
		},
		responses: []core.Message{{}, {}, {}, {}, {}, {}, {Role: "assistant", Content: "ok-new-session"}},
	}
	eng := &agent.Engine{Registry: reg, SessionStore: store, Client: client}
	state := &chatState{SessionID: "s1", SystemPrompt: "sp", History: []core.Message{{Role: "user", Content: "x"}, {Role: "assistant", Content: "y"}}}
	res, err := runWithContextRecovery(context.Background(), eng, state, "ping")
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalResponse != "ok-new-session" {
		t.Fatalf("unexpected final response: %q", res.FinalResponse)
	}
	if state.SessionID == "s1" {
		t.Fatal("expected session switched to a new id")
	}
}

func TestRunChatFirstMessageEOFReturnsNil(t *testing.T) {
	client := &scriptedClient{response: "ok"}
	eng := makeEngineForSlashTests(nil)
	eng.Client = client

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	if err := RunChat(context.Background(), eng, "s1", "测试", ""); err != nil {
		t.Fatalf("RunChat returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
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

func TestPrintSlashHelpUsesSharedCatalog(t *testing.T) {
	line := captureStdout(t, func() {
		printSlashHelp()
	})
	if !strings.Contains(line, "/help | /commands") {
		t.Fatalf("unexpected help output: %s", line)
	}
	if !strings.Contains(line, "/quit | /exit") {
		t.Fatalf("missing quit help output: %s", line)
	}
}
