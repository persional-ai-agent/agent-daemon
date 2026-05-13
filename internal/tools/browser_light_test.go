package tools

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestBrowserSnapshotRespectsMaxChars(t *testing.T) {
	sessionID := "test-browser-snapshot-max-chars"
	st := getLightBrowser(sessionID)
	st.stack = []lightBrowserPage{
		{
			URL:      "https://example.com",
			HTML:     "<html><body>" + strings.Repeat("abc ", 200) + "</body></html>",
			LoadedAt: "2026-01-01T00:00:00Z",
			Status:   200,
		},
	}
	st.elements = map[string]lightElement{}

	b := &BuiltinTools{}
	res, err := b.browserSnapshot(context.Background(), map[string]any{"max_chars": 40}, ToolContext{SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	content, _ := res["content"].(string)
	if content == "" {
		t.Fatalf("expected non-empty snapshot content: %+v", res)
	}
	if len(content) > 300 {
		t.Fatalf("snapshot content too long, max_chars likely ignored: len=%d", len(content))
	}
}

func TestBrowserConsoleReturnsAppliedLimitInLightMode(t *testing.T) {
	sessionID := "test-browser-console-limit"
	st := getLightBrowser(sessionID)
	st.stack = []lightBrowserPage{
		{
			URL:      "https://example.com",
			HTML:     "<html><body>ok</body></html>",
			LoadedAt: "2026-01-01T00:00:00Z",
			Status:   200,
		},
	}

	b := &BuiltinTools{}
	res, err := b.browserConsole(context.Background(), map[string]any{"limit": 7}, ToolContext{SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := res["applied_limit"].(int); got != 7 {
		t.Fatalf("applied_limit=%v, want 7", res["applied_limit"])
	}
	if got, _ := res["count"].(int); got != 0 {
		t.Fatalf("count=%v, want 0", res["count"])
	}
}

func TestBrowserGetImagesRespectsLimit(t *testing.T) {
	sessionID := "test-browser-images-limit"
	st := getLightBrowser(sessionID)
	st.stack = []lightBrowserPage{
		{
			URL: "https://example.com/base/",
			HTML: `<html><body>
<img src="a.png"><img src="b.png"><img src="c.png"><img src="d.png">
</body></html>`,
			LoadedAt: "2026-01-01T00:00:00Z",
			Status:   200,
		},
	}

	b := &BuiltinTools{}
	res, err := b.browserGetImages(context.Background(), map[string]any{"limit": 2}, ToolContext{SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := res["count"].(int); got != 2 {
		t.Fatalf("count=%v, want 2", res["count"])
	}
	images, _ := res["images"].([]string)
	if len(images) != 2 {
		t.Fatalf("images len=%d, want 2", len(images))
	}
	if got, _ := res["applied_limit"].(int); got != 2 {
		t.Fatalf("applied_limit=%v, want 2", res["applied_limit"])
	}
}

func TestBrowserScrollReturnsNormalizedArgs(t *testing.T) {
	b := &BuiltinTools{}
	res, err := b.browserScroll(context.Background(), map[string]any{"direction": "SIDEWAYS", "amount": -3}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := res["direction"].(string); got != "down" {
		t.Fatalf("direction=%v, want down", res["direction"])
	}
	if got, _ := res["amount"].(int); got != 1 {
		t.Fatalf("amount=%v, want 1", res["amount"])
	}
	if got, _ := res["scroll_performed"].(bool); got {
		t.Fatalf("scroll_performed=%v, want false", res["scroll_performed"])
	}
}

func TestBrowserNavigateSupportsPostAndHeaders(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s want=POST", r.Method)
		}
		if got := r.Header.Get("X-Test"); got != "ok" {
			t.Fatalf("X-Test header=%q", got)
		}
		_, _ = w.Write([]byte("<html><body>done</body></html>"))
	}))
	defer srv.Close()
	b := &BuiltinTools{}
	out, err := b.browserNavigate(context.Background(), map[string]any{
		"url":          srv.URL,
		"method":       "POST",
		"body":         "a=1",
		"headers":      map[string]any{"X-Test": "ok"},
		"timeout_seconds": 5,
	}, ToolContext{SessionID: "browser-post"})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := out["success"].(bool); !ok {
		t.Fatalf("navigate failed: %#v", out)
	}
	if got, _ := out["method"].(string); got != "POST" {
		t.Fatalf("method=%q want POST", got)
	}
}

func TestBrowserNavigateRespectsMaxBytes(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body>" + strings.Repeat("x", 5000) + "</body></html>"))
	}))
	defer srv.Close()
	b := &BuiltinTools{}
	out, err := b.browserNavigate(context.Background(), map[string]any{
		"url":       srv.URL,
		"max_bytes": 256,
	}, ToolContext{SessionID: "browser-max-bytes"})
	if err != nil {
		t.Fatal(err)
	}
	if truncated, _ := out["truncated"].(bool); !truncated {
		t.Fatalf("expected truncated=true, got %#v", out)
	}
}
