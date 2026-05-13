#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[ui-tui-smoke] start"

TMP_HOME="$(mktemp -d)"
trap 'chmod -R u+w "$TMP_HOME" >/dev/null 2>&1 || true; rm -rf "$TMP_HOME" >/dev/null 2>&1 || true' EXIT

BASE_HTTP="${AGENT_HTTP_BASE:-http://127.0.0.1:8080}"
BASE_WS="${AGENT_API_BASE:-ws://127.0.0.1:8080/v1/chat/ws}"

OUT_LOCAL="$(mktemp)"
HOME="$TMP_HOME" AGENT_HTTP_BASE="$BASE_HTTP" AGENT_API_BASE="$BASE_WS" \
	timeout 20s go run ./ui-tui <<'EOF' >"$OUT_LOCAL"
/help
/status
/reload-config
/history 5
/events 5
/bookmark add smoke
/bookmark list
/bookmark use smoke
/quit
EOF

grep -q "commands:" "$OUT_LOCAL"
grep -q "status=" "$OUT_LOCAL"
grep -q "bookmark saved: smoke" "$OUT_LOCAL"
grep -q "bookmark loaded: smoke" "$OUT_LOCAL"
echo "[ui-tui-smoke] local command path ok"

go test ./ui-tui -run 'TestSendTurnReconnect|TestFindLatestPendingApproval|TestFindPendingApprovals|TestParseEventSaveArgsAndFilter|TestLoadRuntimeStateCorruptBackup' -count=1 >/dev/null
echo "[ui-tui-smoke] reconnect/cancel/approval/parser/state recovery regression ok"

if curl -fsS "$BASE_HTTP/health" >/dev/null 2>&1; then
	OUT_HTTP="$(mktemp)"
	HOME="$TMP_HOME" AGENT_HTTP_BASE="$BASE_HTTP" AGENT_API_BASE="$BASE_WS" \
		timeout 20s go run ./ui-tui <<'EOF' >"$OUT_HTTP"
/health
/status
/quit
EOF
	grep -q "status=" "$OUT_HTTP"
	echo "[ui-tui-smoke] backend health path ok"
else
	echo "[ui-tui-smoke] backend not reachable, skip /health integration"
fi

echo "[ui-tui-smoke] done"
