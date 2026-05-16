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
