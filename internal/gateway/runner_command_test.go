package gateway

import "testing"

func TestNormalizeGatewayCommandForYuanbao(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "批准", want: "/approve"},
		{input: "同意", want: "/approve"},
		{input: "通过", want: "/approve"},
		{input: "拒绝", want: "/deny"},
		{input: "驳回", want: "/deny"},
		{input: "状态", want: "/status"},
		{input: "待审批", want: "/pending"},
		{input: "审批", want: "/approvals"},
		{input: "帮助", want: "/help"},
		{input: "/status", want: "/status"},
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
}
