package recorder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestStartRefusesIfAlreadyRecording(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[100] = "ffmpeg"

	st := &state.RecordingState{
		Pid:  100,
		File: filepath.Join(tmp, "rec.mp4"),
	}
	if err := state.WriteState(statePath, st); err != nil {
		t.Fatalf("write state error: %v", err)
	}

	_, err := Start(StartOptions{}, statePath, fake)
	if err == nil || !stringContains(err.Error(), "already recording") {
		t.Fatalf("expected already recording error, got %v", err)
	}
}

func TestStartRefusesIfFileExists(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()

	existingFile := filepath.Join(tmp, "test.mp4")
	if err := os.WriteFile(existingFile, []byte("data"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	_, err := Start(StartOptions{Out: existingFile}, statePath, fake)
	if err == nil || !stringContains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
}

func TestStartStartupCheckFailsIfProcessDies(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	fake.SpawnDetachedFn = func(bin string, args []string, log string) (int, error) {
		return 999, nil
	}

	_, err := Start(StartOptions{Out: filepath.Join(tmp, "rec.mp4")}, statePath, fake)
	if err == nil || !stringContains(err.Error(), "exited on startup") {
		t.Fatalf("expected exited on startup error, got %v", err)
	}
	if state.ReadActive(statePath, fake) != nil {
		t.Fatalf("expected state to be cleared after startup failure")
	}
}

func TestStopMuxSuccessRemovesParts(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[10] = "ffmpeg"

	finalFile := filepath.Join(tmp, "rec.mp4")
	videoPart := filepath.Join(tmp, "rec.video.mp4")
	sysPart := filepath.Join(tmp, "rec.system.m4a")

	_ = os.WriteFile(videoPart, []byte("video content"), 0644)
	_ = os.WriteFile(sysPart, []byte("audio content"), 0644)

	st := &state.RecordingState{
		Pid:         10,
		File:        finalFile,
		VideoFile:   videoPart,
		StartedAt:   time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339),
		SystemAudio: &state.AudioPart{Pid: 20, File: sysPart},
	}
	_ = state.WriteState(statePath, st)

	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		_ = os.WriteFile(finalFile, []byte("muxed content"), 0644)
		return "", "", 0, nil
	}

	res, err := Stop(statePath, fake)
	if err != nil {
		t.Fatalf("stop error: %v", err)
	}
	if res.MixError != "" {
		t.Fatalf("unexpected mix error: %s", res.MixError)
	}
	if _, err := os.Stat(videoPart); err == nil {
		t.Errorf("expected video part to be deleted")
	}
	if _, err := os.Stat(sysPart); err == nil {
		t.Errorf("expected sys part to be deleted")
	}
	if _, err := os.Stat(finalFile); err != nil {
		t.Errorf("expected final file to exist")
	}
}

func TestStopMuxFailureKeepsParts(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[10] = "ffmpeg"

	finalFile := filepath.Join(tmp, "rec.mp4")
	videoPart := filepath.Join(tmp, "rec.video.mp4")
	sysPart := filepath.Join(tmp, "rec.system.m4a")

	_ = os.WriteFile(videoPart, []byte("video content"), 0644)
	_ = os.WriteFile(sysPart, []byte("audio content"), 0644)

	st := &state.RecordingState{
		Pid:         10,
		File:        finalFile,
		VideoFile:   videoPart,
		StartedAt:   time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339),
		SystemAudio: &state.AudioPart{Pid: 20, File: sysPart},
	}
	_ = state.WriteState(statePath, st)

	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		return "", "mux error failed", 1, nil
	}

	res, err := Stop(statePath, fake)
	if err != nil {
		t.Fatalf("stop error: %v", err)
	}
	if res.MixError != "mux error failed" {
		t.Fatalf("expected mux error, got %q", res.MixError)
	}
	if len(res.KeptParts) != 2 {
		t.Fatalf("expected 2 kept parts, got %v", res.KeptParts)
	}
	if _, err := os.Stat(videoPart); err != nil {
		t.Errorf("expected video part to be kept")
	}
	if _, err := os.Stat(sysPart); err != nil {
		t.Errorf("expected sys part to be kept")
	}
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
