package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestWhatsAppAdapterName(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Name(); got != "whatsapp" {
		t.Fatalf("Name()=%q", got)
	}
}

func TestWhatsAppAdapterSend(t *testing.T) {
	var (
		gotAuth string
		gotPath string
		gotBody map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.abc"}]}`))
	}))
	defer srv.Close()

	a, err := NewWhatsAppAdapter("test-token", "555", "verify")
	if err != nil {
		t.Fatal(err)
	}
	a.baseURL = srv.URL
	a.httpClient = srv.Client()
	res, err := a.Send(context.Background(), "8613800138000", "hello", "wamid.reply")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.MessageID != "wamid.abc" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotPath != "/v21.0/555/messages" {
		t.Fatalf("path=%q", gotPath)
	}
	if strings.TrimSpace(anyToString(gotBody["to"])) != "8613800138000" {
		t.Fatalf("to=%v", gotBody["to"])
	}
}

func TestWhatsAppWebhookVerify(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify-token")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=verify-token&hub.challenge=abc123", nil)
	rr := httptest.NewRecorder()
	a.HandleWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "abc123" {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestWhatsAppWebhookMessageDispatch(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify-token")
	if err != nil {
		t.Fatal(err)
	}
	var got gateway.MessageEvent
	a.OnMessage(context.Background(), func(_ context.Context, event gateway.MessageEvent) {
		got = event
	})
	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"8613800138000","id":"wamid.msg1","type":"text","text":{"body":"hello"},"context":{"id":"wamid.prev"}}]}}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.HandleWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.ChatID != "8613800138000" || got.Text != "hello" || got.ReplyToID != "wamid.prev" {
		t.Fatalf("event=%+v", got)
	}
}

func anyToString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
