package yuanbao

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Minimal COS upload flow ported from Hermes:
// /data/source/hermes-agent/gateway/platforms/yuanbao_media.py

const uploadInfoPath = "/api/resource/genUploadInfo"

type UploadInfo struct {
	BucketName         string `json:"bucketName"`
	Region             string `json:"region"`
	Location           string `json:"location"`
	EncryptTmpSecretID string `json:"encryptTmpSecretId"`
	EncryptTmpSecretKey string `json:"encryptTmpSecretKey"`
	EncryptToken       string `json:"encryptToken"`
	StartTime          int64  `json:"startTime"`
	ExpiredTime        int64  `json:"expiredTime"`
	ResourceURL        string `json:"resourceUrl"`
	ResourceID         string `json:"resourceID"`
}

func GetUploadInfo(ctx context.Context, cfg EnvConfig, filename, fileID string) (UploadInfo, error) {
	if strings.TrimSpace(cfg.APIDomain) == "" {
		return UploadInfo{}, errors.New("api_domain required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return UploadInfo{}, errors.New("token required")
	}
	if strings.TrimSpace(cfg.BotID) == "" && strings.TrimSpace(cfg.AppID) == "" {
		return UploadInfo{}, errors.New("bot_id required (or app_id for x-id)")
	}
	if strings.TrimSpace(filename) == "" {
		filename = "file"
	}
	if strings.TrimSpace(fileID) == "" {
		fileID = randomHex(16)
	}

	body := map[string]any{
		"fileName":   filename,
		"fileId":     fileID,
		"docFrom":    "localDoc",
		"docOpenId":  "",
	}
	bs, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.APIDomain, "/")+uploadInfoPath, bytes.NewReader(bs))
	if err != nil {
		return UploadInfo{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", cfg.Token)
	xid := strings.TrimSpace(cfg.BotID)
	if xid == "" {
		xid = strings.TrimSpace(cfg.AppID)
	}
	req.Header.Set("X-ID", xid)
	req.Header.Set("X-Source", "web")
	if strings.TrimSpace(cfg.RouteEnv) != "" {
		req.Header.Set("X-Route-Env", strings.TrimSpace(cfg.RouteEnv))
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return UploadInfo{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UploadInfo{}, fmt.Errorf("genUploadInfo http status %d: %s", resp.StatusCode, string(raw))
	}

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return UploadInfo{}, err
	}
	if envelope.Code != 0 {
		return UploadInfo{}, fmt.Errorf("genUploadInfo error: code=%d msg=%s", envelope.Code, envelope.Msg)
	}

	var info UploadInfo
	if len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, &info); err != nil {
			return UploadInfo{}, err
		}
	} else {
		// Some implementations may return fields at top level.
		if err := json.Unmarshal(raw, &info); err != nil {
			return UploadInfo{}, err
		}
	}
	if strings.TrimSpace(info.BucketName) == "" || strings.TrimSpace(info.Location) == "" {
		return UploadInfo{}, errors.New("genUploadInfo returned incomplete data (missing bucketName/location)")
	}
	if strings.TrimSpace(info.Region) == "" {
		// Hermes tolerates missing region; default to ap-guangzhou as a conservative guess.
		info.Region = "ap-guangzhou"
	}
	return info, nil
}

type UploadResult struct {
	URL      string
	UUID     string
	Size     int
	Width    int
	Height   int
	MimeType string
	FileName string
}

type kv struct{ k, v string }

