package tools

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestWebExtractBasicHTMLToText(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body><h1>Hello</h1><p>World</p></body></html>"))
	}))
	defer srv.Close()

	b := &BuiltinTools{}
	res, err := b.webExtract(context.Background(), map[string]any{"url": srv.URL, "max_chars": 1000}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	content, _ := res["content"].(string)
	if content == "" || content == "<html><body>" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestWebSearchParsesDDGLinks(t *testing.T) {
	ddgHTML := `
<a class="result__a" href="https://duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fhello">Example</a>
<a class="result__a" href="https://example.com/2">Second</a>
`
	results := parseDuckDuckGoHTMLResults(ddgHTML, 10)
	if len(results) != 2 {
		t.Fatalf("results=%d", len(results))
	}
	if results[0]["url"] != "https://example.com/hello" {
		t.Fatalf("url0=%v", results[0]["url"])
	}
}

func TestWebExtractTruncatedFlagIsAccurate(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body>abcdef</body></html>"))
	}))
	defer srv.Close()

	b := &BuiltinTools{}
	res, err := b.webExtract(context.Background(), map[string]any{"url": srv.URL, "max_chars": 6}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if truncated, _ := res["truncated"].(bool); truncated {
		t.Fatalf("truncated=%v, want false when content length equals max_chars", truncated)
	}

	res2, err := b.webExtract(context.Background(), map[string]any{"url": srv.URL, "max_chars": 5}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if truncated, _ := res2["truncated"].(bool); !truncated {
		t.Fatalf("truncated=%v, want true when content exceeds max_chars", truncated)
	}
}

func TestWebExtractSchemaDocumentsMaxCharsBounds(t *testing.T) {
	props, _ := webExtractParams()["properties"].(map[string]any)
	maxChars, _ := props["max_chars"].(map[string]any)
	desc, _ := maxChars["description"].(string)
	if !strings.Contains(desc, "default 8000") || !strings.Contains(desc, "max 200000") {
		t.Fatalf("web_extract max_chars description=%q, want default/max hint", desc)
	}
}
