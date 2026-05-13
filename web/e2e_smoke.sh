#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[web-smoke] start"

ARTIFACTS_DIR="${ARTIFACTS_DIR:-$ROOT_DIR/artifacts}"
mkdir -p "$ARTIFACTS_DIR"

npm --prefix web run test >/dev/null
npm --prefix web run build >/dev/null
node web/scripts/gen_diag_sample.mjs "$ARTIFACTS_DIR/diag-web.sample.json"
go run ./scripts/diag_bundle -file "$ARTIFACTS_DIR/diag-web.sample.json" >/dev/null

echo "[web-smoke] done"
