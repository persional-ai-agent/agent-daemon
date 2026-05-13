#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${1:-dev}"
COMMIT="$(git rev-parse --short HEAD)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
OUT_DIR="${OUT_DIR:-dist}"
mkdir -p "$OUT_DIR"

OUT_FILE="$OUT_DIR/ui-tui-${VERSION}"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.BuildVersion=${VERSION} -X main.BuildCommit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" -o "$OUT_FILE" ./ui-tui
echo "built: $OUT_FILE"
