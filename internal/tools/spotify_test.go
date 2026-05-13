package tools

import (
	"reflect"
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

func TestSpotifyActionSchemasExposeEnum(t *testing.T) {
	check := func(name string, params map[string]any, want []string) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		action, _ := props["action"].(map[string]any)
		enum, _ := action["enum"].([]string)
		if !reflect.DeepEqual(enum, want) {
			t.Fatalf("%s action enum=%v, want=%v", name, enum, want)
		}
	}
	check("spotify_playback", spotifyPlaybackParams(), []string{"get", "pause", "play"})
	check("spotify_queue", spotifyQueueParams(), []string{"get", "add"})
}

func TestSpotifySchemasDocumentDefaultsAndBounds(t *testing.T) {
	props := func(params map[string]any) map[string]any {
		p, _ := params["properties"].(map[string]any)
		return p
	}
	search := props(spotifySearchParams())
	searchType, _ := search["type"].(map[string]any)
	searchLimit, _ := search["limit"].(map[string]any)
	if desc, _ := searchType["description"].(string); !strings.Contains(desc, "default: track") {
		t.Fatalf("spotify_search.type description=%q", desc)
	}
	if enum, _ := searchType["enum"].([]string); !reflect.DeepEqual(enum, []string{"album", "artist", "playlist", "track", "show", "episode", "audiobook"}) {
		t.Fatalf("spotify_search.type enum=%v", enum)
	}
	if desc, _ := searchLimit["description"].(string); !strings.Contains(desc, "default 10") || !strings.Contains(desc, "max 50") {
		t.Fatalf("spotify_search.limit description=%q", desc)
	}
	for _, tc := range []struct {
		name   string
		params map[string]any
	}{
		{"spotify_playlists", spotifyPlaylistsParams()},
		{"spotify_albums", spotifyAlbumsParams()},
		{"spotify_library", spotifyLibraryParams()},
	} {
		p := props(tc.params)
		limit, _ := p["limit"].(map[string]any)
		offset, _ := p["offset"].(map[string]any)
		if desc, _ := limit["description"].(string); !strings.Contains(desc, "default 20") || !strings.Contains(desc, "max 50") {
			t.Fatalf("%s.limit description=%q", tc.name, desc)
		}
		if desc, _ := offset["description"].(string); !strings.Contains(desc, "default 0") {
			t.Fatalf("%s.offset description=%q", tc.name, desc)
		}
	}
}
