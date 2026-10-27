package commands

import (
	"bytes"
	"testing"
)

func TestLastCommandNoRecordings(t *testing.T) {
	var out, errOut bytes.Buffer
	code := LastCommand([]string{}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected code 1 on empty dir, got %d", code)
	}
}
