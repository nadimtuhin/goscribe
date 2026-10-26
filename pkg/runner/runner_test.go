package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRealRunner_Which(t *testing.T) {
	r := &RealRunner{}

	t.Run("existing binary", func(t *testing.T) {
		path, err := r.Which("sh")
		if err != nil {
			t.Fatalf("expected Which(\"sh\") to succeed, got %v", err)
		}
		if path == "" {
			t.Fatal("expected non-empty path for \"sh\"")
		}
	})

	t.Run("missing binary", func(t *testing.T) {
		_, err := r.Which("definitely_nonexistent_binary_xyz_12345")
		if err == nil {
			t.Fatal("expected error for nonexistent binary, got nil")
		}
	})
}

func TestRealRunner_Run(t *testing.T) {
	r := &RealRunner{}

	t.Run("success with stdout", func(t *testing.T) {
		stdout, stderr, code, err := r.Run(context.Background(), []string{"echo", "hello-goscribe"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		if strings.TrimSpace(stdout) != "hello-goscribe" {
			t.Fatalf("unexpected stdout: %q", stdout)
		}
		if stderr != "" {
			t.Fatalf("unexpected stderr: %q", stderr)
		}
	})

	t.Run("non-zero exit code with stderr", func(t *testing.T) {
		stdout, stderr, code, err := r.Run(context.Background(), []string{"sh", "-c", "echo error-msg >&2; exit 42"})
		if err == nil {
			t.Fatal("expected error for non-zero exit code, got nil")
		}
		if code != 42 {
			t.Fatalf("expected exit code 42, got %d", code)
		}
		if strings.TrimSpace(stderr) != "error-msg" {
			t.Fatalf("unexpected stderr: %q", stderr)
		}
		if stdout != "" {
			t.Fatalf("unexpected stdout: %q", stdout)
		}
	})

	t.Run("killed by signal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, _, code, err := r.Run(ctx, []string{"sleep", "5"})
		if err == nil {
			t.Fatal("expected error for canceled context, got nil")
		}
		if code != 1 {
			t.Fatalf("expected exit code 1 for signal-killed process, got %d", code)
		}
	})

	t.Run("empty command error", func(t *testing.T) {
		_, _, code, err := r.Run(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for empty command")
		}
		if code != -1 {
			t.Fatalf("expected exit code -1, got %d", code)
		}
	})
}

func TestRealRunner_SpawnDetached(t *testing.T) {
	r := &RealRunner{}
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "detached.log")

	pid, err := r.SpawnDetached("sleep", []string{"10"}, logFile)
	if err != nil {
		t.Fatalf("SpawnDetached failed: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	if !r.IsAlive(pid) {
		t.Fatalf("expected pid %d to be alive", pid)
	}
	if !r.IsRunning(pid, "sleep") {
		t.Fatalf("expected pid %d to be running sleep", pid)
	}
	if r.IsRunning(pid, "other_proc") {
		t.Fatalf("expected pid %d to NOT be running other_proc", pid)
	}

	if err := r.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Poll until not running
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !r.IsRunning(pid, "sleep") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("after kill: IsAlive=%v, IsRunning=%v", r.IsAlive(pid), r.IsRunning(pid, "sleep"))

	// Verify log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("log file was not created: %v", err)
	}
}

func TestRealRunner_Sleep(t *testing.T) {
	r := &RealRunner{}
	start := time.Now()
	r.Sleep(20 * time.Millisecond)
	elapsed := time.Since(start)
	if elapsed < 15*time.Millisecond {
		t.Fatalf("Sleep finished too quickly: %v", elapsed)
	}
}

func TestRealRunner_EdgeCases(t *testing.T) {
	r := &RealRunner{}

	if r.IsAlive(0) {
		t.Fatal("pid 0 must not be considered alive")
	}
	if r.IsAlive(-1) {
		t.Fatal("pid -1 must not be considered alive")
	}
	if r.IsRunning(0, "sleep") {
		t.Fatal("pid 0 must not be running")
	}
	if r.IsRunning(-1, "sleep") {
		t.Fatal("pid -1 must not be running")
	}
	if err := r.Kill(0, syscall.SIGKILL); err == nil {
		t.Fatal("kill on pid 0 must return error")
	}
	if err := r.Kill(-1, syscall.SIGKILL); err == nil {
		t.Fatal("kill on pid -1 must return error")
	}
}

func TestFakeRunner(t *testing.T) {
	fake := NewFakeRunner()

	// Which
	w, err := fake.Which("ffmpeg")
	if err != nil || w != "/fake/bin/ffmpeg" {
		t.Fatalf("unexpected Which: %q, %v", w, err)
	}

	// Run
	stdout, stderr, code, err := fake.Run(context.Background(), []string{"ffmpeg", "-version"})
	if err != nil || code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("unexpected Run: %d, %v", code, err)
	}

	// SpawnDetached
	pid, err := fake.SpawnDetached("ffmpeg", []string{"-i", "in.mp4"}, "/tmp/out.log")
	if err != nil || pid != 1000 {
		t.Fatalf("unexpected SpawnDetached: %d, %v", pid, err)
	}
	if !fake.IsAlive(pid) {
		t.Fatal("expected fake pid to be alive")
	}
	if !fake.IsRunning(pid, "ffmpeg") {
		t.Fatal("expected fake pid to be running ffmpeg")
	}
	if fake.IsRunning(pid, "sh") {
		t.Fatal("expected fake pid to not be running sh")
	}

	// Kill
	if err := fake.Kill(pid, syscall.SIGTERM); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	if fake.IsAlive(pid) {
		t.Fatal("expected fake pid to be dead after kill")
	}

	// Sleep
	fake.Sleep(5 * time.Second) // should not block
	if len(fake.SleepCalls) != 1 || fake.SleepCalls[0] != 5*time.Second {
		t.Fatalf("unexpected SleepCalls: %v", fake.SleepCalls)
	}

	// Check recorded calls
	if len(fake.WhichCalls) != 1 || fake.WhichCalls[0] != "ffmpeg" {
		t.Fatalf("unexpected WhichCalls: %v", fake.WhichCalls)
	}
	if len(fake.RunCalls) != 1 || len(fake.RunCalls[0]) != 2 {
		t.Fatalf("unexpected RunCalls: %v", fake.RunCalls)
	}
	if len(fake.SpawnDetachedCalls) != 1 || fake.SpawnDetachedCalls[0].Bin != "ffmpeg" {
		t.Fatalf("unexpected SpawnDetachedCalls: %v", fake.SpawnDetachedCalls)
	}
	if len(fake.KillCalls) != 1 || fake.KillCalls[0].Sig != syscall.SIGTERM {
		t.Fatalf("unexpected KillCalls: %v", fake.KillCalls)
	}
}
