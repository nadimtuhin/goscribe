package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestStatusCommandInactive(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	var buf bytes.Buffer
	code := StatusCommand(statePath, fake, false, nil, &buf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if got := buf.String(); got != "not recording\n" {
		t.Fatalf("expected 'not recording\\n', got %q", got)
	}

	buf.Reset()
	code = StatusCommand(statePath, fake, true, nil, &buf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if got := buf.String(); got != "{\"recording\":false}\n" {
		t.Fatalf("expected '{\"recording\":false}\\n', got %q", got)
	}
}

func TestStatusCommandStaleState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	// Write state for a non-running PID
	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       9999,
		Exe:       "ffmpeg",
		File:      filepath.Join(tempDir, "test.mp4"),
		StartedAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	var buf bytes.Buffer
	code := StatusCommand(statePath, fake, false, nil, &buf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if got := buf.String(); got != "not recording\n" {
		t.Fatalf("expected 'not recording\\n', got %q", got)
	}
}

func TestStatusCommandActive(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	pid := 1234
	fake.Alive[pid] = "ffmpeg"

	started := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	recFile := filepath.Join(tempDir, "rec.mp4")
	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: started.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	fixedNow := func() time.Time {
		return started.Add(75 * time.Second) // 1m15s
	}

	// Plain text output
	var buf bytes.Buffer
	code := StatusCommand(statePath, fake, false, fixedNow, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	expectedText := "recording " + recFile + " (1m15s)\n"
	if got := buf.String(); got != expectedText {
		t.Fatalf("expected %q, got %q", expectedText, got)
	}

	// JSON output
	buf.Reset()
	code = StatusCommand(statePath, fake, true, fixedNow, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	expectedJSON := "{\"recording\":true,\"file\":\"" + recFile + "\",\"startedAt\":\"" + started.Format(time.RFC3339) + "\",\"elapsedMs\":75000}\n"
	if got := buf.String(); got != expectedJSON {
		t.Fatalf("expected %q, got %q", expectedJSON, got)
	}
}

func TestStatusCommandActiveInvalidStartedAt(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	pid := 1234
	fake.Alive[pid] = "ffmpeg"

	recFile := filepath.Join(tempDir, "rec.mp4")
	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: "not-a-valid-timestamp",
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	now := time.Now()
	fixedNow := func() time.Time { return now }

	var buf bytes.Buffer
	code := StatusCommand(statePath, fake, true, fixedNow, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(buf.String(), `"elapsedMs":0`) {
		t.Fatalf("expected elapsedMs:0 on fallback, got %q", buf.String())
	}
}

func TestStatusCommandActiveFutureStartedAt(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	pid := 1234
	fake.Alive[pid] = "ffmpeg"

	recFile := filepath.Join(tempDir, "rec.mp4")
	now := time.Now()
	err := state.WriteState(statePath, &state.RecordingState{
		Pid:       pid,
		Exe:       "ffmpeg",
		File:      recFile,
		StartedAt: now.Add(10 * time.Minute).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	var buf bytes.Buffer
	code := StatusCommand(statePath, fake, true, func() time.Time { return now }, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(buf.String(), `"elapsedMs":0`) {
		t.Fatalf("expected elapsedMs:0 when started in future, got %q", buf.String())
	}
}

func TestStatusCommandNilWriterAndNow(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	fake := runner.NewFakeRunner()

	// Should not panic with nil out
	code := StatusCommand(statePath, fake, false, nil, nil)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}
