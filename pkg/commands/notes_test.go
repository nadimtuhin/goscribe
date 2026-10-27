package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestNotesCommandMissingTarget(t *testing.T) {
	var out, errOut bytes.Buffer
	code := NotesCommand([]string{}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "notes requires a recording or transcript path") {
		t.Fatalf("unexpected err output: %s", errOut.String())
	}
}

func TestNotesCommandInvalidLang(t *testing.T) {
	var out, errOut bytes.Buffer
	code := NotesCommand([]string{"--lang", "invalid_lang_xyz", "some_file.mp4"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}
