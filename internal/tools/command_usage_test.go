package tools

import "testing"

func TestUsageHelpersZH(t *testing.T) {
	if got := UsageZHOptionalN("/history"); got != "用法: /history [n]" {
		t.Fatalf("unexpected optional n usage: %s", got)
	}
	if got := UsageZHOptionalNPositive("/history"); got != "用法: /history [n]（n 必须是正整数）" {
		t.Fatalf("unexpected optional n positive usage: %s", got)
	}
	if got := UsageZHRequiredIndex("/pick"); got != "用法: /pick <index>" {
		t.Fatalf("unexpected required index usage: %s", got)
	}
	if got := UsageZHRequiredIndexPositive("/pick"); got != "用法: /pick <index>（index 必须是正整数）" {
		t.Fatalf("unexpected required index positive usage: %s", got)
	}
	if got := UsageZHActionIndexRange(7); got != "用法: /actions <index> (1..7)" {
		t.Fatalf("unexpected action range usage: %s", got)
	}
}

func TestUsageHelpersEN(t *testing.T) {
	if got := UsageENEither("/approve <approval_id>", "/deny <approval_id>"); got != "Usage: /approve <approval_id> or /deny <approval_id>" {
		t.Fatalf("unexpected either usage: %s", got)
	}
	if got := NotSupportedBySessionStoreEN("Show"); got != "_Show not supported by session store._" {
		t.Fatalf("unexpected not supported text: %s", got)
	}
}

func TestCLIMessageHelpers(t *testing.T) {
	if got := SessionStoreUnavailableEN(); got != "session store unavailable" {
		t.Fatalf("unexpected session store message: %s", got)
	}
	if got := SessionStoreNotSupportedZH("会话列表"); got != "当前会话存储不支持会话列表。" {
		t.Fatalf("unexpected not supported zh: %s", got)
	}
	if got := CLICancelNotSupportedZH(); got != "当前 CLI 模式不支持 /cancel；请使用 Ctrl+C 中断当前轮。" {
		t.Fatalf("unexpected cancel not supported zh: %s", got)
	}
}

func TestNotFoundHelpers(t *testing.T) {
	if got := NotFoundEN("tool", "send_message"); got != "tool not found: send_message" {
		t.Fatalf("unexpected not found en: %s", got)
	}
	if got := PendingApprovalNotFoundZH(); got != "未找到待处理审批" {
		t.Fatalf("unexpected pending approval zh: %s", got)
	}
	if got := SkillsDirectoryNotFoundEN(); got != "skills directory not found" {
		t.Fatalf("unexpected skills directory not found: %s", got)
	}
}
