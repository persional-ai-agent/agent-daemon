package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type gatewayStatusStoreStub struct{}

func (gatewayStatusStoreStub) AppendMessage(string, core.Message) error { return nil }
func (gatewayStatusStoreStub) LoadMessages(string, int) ([]core.Message, error) {
	return nil, nil
}
func (gatewayStatusStoreStub) SessionStats(sessionID string) (map[string]any, error) {
	return map[string]any{"session_id": sessionID, "message_count": 42}, nil
}

type gatewaySessionMapStore struct {
	bySession map[string][]core.Message
}

func (s gatewaySessionMapStore) AppendMessage(string, core.Message) error { return nil }
func (s gatewaySessionMapStore) LoadMessages(sessionID string, _ int) ([]core.Message, error) {
	msgs := s.bySession[sessionID]
	out := make([]core.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

type gatewayAdapterStub struct {
	name string
}

func (a *gatewayAdapterStub) Name() string                       { return a.name }
func (a *gatewayAdapterStub) Connect(_ context.Context) error    { return nil }
func (a *gatewayAdapterStub) Disconnect(_ context.Context) error { return nil }
func (a *gatewayAdapterStub) Send(_ context.Context, _, _, _ string) (SendResult, error) {
	return SendResult{Success: true}, nil
}
func (a *gatewayAdapterStub) EditMessage(_ context.Context, _, _, _ string) error { return nil }
func (a *gatewayAdapterStub) SendTyping(_ context.Context, _ string) error        { return nil }
func (a *gatewayAdapterStub) OnMessage(_ context.Context, _ MessageHandler)       {}

func TestNormalizeGatewayCommandForYuanbao(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "批准", want: "/approve"},
		{input: "批准 ap-1", want: "/approve ap-1"},
		{input: "同意", want: "/approve"},
		{input: "通过", want: "/approve"},
		{input: "拒绝", want: "/deny"},
		{input: "拒绝 ap-2", want: "/deny ap-2"},
		{input: "驳回", want: "/deny"},
		{input: "状态", want: "/status"},
		{input: "待审批", want: "/pending"},
		{input: "审批", want: "/approvals"},
		{input: "帮助", want: "/help"},
		{input: "/status", want: "/status"},
		{input: "/STATUS", want: "/status"},
		{input: "/APPROVAL", want: "/approvals"},
		{input: "/PENDINGS", want: "/pending"},
		{input: "/STOP", want: "/cancel"},
		{input: "/APPROVE AP-3", want: "/approve AP-3"},
		{input: "/Q", want: "/queue"},
		{input: "APPROVE ap-4", want: "/approve ap-4"},
		{input: "STATUS", want: "/status"},
		{input: "approval", want: "/approvals"},
		{input: "pendings", want: "/pending"},
		{input: "abort", want: "/cancel"},
		{input: "custom text", want: "custom text"},
	}
	for _, tt := range tests {
		got := normalizeGatewayCommand("yuanbao", tt.input)
		if got != tt.want {
			t.Fatalf("normalize(%q)=%q want=%q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeGatewayCommandNonYuanbao(t *testing.T) {
	if got := normalizeGatewayCommand("slack", "状态"); got != "状态" {
		t.Fatalf("non-yuanbao should keep input, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/STATUS"); got != "/status" {
		t.Fatalf("slash command should normalize case, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/NEW demo"); got != "/new demo" {
		t.Fatalf("slash command should normalize new, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RESET"); got != "/reset" {
		t.Fatalf("slash command should normalize reset, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SESSION s-1"); got != "/session s-1" {
		t.Fatalf("slash command should normalize session, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/HISTORY 5"); got != "/history 5" {
		t.Fatalf("slash command should normalize history, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SHOW s1 0 20"); got != "/show s1 0 20" {
		t.Fatalf("slash command should normalize show, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/NEXT"); got != "/next" {
		t.Fatalf("slash command should normalize next, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/PREV"); got != "/prev" {
		t.Fatalf("slash command should normalize prev, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SESSIONS 5"); got != "/sessions 5" {
		t.Fatalf("slash command should normalize sessions, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/PICK 2"); got != "/pick 2" {
		t.Fatalf("slash command should normalize pick, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/STATS s-1"); got != "/stats s-1" {
		t.Fatalf("slash command should normalize stats, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RESUME sess-1"); got != "/resume sess-1" {
		t.Fatalf("slash command should normalize resume, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RECOVER context"); got != "/recover context" {
		t.Fatalf("slash command should normalize recover, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RETRY"); got != "/retry" {
		t.Fatalf("slash command should normalize retry, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/UNDO"); got != "/undo" {
		t.Fatalf("slash command should normalize undo, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/CLEAR"); got != "/clear" {
		t.Fatalf("slash command should normalize clear, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RELOAD"); got != "/reload" {
		t.Fatalf("slash command should normalize reload, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SAVE out.json"); got != "/save out.json" {
		t.Fatalf("slash command should normalize save, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SETHOME telegram 123"); got != "/sethome telegram 123" {
		t.Fatalf("slash command should normalize sethome, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SETHOME telegram:group:123"); got != "/sethome telegram:group:123" {
		t.Fatalf("slash command should keep sethome target token, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SKILLS demo"); got != "/skills demo" {
		t.Fatalf("slash command should normalize skills, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/TOOLS SHOW send_message"); got != "/tools SHOW send_message" {
		t.Fatalf("slash command should normalize tools root, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/USAGE"); got != "/usage" {
		t.Fatalf("slash command should normalize usage, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/USAGE s-123"); got != "/usage s-123" {
		t.Fatalf("slash command should normalize usage with arg, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/MODEL"); got != "/model" {
		t.Fatalf("slash command should normalize model, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/PERSONALITY show"); got != "/personality show" {
		t.Fatalf("slash command should normalize personality, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/TARGETS telegram"); got != "/targets telegram" {
		t.Fatalf("slash command should normalize targets, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/WHOAMI"); got != "/whoami" {
		t.Fatalf("slash command should normalize whoami, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/RESOLVE telegram group 1001 u1"); got != "/resolve telegram group 1001 u1" {
		t.Fatalf("slash command should normalize resolve, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/CONTINUITY user_name"); got != "/continuity user_name" {
		t.Fatalf("slash command should normalize continuity, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/SETID user-1"); got != "/setid user-1" {
		t.Fatalf("slash command should normalize setid, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/UNSETID"); got != "/unsetid" {
		t.Fatalf("slash command should normalize unsetid, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "/COMPRESS 30"); got != "/compress 30" {
		t.Fatalf("slash command should normalize compress, got=%q", got)
	}
	if got := normalizeGatewayCommand("slack", "approve ap-7"); got != "/approve ap-7" {
		t.Fatalf("builtin command should canonicalize, got=%q", got)
	}
}

func TestGatewayUserInputWithMedia(t *testing.T) {
	in := MessageEvent{
		Text:      "分析这张图",
		MediaURLs: []string{"https://cdn.example.com/a.png"},
	}
	got := gatewayUserInput(in)
	if got == "" || got == "分析这张图" {
		t.Fatalf("unexpected input=%q", got)
	}
	if want := "Media URLs:"; !strings.Contains(got, want) {
		t.Fatalf("missing %q in %q", want, got)
	}
}

func TestParseGatewayCommand(t *testing.T) {
	got := parseGatewayCommand("slack", "APPROVE ap-1")
	if !got.isSlash || got.head != "/approve" || len(got.args) != 1 || got.args[0] != "ap-1" {
		t.Fatalf("unexpected parsed command: %+v", got)
	}

	got = parseGatewayCommand("yuanbao", "批准 ap-2")
	if !got.isSlash || got.head != "/approve" || len(got.args) != 1 || got.args[0] != "ap-2" {
		t.Fatalf("unexpected yuanbao parsed command: %+v", got)
	}

	got = parseGatewayCommand("slack", "custom text")
	if got.isSlash || got.head != "custom" || len(got.args) != 1 || got.args[0] != "text" {
		t.Fatalf("unexpected plain parsed command: %+v", got)
	}
}

func TestParseApprovalManageCommand(t *testing.T) {
	args, usage := parseApprovalManageCommand("/grant", []string{"pattern", "tool_*", "3600"})
	if usage != "" {
		t.Fatalf("unexpected usage error: %s", usage)
	}
	if args["scope"] != "pattern" || args["pattern"] != "tool_*" || args["ttl_seconds"] != 3600 {
		t.Fatalf("unexpected grant pattern args: %+v", args)
	}

	args, usage = parseApprovalManageCommand("/revoke", []string{"pattern", "tool_*"})
	if usage != "" {
		t.Fatalf("unexpected revoke usage error: %s", usage)
	}
	if args["scope"] != "pattern" || args["pattern"] != "tool_*" {
		t.Fatalf("unexpected revoke pattern args: %+v", args)
	}

	args, usage = parseApprovalManageCommand("/grant", []string{"bad-ttl"})
	if args != nil || usage == "" {
		t.Fatalf("expected usage error for bad grant ttl, got args=%+v usage=%q", args, usage)
	}

	args, usage = parseApprovalManageCommand("/grant", []string{"pattern"})
	if args != nil || usage != GatewayGrantPatternOrRevokePatternUsage() {
		t.Fatalf("expected pattern usage error, got args=%+v usage=%q", args, usage)
	}
}

func TestIsGatewayContextLimitError(t *testing.T) {
	cases := []struct {
		errText string
		want    bool
	}{
		{"openai api error (400): {\"error\":{\"type\":\"exceed_context_size_error\"}}", true},
		{"request (32985 tokens) exceeds the available context size (32768 tokens)", true},
		{"network timeout", false},
	}
	for _, tc := range cases {
		got := isGatewayContextLimitError(errors.New(tc.errText))
		if got != tc.want {
			t.Fatalf("err=%q got=%v want=%v", tc.errText, got, tc.want)
		}
	}
}

func TestCompactGatewayHistory(t *testing.T) {
	history := []core.Message{
		{Role: "user", Content: "1"},
		{Role: "assistant", Content: "2"},
		{Role: "user", Content: "3"},
		{Role: "assistant", Content: "4"},
	}
	got := compactGatewayHistory(history, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Content != "3" || got[1].Content != "4" {
		t.Fatalf("unexpected compacted history: %+v", got)
	}
}

func TestCompactGatewayHistoryTailDefaults(t *testing.T) {
	history := make([]core.Message, 0, 30)
	for i := 0; i < 30; i++ {
		history = append(history, core.Message{Role: "user", Content: itoa(i)})
	}
	got := compactGatewayHistory(history, 0)
	if len(got) != 20 {
		t.Fatalf("expected default compact size=20, got=%d", len(got))
	}
	if got[0].Content != "10" || got[19].Content != "29" {
		t.Fatalf("unexpected compact tail window: first=%q last=%q", got[0].Content, got[19].Content)
	}
}

func TestRenderGatewayTargets(t *testing.T) {
	text := renderGatewayTargets([]map[string]any{
		{"platform": "telegram", "chat_id": "100", "target": "telegram:100", "home_target": "100", "user_id": "u1"},
		{"platform": "discord", "chat_id": "c1", "target": "discord:c1"},
	}, "telegram")
	if !strings.Contains(text, "telegram:100") {
		t.Fatalf("missing target row: %q", text)
	}
	if strings.Contains(text, "discord:c1") {
		t.Fatalf("filter should hide discord row: %q", text)
	}
}

func TestAutoGlobalIdentity(t *testing.T) {
	if got := tools.AutoGlobalIdentity("off", "u1", "Alice"); got != "" {
		t.Fatalf("off mode should not map, got=%q", got)
	}
	if got := tools.AutoGlobalIdentity("user_id", "u1", "Alice"); got != "uid:u1" {
		t.Fatalf("user_id mode mismatch: %q", got)
	}
	if got := tools.AutoGlobalIdentity("user_name", "u1", "Alice Bob"); got != "uname:alice_bob" {
		t.Fatalf("user_name mode mismatch: %q", got)
	}
}

func TestParseGatewayModelSpec(t *testing.T) {
	spec, err := tools.ParseGatewayModelSpecArgs([]string{"openai:gpt-5"})
	if err != nil || spec.Provider != "openai" || spec.Model != "gpt-5" {
		t.Fatalf("unexpected parse result: spec=%+v err=%v", spec, err)
	}
	spec, err = tools.ParseGatewayModelSpecArgs([]string{"codex", "gpt-5-codex"})
	if err != nil || spec.Provider != "codex" || spec.Model != "gpt-5-codex" {
		t.Fatalf("unexpected parse result: spec=%+v err=%v", spec, err)
	}
	if _, err = tools.ParseGatewayModelSpecArgs([]string{"invalid"}); err == nil {
		t.Fatal("expected invalid parse for single token without colon")
	}
	if _, err = tools.ParseGatewayModelSpecArgs([]string{"", "x"}); err == nil {
		t.Fatal("expected invalid parse for empty provider")
	}
}

func TestAllowByInteractionPolicy(t *testing.T) {
	w := &sessionWorker{adapter: &gatewayAdapterStub{name: "telegram"}}
	policy := tools.GatewayInteractionPolicy{
		MentionRequired: true,
		GroupDMPolicy:   "both",
		MentionKeywords: []string{"@agent"},
	}
	if w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "hello", IsCommand: false}) {
		t.Fatal("group free text without mention should be blocked when mention is required")
	}
	if !w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "@agent hello", IsCommand: false}) {
		t.Fatal("group text with mention should pass")
	}
	if !w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "/status", IsCommand: true}) {
		t.Fatal("slash command should pass policy gate")
	}
	policy.GroupDMPolicy = "dm_only"
	if w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "@agent hello", IsCommand: false}) {
		t.Fatal("group message should be blocked in dm_only policy")
	}
	policy.GroupDMPolicy = "both"
	policy.IgnoredChannels = []string{"telegram:g1"}
	if w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "@agent hello", IsCommand: false}) {
		t.Fatal("ignored channel should be blocked")
	}
	policy.IgnoredChannels = nil
	policy.FreeResponseChannels = []string{"telegram:g1"}
	if !w.allowByInteractionPolicy(policy, MessageEvent{ChatType: "group", ChatID: "g1", Text: "hello", IsCommand: false}) {
		t.Fatal("free response channel should bypass mention requirement")
	}
}

func TestResolveMappedSessionIDWithAutoMode(t *testing.T) {
	t.Setenv("AGENT_GATEWAY_CONTINUITY", "user_id")
	workdir := t.TempDir()
	r := &Runner{identityStore: newIdentityStore(workdir), engine: &agent.Engine{Workdir: workdir}}
	got := r.resolveMappedSessionID("telegram", "u9", "Alice")
	want := BuildSessionKey("global", "user", "uid:u9")
	if got != want {
		t.Fatalf("auto continuity mapping mismatch: got=%q want=%q", got, want)
	}
}

func TestContinuityModeFromRaw(t *testing.T) {
	if got := tools.NormalizeContinuityMode("name"); got != "user_name" {
		t.Fatalf("name alias mismatch: %q", got)
	}
	if got := tools.NormalizeContinuityMode("id"); got != "user_id" {
		t.Fatalf("id alias mismatch: %q", got)
	}
	if got := tools.NormalizeContinuityMode("xxx"); got != "off" {
		t.Fatalf("unknown alias should be off: %q", got)
	}
}

func TestLatestUserInputFromMessages(t *testing.T) {
	history := []core.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "/status"},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: " retry this "},
	}
	got := latestUserInputFromMessages(history)
	if got != "retry this" {
		t.Fatalf("unexpected latest user input: %q", got)
	}
}

func TestRemoveLastTurnFromMessages(t *testing.T) {
	history := []core.Message{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "u2"},
		{Role: "assistant", Content: "a2"},
	}
	next, removed := removeLastTurnFromMessages(history)
	if removed != 2 {
		t.Fatalf("unexpected removed count: %d", removed)
	}
	if len(next) != 2 || next[0].Content != "u1" || next[1].Content != "a1" {
		t.Fatalf("unexpected remaining messages: %+v", next)
	}
}

func TestRenderGatewayHistory(t *testing.T) {
	history := []core.Message{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "u2"},
	}
	got := renderGatewayHistory(history, 2)
	if !strings.Contains(got, "Recent history:") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "2. [assistant] a1") || !strings.Contains(got, "3. [user] u2") {
		t.Fatalf("unexpected history output: %q", got)
	}
}

func TestRenderGatewayStats(t *testing.T) {
	stats := map[string]any{
		"session_id":    "s1",
		"message_count": 12,
	}
	got := renderGatewayStats("s1", stats)
	if !strings.Contains(got, "Session stats: s1") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "message_count: 12") {
		t.Fatalf("missing message_count: %q", got)
	}
	if !strings.Contains(got, "session_id: s1") {
		t.Fatalf("missing session_id: %q", got)
	}
}

func TestRenderGatewayShow(t *testing.T) {
	msgs := []core.Message{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
	}
	got := renderGatewayShow("s1", 10, 20, msgs)
	if !strings.Contains(got, "Session messages: s1") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "offset=10 limit=20 count=2") {
		t.Fatalf("missing meta: %q", got)
	}
	if !strings.Contains(got, "11. [user] u1") || !strings.Contains(got, "12. [assistant] a1") {
		t.Fatalf("missing rows: %q", got)
	}
}

func TestParseGatewayShowArgs(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		defSID     string
		wantSID    string
		wantOffset int
		wantLimit  int
		wantErr    bool
	}{
		{name: "default", args: nil, defSID: "s1", wantSID: "s1", wantOffset: 0, wantLimit: 20},
		{name: "explicit session", args: []string{"s2"}, defSID: "s1", wantSID: "s2", wantOffset: 0, wantLimit: 20},
		{name: "session offset limit", args: []string{"s2", "10", "30"}, defSID: "s1", wantSID: "s2", wantOffset: 10, wantLimit: 30},
		{name: "offset limit with default session", args: []string{"5", "15"}, defSID: "s1", wantSID: "s1", wantOffset: 5, wantLimit: 15},
		{name: "offset only with default session", args: []string{"7"}, defSID: "s1", wantSID: "s1", wantOffset: 7, wantLimit: 20},
		{name: "negative offset", args: []string{"-1"}, defSID: "s1", wantErr: true},
		{name: "bad limit", args: []string{"s2", "0", "0"}, defSID: "s1", wantErr: true},
		{name: "too many args", args: []string{"s2", "1", "2", "3"}, defSID: "s1", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sid, offset, limit, err := parseGatewayShowArgs(tc.args, tc.defSID)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got sid=%q offset=%d limit=%d", sid, offset, limit)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sid != tc.wantSID || offset != tc.wantOffset || limit != tc.wantLimit {
				t.Fatalf("got sid=%q offset=%d limit=%d want sid=%q offset=%d limit=%d", sid, offset, limit, tc.wantSID, tc.wantOffset, tc.wantLimit)
			}
		})
	}
}

func TestRenderGatewaySessions(t *testing.T) {
	items := []map[string]any{
		{"session_id": "s1", "last_seen": "2026-05-15T10:00:00Z"},
		{"session_id": "s2", "last_seen": "2026-05-15T09:00:00Z"},
	}
	got := renderGatewaySessions("s2", items)
	if !strings.Contains(got, "Recent sessions:") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "1. s1 last_seen=2026-05-15T10:00:00Z") {
		t.Fatalf("missing first row: %q", got)
	}
	if !strings.Contains(got, "2. s2 last_seen=2026-05-15T09:00:00Z [active]") {
		t.Fatalf("missing active row: %q", got)
	}
}

func TestSetAndGetLastSessionIDs(t *testing.T) {
	w := &sessionWorker{}
	w.setLastSessionIDs("u1", []string{"s1", "s2"})
	got := w.getLastSessionIDs("u1")
	if len(got) != 2 || got[0] != "s1" || got[1] != "s2" {
		t.Fatalf("unexpected cached session ids: %+v", got)
	}
	got[0] = "changed"
	again := w.getLastSessionIDs("u1")
	if again[0] != "s1" {
		t.Fatalf("expected copy semantics, got %+v", again)
	}
}

func TestGatewayStatusTextUsesLastUserPairingAndStats(t *testing.T) {
	w := &sessionWorker{
		key:             "route:s1",
		activeSessionID: "active:s1",
		adapter:         &gatewayAdapterStub{name: "slack"},
		engine:          &agent.Engine{SessionStore: gatewayStatusStoreStub{}},
		runner: &Runner{
			pairings: map[string]map[string]bool{
				"slack": {"u-1": true},
			},
		},
	}
	w.setLastUserID("u-1")
	got := w.gatewayStatusText()
	if !strings.Contains(got, "paired: yes") {
		t.Fatalf("expected paired yes, got: %q", got)
	}
	if !strings.Contains(got, "message_count: 42") {
		t.Fatalf("expected message_count in status, got: %q", got)
	}
	if !strings.Contains(got, "continuity_mode: off") {
		t.Fatalf("expected continuity mode in status, got: %q", got)
	}
	if !strings.Contains(got, "route_session: route:s1") || !strings.Contains(got, "active_session: active:s1") {
		t.Fatalf("expected route/active session lines, got: %q", got)
	}
}

func TestActivateSessionSyncsCursorAndLastUserInput(t *testing.T) {
	w := &sessionWorker{
		key: "route:s1",
		engine: &agent.Engine{
			SessionStore: gatewaySessionMapStore{
				bySession: map[string][]core.Message{
					"s2": {
						{Role: "user", Content: "hello"},
						{Role: "assistant", Content: "ok"},
						{Role: "user", Content: "latest input"},
					},
				},
			},
		},
	}
	w.activateSession("s2", "u1")
	if got := w.currentSessionID(); got != "s2" {
		t.Fatalf("active session mismatch: %q", got)
	}
	sid, offset, limit := w.showCursor("u1")
	if sid != "s2" || offset != 0 || limit != 20 {
		t.Fatalf("show cursor mismatch: sid=%q offset=%d limit=%d", sid, offset, limit)
	}
	if got := w.getLastUserInput("u1"); got != "latest input" {
		t.Fatalf("last user input mismatch: %q", got)
	}
}

func TestSaveGatewayHistory(t *testing.T) {
	dir := t.TempDir()
	history := []core.Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "world"}}
	path, err := saveGatewayHistory(dir, "s1", history, "")
	if err != nil {
		t.Fatalf("saveGatewayHistory error: %v", err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("expected path under temp dir, got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["session_id"] != "s1" {
		t.Fatalf("unexpected session_id: %+v", out)
	}
	path2, err := saveGatewayHistory(dir, "s2", history, filepath.Join("exports", "x.json"))
	if err != nil {
		t.Fatalf("saveGatewayHistory custom path error: %v", err)
	}
	if !strings.HasSuffix(path2, filepath.Join("exports", "x.json")) {
		t.Fatalf("unexpected custom path: %q", path2)
	}
}

func TestRenderGatewayToolsList(t *testing.T) {
	got := renderGatewayToolsList([]string{"b_tool", "a_tool"})
	if !strings.Contains(got, "Tools (2):") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "1. a_tool") || !strings.Contains(got, "2. b_tool") {
		t.Fatalf("unexpected ordering: %q", got)
	}
}
