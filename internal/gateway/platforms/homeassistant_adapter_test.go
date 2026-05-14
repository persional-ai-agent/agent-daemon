package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestHomeAssistantAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-HA-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter, err := NewHomeAssistantAdapter(srv.URL, "token", "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	res, err := adapter.Send(context.Background(), "notif", "hello", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if !res.Success {
		t.Fatalf("unexpected result: %+v", res)
	}
	if gotSecret != "sec" {
		t.Fatalf("expected secret header, got=%q", gotSecret)
	}
}

func TestHomeAssistantAdapterHandleWebhook(t *testing.T) {
	adapter, _ := NewHomeAssistantAdapter("http://ha.test", "token", "sec")
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })

	body := `{"text":"/status","user_id":"u1","chat_id":"room-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/homeassistant/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-HA-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "room-1" || got.UserID != "u1" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
