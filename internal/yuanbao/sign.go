package yuanbao

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type SignToken struct {
	Token    string
	BotID    string
	ExpireAt time.Time
}

type EnvConfig struct {
	AppID     string
	AppSecret string
	APIDomain string
	WSURL     string
	Token     string
	BotID     string
	RouteEnv  string
}

func ConfigFromEnv() (EnvConfig, error) {
	cfg := EnvConfig{
		AppID:     strings.TrimSpace(os.Getenv("YUANBAO_APP_ID")),
		AppSecret: strings.TrimSpace(os.Getenv("YUANBAO_APP_SECRET")),
		APIDomain: strings.TrimSpace(os.Getenv("YUANBAO_API_DOMAIN")),
		WSURL:     strings.TrimSpace(os.Getenv("YUANBAO_WS_URL")),
		Token:     strings.TrimSpace(os.Getenv("YUANBAO_TOKEN")),
		BotID:     strings.TrimSpace(os.Getenv("YUANBAO_BOT_ID")),
		RouteEnv:  strings.TrimSpace(os.Getenv("YUANBAO_ROUTE_ENV")),
	}
	if cfg.APIDomain == "" {
		cfg.APIDomain = "https://bot.yuanbao.tencent.com"
	}
	if cfg.WSURL == "" {
		cfg.WSURL = "wss://bot-wss.yuanbao.tencent.com/wss/connection"
	}
	cfg.APIDomain = strings.TrimRight(cfg.APIDomain, "/")

	if cfg.Token != "" {
		return cfg, nil
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return EnvConfig{}, errors.New("yuanbao credentials missing: set YUANBAO_TOKEN or (YUANBAO_APP_ID and YUANBAO_APP_SECRET)")
	}
	return cfg, nil
}

func signature(nonce, timestamp, appID, appSecret string) string {
	plain := nonce + timestamp + appID + appSecret
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(plain))
	return hex.EncodeToString(mac.Sum(nil))
}

func timestampBJ() string {
	loc := time.FixedZone("CST-8", 8*3600)
	return time.Now().In(loc).Format("2006-01-02T15:04:05-07:00")
}

func FetchSignToken(ctx context.Context, appID, appSecret, apiDomain string) (SignToken, error) {
	type respT struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}

	nonce := randomHex(16)
	ts := timestampBJ()
	sig := signature(nonce, ts, appID, appSecret)
	payload := map[string]any{
		"app_key":   appID,
		"nonce":     nonce,
		"signature": sig,
		"timestamp": ts,
	}
	bs, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiDomain, "/")+"/api/v5/robotLogic/sign-token", bytes.NewReader(bs))
	if err != nil {
		return SignToken{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return SignToken{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SignToken{}, fmt.Errorf("sign-token http status %d: %s", resp.StatusCode, string(body))
	}
	var r respT
	if err := json.Unmarshal(body, &r); err != nil {
		return SignToken{}, err
	}
	if r.Code != 0 {
		return SignToken{}, fmt.Errorf("sign-token error: code=%d msg=%s", r.Code, r.Msg)
	}
	token, _ := r.Data["token"].(string)
	botID, _ := r.Data["bot_id"].(string)
	durationF, _ := r.Data["duration"].(float64)
	dur := time.Duration(durationF) * time.Second
	if dur <= 0 {
		dur = time.Hour
	}
	return SignToken{Token: token, BotID: botID, ExpireAt: time.Now().Add(dur)}, nil
}

func randomHex(nBytes int) string {
	now := time.Now().UnixNano()
	b := make([]byte, 0, nBytes)
	for i := 0; i < nBytes; i++ {
		now = now*1664525 + 1013904223
		b = append(b, byte(now>>24))
	}
	return hex.EncodeToString(b)
}

