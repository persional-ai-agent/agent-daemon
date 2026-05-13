package tools

import (
	"strings"
	"testing"
)

func TestSpotifyLimitOffsetBounds(t *testing.T) {
	limit, offset := spotifyLimitOffset(map[string]any{"limit": -1, "offset": -2}, 20)
	if limit != 20 || offset != 0 {
		t.Fatalf("limit=%d offset=%d, want 20 0", limit, offset)
	}
	limit, offset = spotifyLimitOffset(map[string]any{"limit": 999, "offset": 7}, 20)
	if limit != 50 || offset != 7 {
		t.Fatalf("limit=%d offset=%d, want 50 7", limit, offset)
	}
}

func TestSpotifySchemasExposePagination(t *testing.T) {
	registry := NewRegistry()
	RegisterBuiltins(registry, NewProcessRegistry(t.TempDir()))
	schemas := registry.Schemas()

	check := func(tool string) {
		t.Helper()
		for _, schema := range schemas {
			if schema.Function.Name != tool {
				continue
			}
			props, _ := schema.Function.Parameters["properties"].(map[string]any)
			if _, ok := props["limit"]; !ok {
				t.Fatalf("%s missing limit property", tool)
			}
			if _, ok := props["offset"]; !ok {
				t.Fatalf("%s missing offset property", tool)
			}
			return
		}
		t.Fatalf("schema not found for %s", tool)
	}

	check("spotify_playlists")
	check("spotify_albums")
	check("spotify_library")
}

func TestSpotifyActionSchemasDocumentDefaultGet(t *testing.T) {
	check := func(name string, params map[string]any) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		action, _ := props["action"].(map[string]any)
		desc, _ := action["description"].(string)
		if !strings.Contains(desc, "default: get") {
			t.Fatalf("%s action description=%q, want default hint", name, desc)
		}
	}
	check("spotify_playback", spotifyPlaybackParams())
	check("spotify_queue", spotifyQueueParams())
}
