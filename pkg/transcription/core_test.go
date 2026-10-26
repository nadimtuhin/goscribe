package transcription

import (
	"strings"
	"testing"
)

func TestFormatTimestamp(t *testing.T) {
	if got := FormatTimestamp(0); got != "00:00" {
		t.Errorf("got %q, want 00:00", got)
	}
	if got := FormatTimestamp(65000); got != "01:05" {
		t.Errorf("got %q, want 01:05", got)
	}
	if got := FormatTimestamp(3661000); got != "61:01" {
		t.Errorf("got %q, want 61:01", got)
	}
}

func TestIsNonSpeech(t *testing.T) {
	if !IsNonSpeech("") {
		t.Errorf("expected empty string to be non-speech")
	}
	if !IsNonSpeech("[BLANK_AUDIO]") {
		t.Errorf("expected [BLANK_AUDIO] to be non-speech")
	}
	if !IsNonSpeech("(applause)") {
		t.Errorf("expected (applause) to be non-speech")
	}
	if !IsNonSpeech("thank you") {
		t.Errorf("expected thank you to be non-speech")
	}
	if IsNonSpeech("Welcome to the meeting") {
		t.Errorf("expected real speech not to be non-speech")
	}
}

func TestDevanagariToBengali(t *testing.T) {
	got := DevanagariToBengali("व")
	if got != "ব" {
		t.Errorf("got %q, want ব", got)
	}
	dev := "नमस्ते"
	bng := DevanagariToBengali(dev)
	if bng == dev {
		t.Errorf("expected translation, got %q", bng)
	}
}

func TestSegmentsToMarkdown(t *testing.T) {
	segments := []WhisperSegment{
		{From: 1000, To: 3000, Text: "  Hello everyone  "},
		{From: 4000, To: 5000, Text: "[BLANK_AUDIO]"},
		{From: 6000, To: 8000, Text: "Let's begin the review"},
	}
	md := SegmentsToMarkdown(segments, "en")
	if !strings.Contains(md, "[00:01] Hello everyone\n") {
		t.Errorf("expected segment 1 in markdown, got:\n%s", md)
	}
	if strings.Contains(md, "BLANK_AUDIO") {
		t.Errorf("expected non-speech filtered out")
	}
	if !strings.Contains(md, "[00:06] Let's begin the review\n") {
		t.Errorf("expected segment 3 in markdown, got:\n%s", md)
	}
}

func TestWhisperModelPath(t *testing.T) {
	env := map[string]string{}
	if path := WhisperModelPath(env, "en"); !strings.HasSuffix(path, "ggml-base.en.bin") {
		t.Errorf("expected base.en model, got %s", path)
	}
	if path := WhisperModelPath(env, "bn"); !strings.HasSuffix(path, "ggml-large-v3-turbo-q5_0.bin") {
		t.Errorf("expected large-v3-turbo model, got %s", path)
	}
	if path := WhisperModelPath(env, "es"); !strings.HasSuffix(path, "ggml-small.bin") {
		t.Errorf("expected small model, got %s", path)
	}
	env["LOCALSCRIBE_WHISPER_MODEL"] = "/custom/model.bin"
	if path := WhisperModelPath(env, "en"); path != "/custom/model.bin" {
		t.Errorf("expected custom model override, got %s", path)
	}
}

func TestRequireWhisperModel(t *testing.T) {
	exists := func(p string) bool {
		return p == "/mock/model.bin"
	}
	env := map[string]string{"LOCALSCRIBE_WHISPER_MODEL": "/mock/model.bin"}
	p, err := RequireWhisperModel(env, "en", exists)
	if err != nil || p != "/mock/model.bin" {
		t.Fatalf("expected existing model returned, got %v, err %v", p, err)
	}

	envMissing := map[string]string{"LOCALSCRIBE_WHISPER_MODEL": "/missing.bin"}
	_, err = RequireWhisperModel(envMissing, "en", exists)
	if err == nil || !strings.Contains(err.Error(), "LOCALSCRIBE_WHISPER_MODEL") {
		t.Fatalf("expected missing model error with hint, got %v", err)
	}
}
