package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestMattermostAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-Mattermost-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter, err := NewMattermostAdapter(srv.URL, "sec")
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

func TestMattermostAdapterHandleWebhook(t *testing.T) {
	adapter, _ := NewMattermostAdapter("http://example", "sec")
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })
	body := `{"user_id":"u1","channel_id":"c1","text":"/status","post_id":"p1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/mattermost/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-Mattermost-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "c1" || got.UserID != "u1" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
