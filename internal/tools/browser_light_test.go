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
