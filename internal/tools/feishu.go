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
	"sync"
	"time"
)

type feishuTokenCache struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

var feishuTenantTokenCache feishuTokenCache

func feishuConfig() (baseURL, appID, appSecret string, err error) {
	baseURL = strings.TrimSpace(os.Getenv("FEISHU_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://open.feishu.cn"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	appID = strings.TrimSpace(os.Getenv("FEISHU_APP_ID"))
	appSecret = strings.TrimSpace(os.Getenv("FEISHU_APP_SECRET"))
	if appID == "" || appSecret == "" {
		return "", "", "", errors.New("Feishu not configured (set FEISHU_APP_ID and FEISHU_APP_SECRET)")
	}
	return baseURL, appID, appSecret, nil
}

func feishuTenantToken(ctx context.Context) (string, error) {
	feishuTenantTokenCache.mu.Lock()
	if feishuTenantTokenCache.token != "" && time.Now().Before(feishuTenantTokenCache.expires.Add(-30*time.Second)) {
		tok := feishuTenantTokenCache.token
		feishuTenantTokenCache.mu.Unlock()
		return tok, nil
	}
	feishuTenantTokenCache.mu.Unlock()

	baseURL, appID, appSecret, err := feishuConfig()
	if err != nil {
		return "", err
	}
	payload := map[string]any{"app_id": appID, "app_secret": appSecret}
	bs, _, err := feishuDo(ctx, http.MethodPost, baseURL+"/open-apis/auth/v3/tenant_access_token/internal/", "", payload)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Token  string `json:"tenant_access_token"`
		Expire int    `json:"expire"`
	}
	if err := json.Unmarshal(bs, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 || resp.Token == "" {
		return "", fmt.Errorf("feishu token error: code=%d msg=%s", resp.Code, resp.Msg)
	}
	exp := time.Now().Add(time.Duration(resp.Expire) * time.Second)
	feishuTenantTokenCache.mu.Lock()
	feishuTenantTokenCache.token = resp.Token
	feishuTenantTokenCache.expires = exp
	feishuTenantTokenCache.mu.Unlock()
	return resp.Token, nil
}

func feishuDo(ctx context.Context, method, fullURL, tenantToken string, body any) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(bs)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, r)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if tenantToken != "" {
		req.Header.Set("Authorization", "Bearer "+tenantToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return out, resp.StatusCode, nil
}

func feishuOK(bs []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(bs, &m); err != nil {
		return nil, err
	}
	if code, ok := m["code"].(float64); ok && int(code) != 0 {
		msg, _ := m["msg"].(string)
		return nil, fmt.Errorf("feishu api error: code=%d msg=%s", int(code), msg)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// feishu_doc_read
// ---------------------------------------------------------------------------

func (b *BuiltinTools) feishuDocRead(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	docToken := strings.TrimSpace(strArg(args, "doc_token"))
	if docToken == "" {
		return nil, errors.New("doc_token required")
	}
	baseURL, _, _, err := feishuConfig()
	if err != nil {
		return map[string]any{"success": false, "available": false, "error": err.Error()}, nil
	}
	tok, err := feishuTenantToken(ctx)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	u := baseURL + "/open-apis/docx/v1/documents/" + url.PathEscape(docToken) + "/raw_content"
	bs, code, err := feishuDo(ctx, http.MethodGet, u, tok, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	m, err := feishuOK(bs)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	data, _ := m["data"].(map[string]any)
	content := ""
	if data != nil {
		if v, ok := data["content"].(string); ok {
			content = v
		}
	}
	return map[string]any{"success": true, "doc_token": docToken, "content": content}, nil
}

func feishuDocReadParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"doc_token": map[string]any{"type": "string"}}, "required": []string{"doc_token"}}
}

// ---------------------------------------------------------------------------
// Drive comments
// ---------------------------------------------------------------------------

func (b *BuiltinTools) feishuDriveListComments(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	fileToken := strings.TrimSpace(strArg(args, "file_token"))
	if fileToken == "" {
		return nil, errors.New("file_token required")
	}
	baseURL, _, _, err := feishuConfig()
	if err != nil {
		return map[string]any{"success": false, "available": false, "error": err.Error()}, nil
	}
	tok, err := feishuTenantToken(ctx)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	fileType := strings.TrimSpace(strArg(args, "file_type"))
	if fileType == "" {
		fileType = "docx"
	}
	pageSize := intArg(args, "page_size", 100)
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 100 {
		pageSize = 100
	}
	pageToken := strings.TrimSpace(strArg(args, "page_token"))
	isWhole := boolArg(args, "is_whole", false)

	q := url.Values{}
	q.Set("file_type", fileType)
	q.Set("user_id_type", "open_id")
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	if isWhole {
		q.Set("is_whole", "true")
	}
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	u := baseURL + "/open-apis/drive/v1/files/" + url.PathEscape(fileToken) + "/comments?" + q.Encode()
	bs, code, err := feishuDo(ctx, http.MethodGet, u, tok, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	m, err := feishuOK(bs)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "data": m["data"]}, nil
}

func (b *BuiltinTools) feishuDriveListCommentReplies(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	fileToken := strings.TrimSpace(strArg(args, "file_token"))
	commentID := strings.TrimSpace(strArg(args, "comment_id"))
	if fileToken == "" || commentID == "" {
		return nil, errors.New("file_token and comment_id required")
	}
	baseURL, _, _, err := feishuConfig()
	if err != nil {
		return map[string]any{"success": false, "available": false, "error": err.Error()}, nil
	}
	tok, err := feishuTenantToken(ctx)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	fileType := strings.TrimSpace(strArg(args, "file_type"))
	if fileType == "" {
		fileType = "docx"
	}
	pageSize := intArg(args, "page_size", 100)
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 100 {
		pageSize = 100
	}
	pageToken := strings.TrimSpace(strArg(args, "page_token"))

	q := url.Values{}
	q.Set("file_type", fileType)
	q.Set("user_id_type", "open_id")
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	u := baseURL + "/open-apis/drive/v1/files/" + url.PathEscape(fileToken) + "/comments/" + url.PathEscape(commentID) + "/replies?" + q.Encode()
	bs, code, err := feishuDo(ctx, http.MethodGet, u, tok, nil)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	m, err := feishuOK(bs)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "data": m["data"]}, nil
}

func (b *BuiltinTools) feishuDriveReplyComment(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	fileToken := strings.TrimSpace(strArg(args, "file_token"))
	commentID := strings.TrimSpace(strArg(args, "comment_id"))
	content := strings.TrimSpace(strArg(args, "content"))
	if fileToken == "" || commentID == "" || content == "" {
		return nil, errors.New("file_token, comment_id, and content required")
	}
	baseURL, _, _, err := feishuConfig()
	if err != nil {
		return map[string]any{"success": false, "available": false, "error": err.Error()}, nil
	}
	tok, err := feishuTenantToken(ctx)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	fileType := strings.TrimSpace(strArg(args, "file_type"))
	if fileType == "" {
		fileType = "docx"
	}
	q := url.Values{}
	q.Set("file_type", fileType)

	body := map[string]any{
		"content": map[string]any{
			"elements": []any{
				map[string]any{"type": "text_run", "text_run": map[string]any{"text": content}},
			},
		},
	}
	u := baseURL + "/open-apis/drive/v1/files/" + url.PathEscape(fileToken) + "/comments/" + url.PathEscape(commentID) + "/replies?" + q.Encode()
	bs, code, err := feishuDo(ctx, http.MethodPost, u, tok, body)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	m, err := feishuOK(bs)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "data": m["data"]}, nil
}

