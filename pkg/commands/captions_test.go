package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestCaptionsCommandInactive(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	var out, errOut bytes.Buffer
	code := CaptionsCommand(statePath, fake, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if got := errOut.String(); got != "not recording\n" {
		t.Fatalf("expected 'not recording\\n', got %q", got)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", out.String())
	}
}

func TestCaptionsCommandTailsLiveFileUntilStop(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	recFile := filepath.Join(tempDir, "rec.mp4")
	liveFile := filepath.Join(tempDir, "rec.live.txt")

	fake := runner.NewFakeRunner()
	pid := 1234
	fake.Alive[pid] = "ffmpeg"

	if err := os.WriteFile(liveFile, []byte("first line\n"), 0644); err != nil {
		t.Fatalf("failed to write live file: %v", err)
	}

	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	tick := 0
	fake.SleepFn = func(d time.Duration) {
		tick++
		if tick == 1 {
			f, err := os.OpenFile(liveFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				t.Fatalf("failed to append to live file: %v", err)
			}
			defer f.Close()
			_, _ = f.WriteString("second line\n")
		} else if tick == 2 {
			f, err := os.OpenFile(liveFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				t.Fatalf("failed to append to live file: %v", err)
			}
			defer f.Close()
			_, _ = f.WriteString("last line\n")
		} else if tick > 2 {
			delete(fake.Alive, pid)
		}
	}

	var out, errOut bytes.Buffer
	code := CaptionsCommand(statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	expected := "first line\nsecond line\nlast line\n"
	if got := out.String(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestCaptionsCommandWaitsForLiveFile(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	recFile := filepath.Join(tempDir, "rec.mp4")
	liveFile := filepath.Join(tempDir, "rec.live.txt")

	fake := runner.NewFakeRunner()
	pid := 5678
	fake.Alive[pid] = "ffmpeg"

	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	tick := 0
	fake.SleepFn = func(d time.Duration) {
		tick++
		if tick == 1 {
			_ = os.WriteFile(liveFile, []byte("hello from live\n"), 0644)
		} else if tick > 1 {
			delete(fake.Alive, pid)
		}
	}

	var out, errOut bytes.Buffer
	code := CaptionsCommand(statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	expected := "hello from live\n"
	if got := out.String(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestCaptionsCommandInterrupted(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	recFile := filepath.Join(tempDir, "rec.mp4")
	liveFile := filepath.Join(tempDir, "rec.live.txt")

	fake := runner.NewFakeRunner()
	pid := 9999
	fake.Alive[pid] = "ffmpeg"

	_ = os.WriteFile(liveFile, []byte("before interrupt\n"), 0644)

	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	fake.SleepFn = func(d time.Duration) {
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
		// Small sleep so signal is delivered before next select
		time.Sleep(10 * time.Millisecond)
	}

	var out, errOut bytes.Buffer
	code := CaptionsCommand(statePath, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	expected := "before interrupt\n"
	if got := out.String(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
