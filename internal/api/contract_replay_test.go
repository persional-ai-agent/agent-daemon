package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

type replayCase struct {
	Name        string         `json:"name"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	ContentType string         `json:"content_type,omitempty"`
	Body        map[string]any `json:"body,omitempty"`
	Snapshot    string         `json:"snapshot"`
}

type replayResult struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Pass   bool   `json:"pass"`
	Error  string `json:"error,omitempty"`
}

type openAPISpec struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

func replayAsMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func replayAsSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func loadReplayCases(t *testing.T) []replayCase {
	t.Helper()
	bs, err := os.ReadFile(filepath.Join("testdata", "replay", "cases.json"))
	if err != nil {
		t.Fatalf("read replay cases: %v", err)
	}
	var out []replayCase
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("decode replay cases: %v", err)
	}
	return out
}

func loadOpenAPISpec(t *testing.T) openAPISpec {
	t.Helper()
	bs, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "ui-chat-contract.openapi.yaml"))
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}
	var spec openAPISpec
	if err := yaml.Unmarshal(bs, &spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}
	return spec
}

func replayServer() *Server {
	reg := tools.NewRegistry()
	reg.Register(apiTestTool{
		name: "x",
		call: func(_ context.Context, _ map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	return &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
			Registry:     reg,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}
}

func findOpenAPIRequiredTop(spec openAPISpec, path, method, statusCode string) []string {
	pathItem := spec.Paths[path]
	if pathItem == nil {
		return nil
	}
	op := replayAsMap(pathItem[strings.ToLower(method)])
	if op == nil {
		return nil
	}
	resp := replayAsMap(replayAsMap(op["responses"])[statusCode])
	if resp == nil {
		return nil
	}
	schema := replayAsMap(replayAsMap(replayAsMap(resp["content"])["application/json"])["schema"])
	req := replayAsSlice(schema["required"])
	out := make([]string, 0, len(req))
	for _, it := range req {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func assertReplayOpenAPI(t *testing.T, spec openAPISpec, rc replayCase, rec *httptest.ResponseRecorder, body map[string]any) {
	t.Helper()
	pathItem := spec.Paths[rc.Path]
	if pathItem == nil {
		t.Fatalf("openapi missing path: %s", rc.Path)
	}
	op := pathItem[strings.ToLower(rc.Method)]
	if op == nil {
		t.Fatalf("openapi missing method: %s %s", rc.Method, rc.Path)
	}
	respCode := "200"
	if rec.Code >= 400 {
		respCode = "4XX"
	}
	responses := replayAsMap(replayAsMap(op)["responses"])
	if _, ok := responses[respCode]; !ok {
		t.Fatalf("openapi missing response code bucket %s for %s %s", respCode, rc.Method, rc.Path)
	}
	for _, key := range findOpenAPIRequiredTop(spec, rc.Path, rc.Method, respCode) {
		if _, ok := body[key]; !ok {
			t.Fatalf("openapi required field missing: %s (%s %s)", key, rc.Method, rc.Path)
		}
	}
}

func writeReplayReport(t *testing.T, results []replayResult) {
	t.Helper()
	reportPath := strings.TrimSpace(os.Getenv("CONTRACT_REPLAY_REPORT"))
	if reportPath == "" {
		reportPath = filepath.Join("..", "..", "artifacts", "contract-replay.json")
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatalf("mkdir replay report: %v", err)
	}
	bs, err := json.MarshalIndent(map[string]any{"results": results}, "", "  ")
	if err != nil {
		t.Fatalf("marshal replay report: %v", err)
	}
	if err := os.WriteFile(reportPath, bs, 0o644); err != nil {
		t.Fatalf("write replay report: %v", err)
	}
}

func replaySnapshotPayload(name string, status int, headers map[string]any, body map[string]any) map[string]any {
	switch name {
	case "chat_success":
		result, _ := body["result"].(map[string]any)
		summary, _ := result["summary"].(map[string]any)
		return map[string]any{
			"status":  status,
			"headers": headers,
			"body": map[string]any{
				"ok":          body["ok"],
				"api_version": body["api_version"],
				"compat":      body["compat"],
				"result": map[string]any{
					"session_id":         result["session_id"],
					"final_response":     result["final_response"],
					"turns_used":         result["turns_used"],
					"finished_naturally": result["finished_naturally"],
					"summary":            summary,
				},
				"session_id":         body["session_id"],
				"final_response":     body["final_response"],
				"turns_used":         body["turns_used"],
				"finished_naturally": body["finished_naturally"],
			},
		}
	case "chat_cancel_success":
		return map[string]any{
			"status":  status,
			"headers": headers,
			"body": map[string]any{
				"ok":          body["ok"],
				"api_version": body["api_version"],
				"compat":      body["compat"],
				"result":      body["result"],
				"session_id":  body["session_id"],
				"cancelled":   body["cancelled"],
			},
		}
	default:
		return map[string]any{
			"status":  status,
			"headers": headers,
			"body":    body,
		}
	}
}

func TestContractReplay(t *testing.T) {
	spec := loadOpenAPISpec(t)
	cases := loadReplayCases(t)
	srv := replayServer()
	results := make([]replayResult, 0, len(cases))
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			var reqBody *bytes.Reader
			if tc.Body != nil {
				bs, _ := json.Marshal(tc.Body)
				reqBody = bytes.NewReader(bs)
			} else {
				reqBody = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(tc.Method, tc.Path, reqBody)
			if strings.TrimSpace(tc.ContentType) != "" {
				req.Header.Set("Content-Type", tc.ContentType)
			}
			rec := httptest.NewRecorder()
			if tc.Name == "chat_cancel_success" {
				srv.mu.Lock()
				srv.active = map[string]activeRun{
					"s-cancel": {token: "t1", cancel: func() {}},
				}
				srv.mu.Unlock()
			}
			srv.Handler().ServeHTTP(rec, req)
			body := decodeJSONMap(t, rec)
			assertReplayOpenAPI(t, spec, tc, rec, body)
			headers := map[string]any{
				"X-Agent-UI-API-Version": rec.Header().Get("X-Agent-UI-API-Version"),
				"X-Agent-UI-API-Compat":  rec.Header().Get("X-Agent-UI-API-Compat"),
			}
			assertContractSnapshot(t, tc.Snapshot, replaySnapshotPayload(tc.Name, rec.Code, headers, body))
			results = append(results, replayResult{Name: tc.Name, Status: rec.Code, Pass: true})
		})
	}
	writeReplayReport(t, results)
}
