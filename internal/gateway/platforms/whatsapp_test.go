package platforms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestWhatsAppAdapterName(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify", "")
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

	a, err := NewWhatsAppAdapter("test-token", "555", "verify", "")
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
	a, err := NewWhatsAppAdapter("token", "123", "verify-token", "")
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
	a, err := NewWhatsAppAdapter("token", "123", "verify-token", "")
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

func TestWhatsAppWebhookMediaDispatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v21.0/mid-1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"https://cdn.example.com/media-1.jpg"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	a, err := NewWhatsAppAdapter("token", "123", "verify-token", "")
	if err != nil {
		t.Fatal(err)
	}
	a.baseURL = srv.URL
	a.httpClient = srv.Client()
	var got gateway.MessageEvent
	a.OnMessage(context.Background(), func(_ context.Context, event gateway.MessageEvent) {
		got = event
	})

	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"8613800138000","id":"wamid.msg2","type":"image","image":{"id":"mid-1","caption":"请看图片"}}]}}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.HandleWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.Text != "请看图片" {
		t.Fatalf("text=%q", got.Text)
	}
	if len(got.MediaURLs) != 1 || got.MediaURLs[0] != "https://cdn.example.com/media-1.jpg" {
		t.Fatalf("media=%v", got.MediaURLs)
	}
}

func TestWhatsAppWebhookSignatureRequired(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify-token", "secret-1")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"8613800138000","id":"wamid.msg3","type":"text","text":{"body":"hello"}}]}}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.HandleWebhook(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestWhatsAppWebhookSignatureValid(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123", "verify-token", "secret-1")
	if err != nil {
		t.Fatal(err)
	}
	var got gateway.MessageEvent
	a.OnMessage(context.Background(), func(_ context.Context, event gateway.MessageEvent) {
		got = event
	})
	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"8613800138000","id":"wamid.msg4","type":"text","text":{"body":"hello signed"}}]}}]}]}`
	mac := hmac.New(sha256.New, []byte("secret-1"))
	_, _ = mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	rr := httptest.NewRecorder()
	a.HandleWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.Text != "hello signed" {
		t.Fatalf("event=%+v", got)
	}
}

func anyToString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
