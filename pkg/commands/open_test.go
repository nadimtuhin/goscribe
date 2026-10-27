package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestOpenCommandNoRecordings(t *testing.T) {
	var out, errOut bytes.Buffer
	code := OpenCommand([]string{"/nonexistent/file.mp4"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "no recordings found") {
		t.Fatalf("unexpected err output: %s", errOut.String())
	}
}
