package tools

import (
	"strings"
	"testing"
)

func TestIntegrationSchemasDocumentDefaults(t *testing.T) {
	check := func(name string, params map[string]any, key string, want string) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		field, _ := props[key].(map[string]any)
		desc, _ := field["description"].(string)
		if !strings.Contains(desc, want) {
			t.Fatalf("%s %s description=%q, want contains %q", name, key, desc, want)
		}
	}
	check("rl", rlParams(), "action", "default: unknown")
	check("spotify", spotifyParams(), "tool", "default: spotify")
	check("yb", ybParams(), "tool", "default: yb")
}
