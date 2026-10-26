package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/recorder"
)

func TestListCommandEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	var buf bytes.Buffer
	code := ListCommand(tempDir, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := buf.String(); got != "no recordings yet\n" {
		t.Fatalf("expected 'no recordings yet\\n', got %q", got)
	}
}

func TestListCommandNonExistentDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")

	var buf bytes.Buffer
	code := ListCommand(nonExistent, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := buf.String(); got != "no recordings yet\n" {
		t.Fatalf("expected 'no recordings yet\\n', got %q", got)
	}
}

func TestListCommandSortedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files:
	// file1.mp4: older, 1024*1024 bytes (1.0 MB)
	// file2.m4a: newer, 2*1024*1024 bytes (2.0 MB)
	// file3.part.mp4: partial file (should be ignored)
	// notes.txt: non-media file (should be ignored)
	file1Path := filepath.Join(dir, "file1.mp4")
	file2Path := filepath.Join(dir, "file2.m4a")
	file3Path := filepath.Join(dir, "file3.part.mp4")
	file4Path := filepath.Join(dir, "notes.txt")

	if err := os.WriteFile(file1Path, make([]byte, 1024*1024), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2Path, make([]byte, 2*1024*1024), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}
	if err := os.WriteFile(file3Path, []byte("part"), 0644); err != nil {
		t.Fatalf("failed to create file3: %v", err)
	}
	if err := os.WriteFile(file4Path, []byte("text"), 0644); err != nil {
		t.Fatalf("failed to create file4: %v", err)
	}

	olderTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	newerTime := time.Date(2025, 1, 2, 15, 30, 0, 0, time.UTC)

	if err := os.Chtimes(file1Path, olderTime, olderTime); err != nil {
		t.Fatalf("failed to chtimes file1: %v", err)
	}
	if err := os.Chtimes(file2Path, newerTime, newerTime); err != nil {
		t.Fatalf("failed to chtimes file2: %v", err)
	}

	var buf bytes.Buffer
	code := ListCommand(dir, &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), lines)
	}

	expectedLine1 := fmt.Sprintf("%s  %9s  %s", newerTime.Local().Format("01/02/2006, 3:04:05 PM"), recorder.FormatSize(2*1024*1024), "file2.m4a")
	expectedLine2 := fmt.Sprintf("%s  %9s  %s", olderTime.Local().Format("01/02/2006, 3:04:05 PM"), recorder.FormatSize(1024*1024), "file1.mp4")

	if lines[0] != expectedLine1 {
		t.Fatalf("line 1 mismatch: expected %q, got %q", expectedLine1, lines[0])
	}
	if lines[1] != expectedLine2 {
		t.Fatalf("line 2 mismatch: expected %q, got %q", expectedLine2, lines[1])
	}
}

func TestListCommandNilWriter(t *testing.T) {
	code := ListCommand(t.TempDir(), nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
