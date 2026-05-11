package tools

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

type yuanbaoSignToken struct {
	Token    string
	BotID    string
	ExpireAt time.Time
}

func yuanbaoConfig() (appID, appSecret, apiDomain, wsURL string, err error) {
	appID = strings.TrimSpace(os.Getenv("YUANBAO_APP_ID"))
	appSecret = strings.TrimSpace(os.Getenv("YUANBAO_APP_SECRET"))
	apiDomain = strings.TrimSpace(os.Getenv("YUANBAO_API_DOMAIN"))
	wsURL = strings.TrimSpace(os.Getenv("YUANBAO_WS_URL"))
	if apiDomain == "" {
		apiDomain = "https://bot.yuanbao.tencent.com"
	}
	if wsURL == "" {
		wsURL = "wss://bot-wss.yuanbao.tencent.com/wss/connection"
	}
	if appID == "" || appSecret == "" {
		return "", "", "", "", errors.New("YUANBAO_APP_ID and YUANBAO_APP_SECRET required")
	}
	return appID, appSecret, strings.TrimRight(apiDomain, "/"), wsURL, nil
}

func yuanbaoSignature(nonce, timestamp, appID, appSecret string) string {
	plain := nonce + timestamp + appID + appSecret
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(plain))
	return hex.EncodeToString(mac.Sum(nil))
}

func yuanbaoTimestampBJ() string {
	// Format: 2006-01-02T15:04:05+08:00
	loc := time.FixedZone("CST-8", 8*3600)
	return time.Now().In(loc).Format("2006-01-02T15:04:05-07:00")
}

func yuanbaoFetchSignToken(ctx context.Context) (yuanbaoSignToken, error) {
	appID, appSecret, apiDomain, _, err := yuanbaoConfig()
	if err != nil {
		return yuanbaoSignToken{}, err
	}
	type respT struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data map[string]any    `json:"data"`
	}

	nonce := randomHex(16)
	ts := yuanbaoTimestampBJ()
	sig := yuanbaoSignature(nonce, ts, appID, appSecret)
	payload := map[string]any{
		"app_key":   appID,
		"nonce":     nonce,
		"signature": sig,
		"timestamp": ts,
	}
	bs, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiDomain+"/api/v5/robotLogic/sign-token", bytes.NewReader(bs))
	if err != nil {
		return yuanbaoSignToken{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return yuanbaoSignToken{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return yuanbaoSignToken{}, fmt.Errorf("sign-token http status %d: %s", resp.StatusCode, string(body))
	}
	var r respT
	if err := json.Unmarshal(body, &r); err != nil {
		return yuanbaoSignToken{}, err
	}
	if r.Code != 0 {
		return yuanbaoSignToken{}, fmt.Errorf("sign-token error: code=%d msg=%s", r.Code, r.Msg)
	}
	token, _ := r.Data["token"].(string)
	botID, _ := r.Data["bot_id"].(string)
	durationF, _ := r.Data["duration"].(float64)
	dur := time.Duration(durationF) * time.Second
	if dur <= 0 {
		dur = time.Hour
	}
	return yuanbaoSignToken{Token: token, BotID: botID, ExpireAt: time.Now().Add(dur)}, nil
}

func randomHex(nBytes int) string {
	// best-effort; avoids adding crypto/rand dependency here.
	now := time.Now().UnixNano()
	b := make([]byte, 0, nBytes)
	for i := 0; i < nBytes; i++ {
		now = now*1664525 + 1013904223
		b = append(b, byte(now>>24))
	}
	return hex.EncodeToString(b)
}

