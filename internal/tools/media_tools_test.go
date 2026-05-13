package tools

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTryWriteImageFromURL(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0})
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "img.bin")
	if err := tryWriteImageFromURL(context.Background(), srv.URL, out); err != nil {
		t.Fatalf("tryWriteImageFromURL failed: %v", err)
	}
	bs, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) == 0 {
		t.Fatal("expected non-empty downloaded file")
	}
}

func TestTextToSpeechParamsIncludeModelAndVoice(t *testing.T) {
	props, _ := textToSpeechParams()["properties"].(map[string]any)
	if _, ok := props["model"]; !ok {
		t.Fatal("expected model property in text_to_speech params")
	}
	if _, ok := props["voice"]; !ok {
		t.Fatal("expected voice property in text_to_speech params")
	}
	format, _ := props["format"].(map[string]any)
	desc, _ := format["description"].(string)
	if !strings.Contains(desc, "mp3") {
		t.Fatalf("unexpected format description: %q", desc)
	}
	if _, ok := props["strict_backend"]; !ok {
		t.Fatal("expected strict_backend property in text_to_speech params")
	}
}

func TestMediaParamsIncludeStrictBackend(t *testing.T) {
	vProps, _ := visionAnalyzeParams()["properties"].(map[string]any)
	if _, ok := vProps["strict_backend"]; !ok {
		t.Fatal("vision_analyze params should contain strict_backend")
	}
	iProps, _ := imageGenerateParams()["properties"].(map[string]any)
	if _, ok := iProps["strict_backend"]; !ok {
		t.Fatal("image_generate params should contain strict_backend")
	}
}

func TestTextToSpeechStrictBackendWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	b := &BuiltinTools{}
	out, err := b.textToSpeech(context.Background(), map[string]any{
		"text":           "hello",
		"output_path":    "speech.wav",
		"strict_backend": true,
	}, ToolContext{Workdir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := out["success"].(bool); ok {
		t.Fatalf("expected strict backend failure, got: %#v", out)
	}
}
