package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhatsAppAdapterName(t *testing.T) {
	a, err := NewWhatsAppAdapter("token", "123")
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

	a, err := NewWhatsAppAdapter("test-token", "555")
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

func anyToString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
