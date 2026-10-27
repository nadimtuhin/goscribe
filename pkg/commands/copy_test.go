package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyCommandNoRecordings(t *testing.T) {
	var out, errOut bytes.Buffer
	code := CopyCommand([]string{}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestCopyCommandPositional(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "rec.mp4")
	notesFile := filepath.Join(tmp, "rec.notes.md")
	_ = os.WriteFile(notesFile, []byte("Notes content"), 0644)

	var out, errOut bytes.Buffer
	code := CopyCommand([]string{target}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, err: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "copied to clipboard") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
