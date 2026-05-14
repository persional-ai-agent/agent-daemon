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
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

type replayCase struct {
	Name        string         `json:"name"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	ContractPath string        `json:"contract_path,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Body        map[string]any `json:"body,omitempty"`
	Snapshot    string         `json:"snapshot"`
	ExpectSSE   bool           `json:"expect_sse,omitempty"`
	ExpectEvents []string      `json:"expect_events,omitempty"`
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

type wsReplayCase struct {
	Name         string         `json:"name"`
	Path         string         `json:"path"`
	Request      map[string]any `json:"request"`
	RawRequest   string         `json:"raw_request,omitempty"`
	ExpectEvents []string       `json:"expect_events"`
}

type wsContract struct {
	Version string              `json:"version"`
	Events  map[string][]string `json:"events"`
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

func loadWSReplayCases(t *testing.T) []wsReplayCase {
	t.Helper()
	bs, err := os.ReadFile(filepath.Join("testdata", "replay", "ws_cases.json"))
	if err != nil {
		t.Fatalf("read ws replay cases: %v", err)
	}
	var out []wsReplayCase
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("decode ws replay cases: %v", err)
	}
	return out
}

func loadWSContract(t *testing.T) wsContract {
	t.Helper()
	bs, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "ws-chat-events.schema.json"))
	if err != nil {
		t.Fatalf("read ws contract: %v", err)
	}
	var out wsContract
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("decode ws contract: %v", err)
	}
	return out
}

func replayServer() *Server {
	reg := tools.NewRegistry()
	reg.Register(apiTestTool{
		name: "x",
		call: func(_ context.Context, _ map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	reg.Register(apiTestTool{
		name: "approval",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{
				"success":     true,
				"action":      args["action"],
				"approval_id": args["approval_id"],
				"approved":    args["approve"],
			}, nil
		},
	})
	return &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
			Registry:     reg,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
		ConfigSnapshotFn: func() map[string]any { return map[string]any{"model_provider": "openai"} },
		ConfigUpdateFn: func(key, value string) (map[string]any, error) {
			return map[string]any{"success": true, "key": key, "value": value}, nil
		},
		GatewayStatusFn: func() map[string]any { return map[string]any{"enabled": true, "running": false} },
		GatewayActionFn: func(action string) (map[string]any, error) {
			return map[string]any{"success": true, "action": action}, nil
		},
		SkillListFn: func() ([]map[string]any, error) {
			return []map[string]any{
				{"name": "skill-a", "path": "skills/skill-a/SKILL.md"},
				{"name": "skill-b", "path": "skills/skill-b/SKILL.md"},
			}, nil
		},
		SkillsReloadFn: func() (map[string]any, error) {
			return map[string]any{"success": true, "count": 2}, nil
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
	contractPath := strings.TrimSpace(rc.ContractPath)
	if contractPath == "" {
		contractPath = rc.Path
	}
	pathItem := spec.Paths[contractPath]
	if pathItem == nil {
		t.Fatalf("openapi missing path: %s", contractPath)
	}
	op := pathItem[strings.ToLower(rc.Method)]
	if op == nil {
		t.Fatalf("openapi missing method: %s %s", rc.Method, contractPath)
	}
	respCode := "200"
	if rec.Code >= 400 {
		respCode = "4XX"
	}
	responses := replayAsMap(replayAsMap(op)["responses"])
	if _, ok := responses[respCode]; !ok {
		t.Fatalf("openapi missing response code bucket %s for %s %s", respCode, rc.Method, contractPath)
	}
	for _, key := range findOpenAPIRequiredTop(spec, contractPath, rc.Method, respCode) {
		if _, ok := body[key]; !ok {
			t.Fatalf("openapi required field missing: %s (%s %s)", key, rc.Method, contractPath)
		}
	}
}

func assertReplaySSE(t *testing.T, rc replayCase, rec *httptest.ResponseRecorder) {
	t.Helper()
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", rec.Header().Get("Content-Type"))
	}
	out := rec.Body.String()
	for _, ev := range rc.ExpectEvents {
		if !strings.Contains(out, ev) {
			t.Fatalf("missing sse event marker %q in body=%s", ev, out)
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
			if tc.ExpectSSE {
				assertReplaySSE(t, tc, rec)
				assertReplayOpenAPI(t, spec, tc, rec, map[string]any{})
				results = append(results, replayResult{Name: tc.Name, Status: rec.Code, Pass: true})
				return
			}
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

func TestContractWSReplay(t *testing.T) {
	cases := loadWSReplayCases(t)
	contract := loadWSContract(t)
	srv := replayServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	wsBase := "ws" + strings.TrimPrefix(ts.URL, "http")

	type wsReplayResult struct {
		Name       string   `json:"name"`
		Pass       bool     `json:"pass"`
		SeenEvents []string `json:"seen_events"`
	}
	results := make([]wsReplayResult, 0, len(cases))
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			conn, _, err := websocket.DefaultDialer.Dial(wsBase+tc.Path, nil)
			if err != nil {
				t.Fatalf("ws dial failed: %v", err)
			}
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			if strings.TrimSpace(tc.RawRequest) != "" {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(tc.RawRequest)); err != nil {
					t.Fatalf("ws write raw request failed: %v", err)
				}
			} else {
				if err := conn.WriteJSON(tc.Request); err != nil {
					t.Fatalf("ws write request failed: %v", err)
				}
			}
			seen := make([]string, 0, 8)
			seenSet := map[string]struct{}{}
			for {
				msg := map[string]any{}
				if err := conn.ReadJSON(&msg); err != nil {
					t.Fatalf("ws read failed: %v", err)
				}
				typ, _ := msg["type"].(string)
				if strings.TrimSpace(typ) == "" {
					t.Fatalf("ws event missing type: %+v", msg)
				}
				if _, ok := seenSet[typ]; !ok {
					seenSet[typ] = struct{}{}
					seen = append(seen, typ)
				}
				reqFields := contract.Events[typ]
				for _, field := range reqFields {
					if _, ok := msg[field]; !ok {
						t.Fatalf("ws event %s missing field %s: %+v", typ, field, msg)
					}
				}
				if typ == "result" || typ == "error" || typ == "cancelled" {
					break
				}
			}
			for _, ev := range tc.ExpectEvents {
				if _, ok := seenSet[ev]; !ok {
					t.Fatalf("expected ws event %s not seen, seen=%v", ev, seen)
				}
			}
			results = append(results, wsReplayResult{Name: tc.Name, Pass: true, SeenEvents: seen})
		})
	}

	reportPath := strings.TrimSpace(os.Getenv("CONTRACT_WS_REPLAY_REPORT"))
	if reportPath == "" {
		reportPath = filepath.Join("..", "..", "artifacts", "contract-ws-replay.json")
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatalf("mkdir ws replay report: %v", err)
	}
	bs, err := json.MarshalIndent(map[string]any{"results": results, "contract_version": contract.Version}, "", "  ")
	if err != nil {
		t.Fatalf("marshal ws replay report: %v", err)
	}
	if err := os.WriteFile(reportPath, bs, 0o644); err != nil {
		t.Fatalf("write ws replay report: %v", err)
	}
}