func UploadToCOS(ctx context.Context, info UploadInfo, path string) (UploadResult, error) {
	// Keep this minimal: read whole file (<= 50MB) for MD5 and optional image size decode.
	f, err := os.Open(path)
	if err != nil {
		return UploadResult{}, err
	}
	defer f.Close()

	// Size cap: 50MB (Hermes default).
	const maxBytes = 50 << 20
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, f, maxBytes+1); err != nil && !errors.Is(err, io.EOF) {
		return UploadResult{}, err
	}
	if buf.Len() > maxBytes {
		return UploadResult{}, fmt.Errorf("file too large: %d bytes > %d", buf.Len(), maxBytes)
	}
	data := buf.Bytes()

	filename := filepath.Base(path)
	mimeType := guessMimeType(filename)
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/octet-stream"
	}

	fileUUID := md5Hex(data)
	resURL := strings.TrimSpace(info.ResourceURL)

	cosHost := fmt.Sprintf("%s.cos.accelerate.myqcloud.com", info.BucketName)
	encodedKey := url.PathEscape(strings.TrimLeft(info.Location, "/"))
	encodedKey = strings.ReplaceAll(encodedKey, "%2F", "/") // keep '/'
	cosURL := "https://" + cosHost + "/" + encodedKey

	if resURL == "" {
		resURL = cosURL
	}

	signHeaders := map[string]string{
		"host":                 cosHost,
		"content-type":         mimeType,
		"x-cos-security-token": strings.TrimSpace(info.EncryptToken),
	}
	now := time.Now().Unix()
	signStart := info.StartTime
	if signStart <= 0 {
		signStart = now
	}
	expireSeconds := int64(3600)
	if info.ExpiredTime > now {
		expireSeconds = info.ExpiredTime - now
	}
	auth := cosSign("put", "/"+encodedKey, nil, signHeaders, info.EncryptTmpSecretID, info.EncryptTmpSecretKey, signStart, expireSeconds)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, cosURL, bytes.NewReader(data))
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", mimeType)
	if strings.TrimSpace(info.EncryptToken) != "" {
		req.Header.Set("x-cos-security-token", strings.TrimSpace(info.EncryptToken))
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return UploadResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return UploadResult{}, fmt.Errorf("cos put http status %d: %s", resp.StatusCode, string(raw))
	}

	width, height := 0, 0
	if strings.HasPrefix(mimeType, "image/") {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
			width, height = cfg.Width, cfg.Height
		}
	}

	return UploadResult{
		URL:      resURL,
		UUID:     fileUUID,
		Size:     len(data),
		Width:    width,
		Height:   height,
		MimeType: mimeType,
		FileName: filename,
	}, nil
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func guessMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func ImageFormatFromMime(mimeType string) uint32 {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return 1
	case "image/gif":
		return 2
	case "image/png":
		return 3
	case "image/bmp":
		return 4
	default:
		return 255
	}
}

func cosSign(method, path string, params map[string]string, headers map[string]string, secretID, secretKey string, startTime, expireSeconds int64) string {
	if params == nil {
		params = map[string]string{}
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if startTime <= 0 {
		startTime = time.Now().Unix()
	}
	if expireSeconds <= 0 {
		expireSeconds = 3600
	}
	qSignTime := fmt.Sprintf("%d;%d", startTime, startTime+expireSeconds)

	// SignKey = HMAC-SHA1(SecretKey, q-sign-time)
	signKey := hmacSha1Hex([]byte(secretKey), []byte(qSignTime))

	paramPairs := make([]kv, 0, len(params))
	for k, v := range params {
		paramPairs = append(paramPairs, kv{k: strings.ToLower(k), v: url.QueryEscape(v)})
	}
	headerPairs := make([]kv, 0, len(headers))
	for k, v := range headers {
		headerPairs = append(headerPairs, kv{k: strings.ToLower(k), v: url.QueryEscape(v)})
	}
	sort.Slice(paramPairs, func(i, j int) bool { return paramPairs[i].k < paramPairs[j].k })
	sort.Slice(headerPairs, func(i, j int) bool { return headerPairs[i].k < headerPairs[j].k })

	urlParamList := joinKeys(paramPairs, ";")
	urlParams := joinPairs(paramPairs, "&")
	headerList := joinKeys(headerPairs, ";")
	headerStr := joinPairs(headerPairs, "&")

	httpString := strings.Join([]string{
		strings.ToLower(method),
		strings.ToLower(path),
		urlParams,
		headerStr,
		"",
	}, "\n")

	sha1OfHTTP := sha1Hex([]byte(httpString))
	stringToSign := strings.Join([]string{
		"sha1",
		qSignTime,
		sha1OfHTTP,
		"",
	}, "\n")

	// Signature = HMAC-SHA1(SignKey, StringToSign)
	sig := hmacSha1Hex([]byte(signKey), []byte(stringToSign))

	return "q-sign-algorithm=sha1" +
		"&q-ak=" + secretID +
		"&q-sign-time=" + qSignTime +
		"&q-key-time=" + qSignTime +
		"&q-header-list=" + headerList +
		"&q-url-param-list=" + urlParamList +
		"&q-signature=" + sig
}

func sha1Hex(b []byte) string {
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}

func hmacSha1Hex(key, msg []byte) string {
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}

func joinKeys(pairs []kv, sep string) string {
	if len(pairs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, p := range pairs {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(p.k)
	}
	return sb.String()
}

func joinPairs(pairs []kv, sep string) string {
	if len(pairs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, p := range pairs {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(p.k)
		sb.WriteString("=")
		sb.WriteString(p.v)
	}
	return sb.String()
}
