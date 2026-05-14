package platforms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestMatrixAdapterSend(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Agent-Matrix-Secret")
		_, _ = w.Write([]byte(`{"event_id":"$evt1"}`))
	}))
	defer srv.Close()

	adapter, err := NewMatrixAdapter(srv.URL, "token", "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	res, err := adapter.Send(context.Background(), "!room:example.org", "hello", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if !res.Success || res.MessageID != "$evt1" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if gotSecret != "sec" {
		t.Fatalf("secret not forwarded")
	}
}

func TestMatrixAdapterHandleWebhook(t *testing.T) {
	adapter, _ := NewMatrixAdapter("http://matrix", "token", "sec")
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })

	body := `{"sender":"@u:example.org","room_id":"!r:example.org","event_id":"$e","content":{"body":"/status"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/matrix/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-Matrix-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID == "" || got.UserID == "" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
