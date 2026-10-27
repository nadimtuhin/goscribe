package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/optimize"
	"github.com/nadimtuhin/goscribe/pkg/paths"
)

func TestQueueCommand(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	var out, errOut bytes.Buffer
	code := QueueCommand([]string{}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "empty") {
		t.Errorf("expected empty queue message, got %s", out.String())
	}

	qPath := paths.OptimizeQueuePath()
	optimize.EnqueueJob(qPath, "in.mp4", "out.mp4", "720p", 24, 100)

	out.Reset()
	code = QueueCommand([]string{}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "in.mp4") {
		t.Errorf("expected job in queue listing, got %s", out.String())
	}

	out.Reset()
	code = QueueCommand([]string{"--clear"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0 on clear, got %d", code)
	}
	if !strings.Contains(out.String(), "cleared") {
		t.Errorf("expected cleared message, got %s", out.String())
	}
}
