package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

func (b *BuiltinTools) visionAnalyze(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil {
		return nil, err
	}
	question := strings.TrimSpace(strArg(args, "question"))
	if question == "" {
		question = "Describe the image in detail."
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, _ := f.Stat()
	if info != nil && info.Size() > 10<<20 {
		return nil, fmt.Errorf("image too large: %d bytes (limit 10MB)", info.Size())
	}
	bs, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	// Optional real vision via OpenAI chat.completions when configured.
		if apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); apiKey != "" {
		baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model := strings.TrimSpace(os.Getenv("OPENAI_VISION_MODEL"))
		if model == "" {
			model = "gpt-4o-mini"
		}
		mime := "image/" + strings.ToLower(format)
		if format == "jpeg" {
			mime = "image/jpeg"
		}
		dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(bs)
		payload := map[string]any{
			"model": model,
			"messages": []any{
				map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"type": "text", "text": question},
						map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURL}},
					},
				},
			},
			"temperature": 0.2,
		}
		j, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(j))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var out struct {
						Choices []struct {
							Message struct {
								Content string `json:"content"`
							} `json:"message"`
						} `json:"choices"`
					}
					if err := json.Unmarshal(body, &out); err == nil && len(out.Choices) > 0 {
						return map[string]any{
							"success": true,
							"path":    path,
							"format":  format,
							"width":   cfg.Width,
							"height":  cfg.Height,
							"question": question,
							"analysis": out.Choices[0].Message.Content,
							"note":    "Generated via OpenAI chat.completions vision (best-effort).",
						}, nil
					}
					return map[string]any{
						"success": false,
						"error":   "openai vision: unexpected response shape",
						"raw":     string(body),
					}, nil
				}
			}
		}
		// Fall through to metadata-only if the request failed.
	}
	return map[string]any{
		"success": true,
		"path":    path,
		"format":  format,
		"width":   cfg.Width,
		"height":  cfg.Height,
		"question": question,
		"note":    "Fallback implementation: returns image metadata only (set OPENAI_API_KEY to enable vision).",
	}, nil
}

func (b *BuiltinTools) imageGenerate(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	// Hermes parity: image_generate is only available when a backend key is configured.
	// agent-daemon supports either FAL_KEY (placeholder) or OPENAI_API_KEY (real backend).
	if strings.TrimSpace(os.Getenv("FAL_KEY")) == "" && strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
		return map[string]any{
			"success":   false,
			"available": false,
			"error":     "image_generate not configured (set env: OPENAI_API_KEY or FAL_KEY)",
		}, nil
	}
	prompt := strings.TrimSpace(strArg(args, "prompt"))
	if prompt == "" {
		return nil, errors.New("prompt required")
	}
	outPath := strings.TrimSpace(strArg(args, "output_path"))
	if outPath == "" {
		outPath = "generated.png"
	}
	path, err := resolvePathWithinWorkdir(tc.Workdir, outPath)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	// Best-effort real backend: OpenAI images generation when configured.
	if apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); apiKey != "" {
		baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model := strings.TrimSpace(strArg(args, "model"))
		if model == "" {
			model = strings.TrimSpace(os.Getenv("OPENAI_IMAGE_MODEL"))
		}
		if model == "" {
			model = "gpt-image-1"
		}
		size := strings.TrimSpace(strArg(args, "size"))
		if size == "" {
			size = "1024x1024"
		}
		payload := map[string]any{
			"model":           model,
			"prompt":          prompt,
			"size":            size,
			"response_format": "b64_json",
		}
		bs, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/images/generations", bytes.NewReader(bs))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 90 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var outResp struct {
						Data []struct {
							B64 string `json:"b64_json"`
							URL string `json:"url"`
						} `json:"data"`
					}
					if err := json.Unmarshal(body, &outResp); err == nil && len(outResp.Data) > 0 && strings.TrimSpace(outResp.Data[0].B64) != "" {
						imgBytes, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(outResp.Data[0].B64))
						if derr != nil {
							return map[string]any{"success": false, "error": derr.Error()}, nil
						}
						if err := os.WriteFile(path, imgBytes, 0o644); err != nil {
							return nil, err
						}
						out := map[string]any{
							"success":  true,
							"path":     path,
							"media":    "MEDIA: " + path,
							"model":    model,
							"size":     size,
							"note":     "Generated via OpenAI images/generations (best-effort).",
						}
						deliver := boolArg(args, "deliver", false)
						caption := strings.TrimSpace(strArg(args, "caption"))
						out = maybeDeliverMedia(ctx, out, tc, deliver, path, caption)
						return out, nil
					}
					return map[string]any{"success": false, "error": "openai image: unexpected response shape", "raw": string(body)}, nil
				}
			}
		}
		// Fall through to placeholder if OpenAI call fails.
	}

	// Placeholder output (requires FAL_KEY to be set for "available").
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	// Deterministic background color derived from prompt.
	hash := uint32(2166136261)
	for i := 0; i < len(prompt); i++ {
		hash ^= uint32(prompt[i])
		hash *= 16777619
	}
	bg := color.RGBA{R: byte(hash >> 16), G: byte(hash >> 8), B: byte(hash), A: 255}
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			img.Set(x, y, bg)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	out := map[string]any{"success": true, "path": path, "media": "MEDIA: " + path, "note": "Placeholder output: deterministic solid-color PNG (no real backend available)."}
	deliver := boolArg(args, "deliver", false)
	caption := strings.TrimSpace(strArg(args, "caption"))
	out = maybeDeliverMedia(ctx, out, tc, deliver, path, caption)
	return out, nil
}

