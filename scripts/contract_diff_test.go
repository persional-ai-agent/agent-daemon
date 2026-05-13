package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempSpec(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return p
}

func containsLine(lines []string, part string) bool {
	for _, l := range lines {
		if strings.Contains(l, part) {
			return true
		}
	}
	return false
}

func TestContractDiffDetectsTypeEnumAndParamBreaking(t *testing.T) {
	dir := t.TempDir()
	base := `openapi: 3.0.3
paths:
  /v1/a:
    get:
      parameters:
        - in: query
          name: mode
          required: false
          schema:
            type: string
            enum: [x, y]
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                required: [status]
                properties:
                  status:
                    type: string
                    enum: [ok, warn]
`
	target := `openapi: 3.0.3
paths:
  /v1/a:
    get:
      parameters:
        - in: query
          name: mode
          required: true
          schema:
            type: integer
            enum: [x]
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                required: [status]
                properties:
                  status:
                    type: integer
                    enum: [ok]
`
	basePath := writeTempSpec(t, dir, "base.yaml", base)
	targetPath := writeTempSpec(t, dir, "target.yaml", target)
	bs, err := readSpec(basePath)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := readSpec(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	report := buildReport(basePath, targetPath, bs, ts)
	if len(report.Breaking) == 0 {
		t.Fatalf("expected breaking changes, got %+v", report)
	}
	if !containsLine(report.Breaking, "changed type for query:mode") {
		t.Fatalf("expected parameter type change breaking, got %+v", report.Breaking)
	}
	if !containsLine(report.Breaking, "parameter became required: query:mode") {
		t.Fatalf("expected required param breaking, got %+v", report.Breaking)
	}
	if !containsLine(report.Breaking, "removed enum value for query:mode: y") {
		t.Fatalf("expected enum shrink breaking, got %+v", report.Breaking)
	}
	if !containsLine(report.Breaking, "response200 changed type for status: string -> integer") {
		t.Fatalf("expected response type breaking, got %+v", report.Breaking)
	}
}

func TestContractDiffResolvesAllOfRef(t *testing.T) {
	dir := t.TempDir()
	base := `openapi: 3.0.3
components:
  schemas:
    Envelope:
      type: object
      required: [ok]
      properties:
        ok: { type: boolean }
paths:
  /v1/a:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/Envelope'
                  - type: object
                    required: [status]
                    properties:
                      status: { type: string }
`
	target := strings.ReplaceAll(base, "type: string", "type: integer")
	basePath := writeTempSpec(t, dir, "base.yaml", base)
	targetPath := writeTempSpec(t, dir, "target.yaml", target)
	bs, _ := readSpec(basePath)
	ts, _ := readSpec(targetPath)
	report := buildReport(basePath, targetPath, bs, ts)
	if !containsLine(report.Breaking, "response200 changed type for status: string -> integer") {
		t.Fatalf("expected allOf/$ref type diff, got %+v", report.Breaking)
	}
}
