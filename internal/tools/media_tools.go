package tools

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (b *BuiltinTools) visionAnalyze(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success": true,
		"path":    path,
		"format":  format,
		"width":   cfg.Width,
		"height":  cfg.Height,
		"note":    "Minimal implementation: returns image metadata only.",
	}, nil
}

func (b *BuiltinTools) imageGenerate(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
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
	return map[string]any{"success": true, "path": path, "note": "Generated a deterministic solid-color PNG (no model backend)."}, nil
}

func (b *BuiltinTools) textToSpeech(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
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
	return map[string]any{"success": true, "path": path, "note": "Placeholder WAV (simple sine beep; no TTS backend)."}, nil
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