func (b *BuiltinTools) textToSpeech(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	text := strings.TrimSpace(strArg(args, "text"))
	if text == "" {
		return nil, errors.New("text required")
	}
	outPath := strings.TrimSpace(strArg(args, "output_path"))
	if outPath == "" {
		outPath = "speech.wav"
	}
	path, err := resolvePathWithinWorkdir(tc.Workdir, outPath)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	// Best-effort "real" TTS via OpenAI audio/speech when configured.
	// Otherwise falls back to a placeholder beep WAV.
	deliver := boolArg(args, "deliver", false)
	if apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); apiKey != "" {
		baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model := strings.TrimSpace(os.Getenv("OPENAI_TTS_MODEL"))
		if model == "" {
			model = "gpt-4o-mini-tts"
		}
		voice := strings.TrimSpace(os.Getenv("OPENAI_TTS_VOICE"))
		if voice == "" {
			voice = "alloy"
		}
		format := strings.ToLower(strings.TrimSpace(strArg(args, "format")))
		if format == "" {
			// Hermes often delivers as a voice message; mp3 is widely supported.
			format = "mp3"
		}
		if format != "mp3" && format != "wav" && format != "opus" && format != "aac" {
			return nil, fmt.Errorf("unsupported format: %s (supported: mp3,wav,opus,aac)", format)
		}

		payload := map[string]any{
			"model":  model,
			"voice":  voice,
			"input":  text,
			"format": format,
		}
		bs, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/audio/speech", bytes.NewReader(bs))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 45 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode >= 200 && resp.StatusCode < 300 && len(body) > 0 {
					if err := os.WriteFile(path, body, 0o644); err != nil {
						return nil, err
					}
					out := map[string]any{
						"success": true,
						"path":    path,
						"media":   "MEDIA: " + path,
						"format":  format,
						"note":    "Generated via OpenAI audio/speech.",
					}
					out = maybeDeliverMedia(ctx, out, tc, deliver, path, "")
					return out, nil
				}
			}
		}
	}

	seconds := float64(len([]rune(text))) / 12.0
	if seconds < 0.5 {
		seconds = 0.5
	}
	if seconds > 10 {
		seconds = 10
	}
	wav := synthBeepWav(time.Duration(seconds * float64(time.Second)))
	if err := os.WriteFile(path, wav, 0o644); err != nil {
		return nil, err
	}
	out := map[string]any{"success": true, "path": path, "media": "MEDIA: " + path, "format": "wav", "note": "Placeholder WAV (simple sine beep; no TTS backend)."}
	out = maybeDeliverMedia(ctx, out, tc, deliver, path, "")
	return out, nil
}

func maybeDeliverMedia(ctx context.Context, out map[string]any, tc ToolContext, deliver bool, path, caption string) map[string]any {
	if !deliver {
		return out
	}
	p := strings.TrimSpace(tc.GatewayPlatform)
	chatID := strings.TrimSpace(tc.GatewayChatID)
	if p == "" || chatID == "" {
		out["delivered"] = false
		out["delivery_error"] = "deliver=true requires a gateway context (platform/chat_id)"
		return out
	}
	a, ok := platform.Get(strings.ToLower(p))
	if !ok {
		out["delivered"] = false
		out["delivery_error"] = "platform adapter not connected: " + p
		return out
	}
	ms, ok := a.(platform.MediaSender)
	if !ok {
		out["delivered"] = false
		out["delivery_error"] = "platform adapter does not support media delivery: " + p
		return out
	}
	res, err := ms.SendMedia(ctx, chatID, path, caption, strings.TrimSpace(tc.GatewayMessageID))
	if err != nil {
		out["delivered"] = false
		out["delivery_error"] = err.Error()
		return out
	}
	if !res.Success && strings.TrimSpace(res.Error) != "" {
		out["delivered"] = false
		out["delivery_error"] = res.Error
		return out
	}
	out["delivered"] = true
	out["delivery_platform"] = strings.ToLower(p)
	out["delivery_chat_id"] = chatID
	out["delivery_message_id"] = res.MessageID
	return out
}

func synthBeepWav(d time.Duration) []byte {
	// 16-bit mono PCM @ 16000 Hz, silence.
	sampleRate := uint32(16000)
	numSamples := uint32(d.Seconds() * float64(sampleRate))
	if numSamples < 1 {
		numSamples = 1
	}
	bitsPerSample := uint16(16)
	numChannels := uint16(1)
	blockAlign := numChannels * (bitsPerSample / 8)
	byteRate := sampleRate * uint32(blockAlign)
	dataSize := uint32(numSamples) * uint32(blockAlign)

	var buf bytes.Buffer
	// RIFF header
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36)+dataSize)
	buf.WriteString("WAVE")
	// fmt chunk
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16)) // PCM fmt chunk size
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM
	_ = binary.Write(&buf, binary.LittleEndian, numChannels)
	_ = binary.Write(&buf, binary.LittleEndian, sampleRate)
	_ = binary.Write(&buf, binary.LittleEndian, byteRate)
	_ = binary.Write(&buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(&buf, binary.LittleEndian, bitsPerSample)
	// data chunk
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, dataSize)
	// samples
	freq := 440.0
	amp := 0.12
	for i := uint32(0); i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		v := int16(math.Sin(2*math.Pi*freq*t) * 32767 * amp)
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}