func (b *BuiltinTools) feishuDriveAddComment(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	fileToken := strings.TrimSpace(strArg(args, "file_token"))
	content := strings.TrimSpace(strArg(args, "content"))
	if fileToken == "" || content == "" {
		return nil, errors.New("file_token and content required")
	}
	baseURL, _, _, err := feishuConfig()
	if err != nil {
		return map[string]any{"success": false, "available": false, "error": err.Error()}, nil
	}
	tok, err := feishuTenantToken(ctx)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	fileType := strings.TrimSpace(strArg(args, "file_type"))
	if fileType == "" {
		fileType = "docx"
	}
	body := map[string]any{
		"file_type": fileType,
		"reply_elements": []any{
			map[string]any{"type": "text", "text": content},
		},
	}
	u := baseURL + "/open-apis/drive/v1/files/" + url.PathEscape(fileToken) + "/new_comments"
	bs, code, err := feishuDo(ctx, http.MethodPost, u, tok, body)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
	}
	m, err := feishuOK(bs)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "data": m["data"]}, nil
}

func feishuDriveListCommentsParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"file_token": map[string]any{"type": "string"}, "file_type": map[string]any{"type": "string", "description": "Drive file type (default docx)."}, "is_whole": map[string]any{"type": "boolean", "description": "Whether to fetch comments for whole document (default false)."}, "page_size": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "Page size (default 100, max 100)."}, "page_token": map[string]any{"type": "string"}}, "required": []string{"file_token"}}
}

func feishuDriveListCommentRepliesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"file_token": map[string]any{"type": "string"}, "comment_id": map[string]any{"type": "string"}, "file_type": map[string]any{"type": "string", "description": "Drive file type (default docx)."}, "page_size": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "Page size (default 100, max 100)."}, "page_token": map[string]any{"type": "string"}}, "required": []string{"file_token", "comment_id"}}
}

func feishuDriveReplyCommentParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"file_token": map[string]any{"type": "string"}, "comment_id": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "file_type": map[string]any{"type": "string", "description": "Drive file type (default docx)."}}, "required": []string{"file_token", "comment_id", "content"}}
}

func feishuDriveAddCommentParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"file_token": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "file_type": map[string]any{"type": "string", "description": "Drive file type (default docx)."}}, "required": []string{"file_token", "content"}}
}
