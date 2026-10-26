package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestStartCommandInvalidFlag(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	var out, errOut bytes.Buffer
	code := StartCommand([]string{"--invalid-flag"}, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("expected flag error in errOut, got %q", errOut.String())
	}
}

func TestStartCommandAlreadyRecording(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[123] = "ffmpeg"

	st := &state.RecordingState{
		Pid:  123,
		File: filepath.Join(tempDir, "rec.mp4"),
	}
	if err := state.WriteState(statePath, st); err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	var out, errOut bytes.Buffer
	code := StartCommand([]string{"--out", filepath.Join(tempDir, "new.mp4")}, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "already recording") {
		t.Fatalf("expected already recording error, got %q", errOut.String())
	}
}

func TestStartCommandProcessDiesOnStartup(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()
	fake.SpawnDetachedFn = func(bin string, args []string, logFile string) (int, error) {
		return 999, nil
	}

	var out, errOut bytes.Buffer
	code := StartCommand([]string{"--out", filepath.Join(tempDir, "rec.mp4"), "--no-system-audio"}, statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "exited on startup") {
		t.Fatalf("expected exited on startup error, got %q", errOut.String())
	}
}

func TestStartCommandSuccess(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	outFile := filepath.Join(tempDir, "output.mp4")
	args := []string{
		"--out", outFile,
		"--name", "test-rec",
		"--fps", "60",
		"--screen", "1",
		"--mic", "2",
		"--no-system-audio",
	}

	var out, errOut bytes.Buffer
	code := StartCommand(args, statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errOut.String())
	}

	expectedOut := "recording to " + outFile + "\n"
	if got := out.String(); got != expectedOut {
		t.Fatalf("expected %q, got %q", expectedOut, got)
	}

	active := state.ReadActive(statePath, fake)
	if active == nil {
		t.Fatalf("expected active recording state to be written")
	}
	if active.File != outFile {
		t.Fatalf("expected file %q, got %q", outFile, active.File)
	}
}

func TestStartCommandNilWriters(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	// Should not panic with nil writers on error
	code := StartCommand([]string{"--invalid-flag"}, statePath, fake, nil, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	// Should not panic with nil writers on success
	outFile := filepath.Join(tempDir, "out2.mp4")
	code = StartCommand([]string{"--out", outFile, "--no-system-audio"}, statePath, fake, nil, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
