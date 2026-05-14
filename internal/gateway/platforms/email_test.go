package platforms

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestEmailAdapterSend(t *testing.T) {
	adapter, err := NewEmailAdapter("smtp.example.com", "587", "user", "pass", "bot@example.com", "sec")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	called := false
	adapter.sendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		if addr != "smtp.example.com:587" || from != "bot@example.com" || len(to) != 1 || to[0] != "u@example.com" {
			t.Fatalf("unexpected smtp args addr=%s from=%s to=%v", addr, from, to)
		}
		if !strings.Contains(string(msg), "Subject: hello") {
			t.Fatalf("missing subject in message: %s", string(msg))
		}
		return nil
	}
	res, err := adapter.Send(context.Background(), "u@example.com", "hello\nbody", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if !called || !res.Success {
		t.Fatalf("unexpected result: %+v called=%v", res, called)
	}
}

func TestEmailAdapterSendFailure(t *testing.T) {
	adapter, _ := NewEmailAdapter("smtp.example.com", "587", "user", "pass", "bot@example.com", "")
	adapter.sendMail = func(string, smtp.Auth, string, []string, []byte) error {
		return errors.New("smtp down")
	}
	res, err := adapter.Send(context.Background(), "u@example.com", "hello", "")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if res.Success || !strings.Contains(res.Error, "smtp down") {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestEmailAdapterHandleWebhook(t *testing.T) {
	adapter, _ := NewEmailAdapter("smtp.example.com", "587", "user", "pass", "bot@example.com", "sec")
	var got gateway.MessageEvent
	adapter.OnMessage(context.Background(), func(_ context.Context, e gateway.MessageEvent) { got = e })

	body := `{"from":"u@example.com","subject":"s","text":"/status","id":"m1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/email/webhook", strings.NewReader(body))
	req.Header.Set("X-Agent-Email-Secret", "sec")
	rec := httptest.NewRecorder()
	adapter.HandleWebhook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.ChatID != "u@example.com" || got.MessageID != "m1" || !got.IsCommand {
		t.Fatalf("unexpected event: %+v", got)
	}
}
