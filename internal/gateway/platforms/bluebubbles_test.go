package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestBlueBubblesAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-BlueBubbles-Secret")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"m-1"}`))
	}))
	defer srv.Close()

	adapter, err := NewBlueBubblesAdapter(srv.URL, "s3")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	res, err := adapter.Send(context.Background(), "chat-1", "hello", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if !res.Success || res.MessageID != "m-1" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if gotSecret != "s3" {
		t.Fatalf("secret not forwarded, got=%q", gotSecret)
	}
}

func TestBlueBubblesAdapterHandleWebhook(t *testing.T) {
	adapter, err := NewBlueBubblesAdapter("http://example.test/out", "secret")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })

	body := `{"text":"/status","chat_id":"chat-1","user_id":"u1","user_name":"alice"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/bluebubbles/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-BlueBubbles-Secret", "secret")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "chat-1" || got.UserID != "u1" || got.Text != "/status" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestBlueBubblesAdapterHandleWebhookForbidden(t *testing.T) {
	adapter, err := NewBlueBubblesAdapter("http://example.test/out", "secret")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/bluebubbles/webhook", strings.NewReader(`{"text":"hi","chat_id":"c1"}`))
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
