package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("httptest server not available in this environment: %v", r)
		}
	}()
	return httptest.NewServer(h)
}

