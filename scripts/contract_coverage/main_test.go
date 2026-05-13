package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageDetectsUncoveredCoreOps(t *testing.T) {
	dir := t.TempDir()
	openapi := `openapi: 3.0.3
paths:
  /v1/ui/tools:
    get: {}
  /v1/chat:
    post: {}
`
	replay := `[
  {"method":"GET","path":"/v1/ui/tools","contract_path":"/v1/ui/tools"}
]`
	openapiPath := filepath.Join(dir, "openapi.yaml")
	replayPath := filepath.Join(dir, "replay.json")
	if err := os.WriteFile(openapiPath, []byte(openapi), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replayPath, []byte(replay), 0o644); err != nil {
		t.Fatal(err)
	}
	coreOps, err := loadOpenAPIOps(openapiPath)
	if err != nil {
		t.Fatal(err)
	}
	replayOps, _, err := loadReplayOps(replayPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := coreOps["POST /v1/chat"]; !ok {
		t.Fatalf("missing expected core op")
	}
	if _, ok := replayOps["GET /v1/ui/tools"]; !ok {
		t.Fatalf("missing expected replay op")
	}
	if _, ok := replayOps["POST /v1/chat"]; ok {
		t.Fatalf("unexpected covered op")
	}
}
