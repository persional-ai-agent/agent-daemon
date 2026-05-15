package gateway

import (
	"strings"
	"testing"
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
		{input: "/APPROVE AP-3", want: "/approve AP-3"},
		{input: "APPROVE ap-4", want: "/approve ap-4"},
		{input: "STATUS", want: "/status"},
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
