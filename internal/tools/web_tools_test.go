package tools

import (
	"context"
	"net/http"
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

