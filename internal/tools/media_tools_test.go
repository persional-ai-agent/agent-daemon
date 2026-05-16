package tools

import (
	"context"
	"encoding/json"
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

func TestTranscriptionParamsIncludeStrictBackend(t *testing.T) {
	props, _ := transcriptionParams()["properties"].(map[string]any)
	if _, ok := props["strict_backend"]; !ok {
		t.Fatal("expected strict_backend property in transcription params")
	}
	if _, ok := props["path"]; !ok {
		t.Fatal("expected path property in transcription params")
	}
}

func TestTranscriptionStrictBackendWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	b := &BuiltinTools{}
	wd := t.TempDir()
	audio := filepath.Join(wd, "memo.wav")
	if err := os.WriteFile(audio, []byte("not-real-audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := b.transcription(context.Background(), map[string]any{
		"path":           "memo.wav",
		"strict_backend": true,
	}, ToolContext{Workdir: wd})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := out["success"].(bool); ok {
		t.Fatalf("expected strict backend failure, got: %#v", out)
	}
}

func TestTranscriptionOpenAIBackend(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/audio/transcriptions") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "hello world"})
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "k-test")
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	b := &BuiltinTools{}
	wd := t.TempDir()
	audio := filepath.Join(wd, "memo.wav")
	if err := os.WriteFile(audio, []byte("fake-audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := b.transcription(context.Background(), map[string]any{
		"path":        "memo.wav",
		"output_path": "memo.txt",
	}, ToolContext{Workdir: wd})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := out["success"].(bool); !ok {
		t.Fatalf("expected success, got: %#v", out)
	}
	if text, _ := out["text"].(string); text != "hello world" {
		t.Fatalf("unexpected text: %#v", out)
	}
	txtPath, _ := out["output_path"].(string)
	bs, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(bs)) != "hello world" {
		t.Fatalf("unexpected transcript file: %q", string(bs))
	}
}
