package tools

import (
	"fmt"
	"testing"
)

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
	if got := InvalidActionIndexZH(); got != "无效的动作索引" {
		t.Fatalf("unexpected invalid action index zh: %s", got)
	}
	if got := WorkflowCommandsEmptyEN(); got != "workflow commands empty" {
		t.Fatalf("unexpected workflow commands empty: %s", got)
	}
	if got := AccessDeniedEN(); got != "_Access denied._" {
		t.Fatalf("unexpected access denied: %s", got)
	}
	if got := PairingUnavailableEN(); got != "_Pairing unavailable._" {
		t.Fatalf("unexpected pairing unavailable: %s", got)
	}
	if got := PairSucceededEN(); got != "_Paired successfully._" {
		t.Fatalf("unexpected pair succeeded: %s", got)
	}
	if got := UnpairUnavailableEN(); got != "_Unpair unavailable._" {
		t.Fatalf("unexpected unpair unavailable: %s", got)
	}
	if got := UnpairedEN(); got != "_Unpaired._" {
		t.Fatalf("unexpected unpaired: %s", got)
	}
	if got := NotPairedEN(); got != "_Not paired._" {
		t.Fatalf("unexpected not paired: %s", got)
	}
	if got := IdentityStoreUnavailableEN(); got != "_Identity store unavailable._" {
		t.Fatalf("unexpected identity store unavailable: %s", got)
	}
	if got := SessionsListRequiredForPickEN(); got != "_No /sessions list available. Run /sessions first._" {
		t.Fatalf("unexpected sessions list required: %s", got)
	}
	if got := PickIndexOutOfRangeEN(8); got != "_Pick index out of range: max=8_" {
		t.Fatalf("unexpected pick out of range: %s", got)
	}
	if got := CancelledEN(); got != "_Cancelled._" {
		t.Fatalf("unexpected cancelled: %s", got)
	}
	if got := NoActiveTaskEN(); got != "_No active task._" {
		t.Fatalf("unexpected no active task: %s", got)
	}
	if got := NoRecentUserInputToReplayEN(); got != "_No recent user input to replay._" {
		t.Fatalf("unexpected no recent input: %s", got)
	}
	if got := RecoverReplayQueueFullEN(); got != "_Recover replay queue is full; please resend manually._" {
		t.Fatalf("unexpected recover replay queue full: %s", got)
	}
	if got := RetryQueueFullEN(); got != "_Retry queue is full; please resend manually._" {
		t.Fatalf("unexpected retry queue full: %s", got)
	}
	if got := NoTurnToUndoEN(); got != "_No turn to undo._" {
		t.Fatalf("unexpected no turn to undo: %s", got)
	}
	if got := RetryNotAvailableZH(); got != "没有可重试的上一条用户消息。" {
		t.Fatalf("unexpected retry not available zh: %s", got)
	}
	if got := TodoStoreUnavailableEN(); got != "todo store unavailable" {
		t.Fatalf("unexpected todo unavailable en: %s", got)
	}
	if got := MemoryStoreUnavailableEN(); got != "memory store unavailable" {
		t.Fatalf("unexpected memory unavailable en: %s", got)
	}
	if got := FailedWithEscapedErrorEN("Resolve", "bad request"); got != "_Resolve failed: bad request_" {
		t.Fatalf("unexpected failed-with-error text: %s", got)
	}
	if got := FailedEN("Pair"); got != "_Pair failed._" {
		t.Fatalf("unexpected failed text: %s", got)
	}
	if got := FailedFromSlashWithEscapedErrorEN("/next", "oops"); got != "_next failed: oops_" {
		t.Fatalf("unexpected failed-from-slash text: %s", got)
	}
	if got := MarshalFailedEN("Schema", nil); got != "Schema marshal failed" {
		t.Fatalf("unexpected marshal failed without error: %s", got)
	}
	if got := MarshalFailedEN("Schema", fmt.Errorf("boom")); got != "Schema marshal failed: boom" {
		t.Fatalf("unexpected marshal failed with error: %s", got)
	}
}
