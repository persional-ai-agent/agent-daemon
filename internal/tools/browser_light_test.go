package tools

import (
	"context"
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
