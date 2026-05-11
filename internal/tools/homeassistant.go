package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func hassConfig() (baseURL string, token string, err error) {
	baseURL = strings.TrimSpace(os.Getenv("HASS_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("HOME_ASSISTANT_URL"))
	}
	token = strings.TrimSpace(os.Getenv("HASS_TOKEN"))
	if baseURL == "" || token == "" {
		return "", "", errors.New("Home Assistant not configured (set HASS_URL and HASS_TOKEN)")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL, token, nil
}

func hassDo(ctx context.Context, method, url, token string, body any) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(bs)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
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

func (b *BuiltinTools) haListEntities(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	baseURL, token, err := hassConfig()
	if err != nil {
		return map[string]any{"success": false, "error": err.Error(), "available": false}, nil
	}
	bs, code, err := hassDo(ctx, http.MethodGet, baseURL+"/api/states", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var out any
	_ = json.Unmarshal(bs, &out)
	return map[string]any{"success": true, "entities": out, "count": sliceLen(out)}, nil
}

func (b *BuiltinTools) haGetState(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	entityID := strings.TrimSpace(strArg(args, "entity_id"))
	if entityID == "" {
		return nil, errors.New("entity_id required")
	}
	baseURL, token, err := hassConfig()
	if err != nil {
		return map[string]any{"success": false, "error": err.Error(), "available": false}, nil
	}
	bs, code, err := hassDo(ctx, http.MethodGet, baseURL+"/api/states/"+entityID, token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var out any
	_ = json.Unmarshal(bs, &out)
	return map[string]any{"success": true, "entity_id": entityID, "state": out}, nil
}

func (b *BuiltinTools) haListServices(ctx context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	baseURL, token, err := hassConfig()
	if err != nil {
		return map[string]any{"success": false, "error": err.Error(), "available": false}, nil
	}
	bs, code, err := hassDo(ctx, http.MethodGet, baseURL+"/api/services", token, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var out any
	_ = json.Unmarshal(bs, &out)
	return map[string]any{"success": true, "services": out, "count": sliceLen(out)}, nil
}

func (b *BuiltinTools) haCallService(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	domain := strings.TrimSpace(strArg(args, "domain"))
	service := strings.TrimSpace(strArg(args, "service"))
	if domain == "" || service == "" {
		return nil, errors.New("domain and service required")
	}
	entityID := strings.TrimSpace(strArg(args, "entity_id"))
	serviceDataAny := args["service_data"]
	serviceData := map[string]any{}
	if m, ok := serviceDataAny.(map[string]any); ok {
		serviceData = m
	}
	if entityID != "" {
		serviceData["entity_id"] = entityID
	}

	baseURL, token, err := hassConfig()
	if err != nil {
		return map[string]any{"success": false, "error": err.Error(), "available": false}, nil
	}
	url := fmt.Sprintf("%s/api/services/%s/%s", baseURL, domain, service)
	bs, code, err := hassDo(ctx, http.MethodPost, url, token, serviceData)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	var out any
	_ = json.Unmarshal(bs, &out)
	return map[string]any{
		"success":     true,
		"domain":      domain,
		"service":     service,
		"entity_id":   entityID,
		"serviceData": serviceData,
		"result":      out,
	}, nil
}

func sliceLen(v any) int {
	switch t := v.(type) {
	case []any:
		return len(t)
	default:
		return 0
	}
}

