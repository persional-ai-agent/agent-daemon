package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestWeComAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-WeCom-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter, err := NewWeComAdapter(srv.URL, "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	res, err := adapter.Send(context.Background(), "", "hello", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if !res.Success || gotSecret != "sec" {
		t.Fatalf("unexpected result=%+v secret=%q", res, gotSecret)
	}
}

func TestWeComAdapterHandleWebhook(t *testing.T) {
	adapter, _ := NewWeComAdapter("http://example", "sec")
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })
	body := `{"from_user":"u1","chat_id":"c1","text":"/status","msg_id":"m1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/wecom/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-WeCom-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "c1" || got.UserID != "u1" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
