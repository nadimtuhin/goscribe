package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestStopCommandInvalidFlag(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	var out, errOut bytes.Buffer
	code := StopCommand([]string{"--invalid-flag"}, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("expected flag error in errOut, got %q", errOut.String())
	}
}

func TestStopCommandNotRecording(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	var out, errOut bytes.Buffer
	code := StopCommand(nil, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "not recording") {
		t.Fatalf("expected 'not recording', got %q", errOut.String())
	}
}

func TestStopCommandSuccess(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[10] = "ffmpeg"

	recFile := filepath.Join(tempDir, "rec.mp4")
	if err := os.WriteFile(recFile, []byte("fake video content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	st := &state.RecordingState{
		Pid:       10,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339),
	}
	if err := state.WriteState(statePath, st); err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	var out, errOut bytes.Buffer
	code := StopCommand(nil, statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errOut.String())
	}

	got := out.String()
	if !strings.HasPrefix(got, "stopped "+recFile) {
		t.Fatalf("expected 'stopped %s ...', got %q", recFile, got)
	}
	if !strings.Contains(got, "0.0 MB") && !strings.Contains(got, "MB") {
		t.Fatalf("expected size in output, got %q", got)
	}

	// State should be cleared
	if state.ReadActive(statePath, fake) != nil {
		t.Fatalf("expected state to be cleared after stop")
	}
}

func TestStopCommandWithNotify(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[20] = "ffmpeg"

	recFile := filepath.Join(tempDir, "rec.mp4")
	_ = os.WriteFile(recFile, []byte("video data"), 0644)

	st := &state.RecordingState{
		Pid:       20,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339),
	}
	_ = state.WriteState(statePath, st)

	var out, errOut bytes.Buffer
	code := StopCommand([]string{"--notify"}, statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errOut.String())
	}

	// Verify notification spawned
	foundNotify := false
	for _, call := range fake.SpawnDetachedCalls {
		if call.Bin == "osascript" {
			foundNotify = true
			if len(call.Args) >= 2 && strings.Contains(call.Args[1], "Goscribe") {
				// Matched
			} else {
				t.Fatalf("osascript args missing expected content: %v", call.Args)
			}
		}
	}
	if !foundNotify {
		t.Fatalf("expected osascript to be called for notification")
	}
}

func TestStopCommandMixError(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[30] = "ffmpeg"

	finalFile := filepath.Join(tempDir, "rec.mp4")
	videoPart := filepath.Join(tempDir, "rec.video.mp4")
	sysPart := filepath.Join(tempDir, "rec.system.m4a")

	_ = os.WriteFile(videoPart, []byte("video"), 0644)
	_ = os.WriteFile(sysPart, []byte("audio"), 0644)

	st := &state.RecordingState{
		Pid:         30,
		File:        finalFile,
		VideoFile:   videoPart,
		StartedAt:   time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339),
		SystemAudio: &state.AudioPart{Pid: 31, File: sysPart},
	}
	_ = state.WriteState(statePath, st)

	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		return "", "muxing error occurred", 1, errors.New("exit code 1")
	}

	var out, errOut bytes.Buffer
	code := StopCommand(nil, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1 on mix error, got %d", code)
	}

	errStr := errOut.String()
	if !strings.Contains(errStr, "failed to mix audio: muxing error occurred") {
		t.Fatalf("expected mix error in errOut, got %q", errStr)
	}
	if !strings.Contains(errStr, "kept raw parts:") {
		t.Fatalf("expected kept raw parts in errOut, got %q", errStr)
	}
}

func TestStopCommandNilWriters(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	// Error with nil writers
	code := StopCommand([]string{"--invalid-flag"}, statePath, fake, nil, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	// Success with nil writers
	fake.Alive[40] = "ffmpeg"
	recFile := filepath.Join(tempDir, "rec.mp4")
	_ = os.WriteFile(recFile, []byte("content"), 0644)
	st := &state.RecordingState{
		Pid:       40,
		File:      recFile,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = state.WriteState(statePath, st)

	code = StopCommand(nil, statePath, fake, nil, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
