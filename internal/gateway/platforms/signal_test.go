package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestSignalAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-Signal-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter, err := NewSignalAdapter(srv.URL, "+123", "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	res, err := adapter.Send(context.Background(), "+456", "hello", "")
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

func TestSignalAdapterHandleWebhook(t *testing.T) {
	adapter, err := NewSignalAdapter("http://example.test", "+123", "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })

	body := `{"envelope":{"source":"+456"},"dataMessage":{"message":"/status"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/signal/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-Signal-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "+456" || got.Text != "/status" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
