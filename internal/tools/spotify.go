package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func spotifyToken() (string, bool) {
	tok := strings.TrimSpace(os.Getenv("SPOTIFY_ACCESS_TOKEN"))
	return tok, tok != ""
}

func spotifyDo(ctx context.Context, method, path, token string, body any) ([]byte, int, error) {
	base := "https://api.spotify.com/v1"
	var r io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(bs)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, r)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	return bs, resp.StatusCode, nil
}

func (b *BuiltinTools) spotifySearch(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	q := strings.TrimSpace(strArg(args, "q"))
	if q == "" {
		return nil, errors.New("q required")
	}
	typ := strings.TrimSpace(strArg(args, "type"))
	if typ == "" {
		typ = "track"
	}
	limit := intArg(args, "limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	u := url.Values{}
	u.Set("q", q)
	u.Set("type", typ)
	u.Set("limit", fmt.Sprintf("%d", limit))
	path := "/search?" + u.Encode()
	bs, code, err := spotifyDo(ctx, http.MethodGet, path, token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var data any
	_ = json.Unmarshal(bs, &data)
	return map[string]any{"success": true, "q": q, "type": typ, "limit": limit, "results": data}, nil
}

func (b *BuiltinTools) spotifyDevices(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/player/devices", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var data any
	_ = json.Unmarshal(bs, &data)
	return map[string]any{"success": true, "devices": data}, nil
}

func (b *BuiltinTools) spotifyPlayback(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		action = "get"
	}
	switch action {
	case "get":
		bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/player", token, nil)
		if err != nil {
			return nil, err
		}
		if code == 204 {
			return map[string]any{"success": true, "playing": false}, nil
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "playback": data}, nil
	case "pause":
		_, code, err := spotifyDo(ctx, http.MethodPut, "/me/player/pause", token, nil)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": code >= 200 && code < 300, "status_code": code}, nil
	case "play":
		payload := map[string]any{}
		if uri := strings.TrimSpace(strArg(args, "uri")); uri != "" {
			// play specific context/track
			if strings.HasPrefix(uri, "spotify:track:") {
				payload["uris"] = []string{uri}
			} else {
				payload["context_uri"] = uri
			}
		}
		_, code, err := spotifyDo(ctx, http.MethodPut, "/me/player/play", token, payload)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": code >= 200 && code < 300, "status_code": code}, nil
	default:
		return nil, fmt.Errorf("unsupported spotify_playback action: %s", action)
	}
}

func (b *BuiltinTools) spotifyQueue(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		action = "get"
	}
	switch action {
	case "get":
		bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/player/queue", token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "queue": data}, nil
	case "add":
		uri := strings.TrimSpace(strArg(args, "uri"))
		if uri == "" {
			return nil, errors.New("uri required for action=add")
		}
		params := url.Values{}
		params.Set("uri", uri)
		_, code, err := spotifyDo(ctx, http.MethodPost, "/me/player/queue?"+params.Encode(), token, nil)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": code >= 200 && code < 300, "status_code": code}, nil
	default:
		return nil, fmt.Errorf("unsupported spotify_queue action: %s", action)
	}
}

func (b *BuiltinTools) spotifyPlaylists(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/playlists", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var data any
	_ = json.Unmarshal(bs, &data)
	return map[string]any{"success": true, "playlists": data}, nil
}

func (b *BuiltinTools) spotifyAlbums(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/albums", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var data any
	_ = json.Unmarshal(bs, &data)
	return map[string]any{"success": true, "albums": data}, nil
}

func (b *BuiltinTools) spotifyLibrary(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := spotifyToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "spotify not configured (missing env: SPOTIFY_ACCESS_TOKEN)"}, nil
	}
	bs, code, err := spotifyDo(ctx, http.MethodGet, "/me/tracks", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var data any
	_ = json.Unmarshal(bs, &data)
	return map[string]any{"success": true, "tracks": data}, nil
}

func spotifySearchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}, "type": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}, "required": []string{"q"}}
}

func spotifyPlaybackParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string"}, "uri": map[string]any{"type": "string"}}}
}

func spotifyQueueParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string"}, "uri": map[string]any{"type": "string"}}}
}

func spotifyDevicesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func spotifyPlaylistsParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func spotifyAlbumsParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func spotifyLibraryParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

var _ = errors.New
