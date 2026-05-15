package gateway

import (
	"errors"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

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
	if got := normalizeGatewayCommand("slack", "/RECOVER context"); got != "/recover context" {
		t.Fatalf("slash command should normalize recover, got=%q", got)
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
