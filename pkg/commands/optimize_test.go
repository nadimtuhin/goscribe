package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func TestOptimizeCommand(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	target := filepath.Join(tmp, "video.mp4")
	_ = os.WriteFile(target, []byte("fake video content"), 0644)

	fake := runner.NewFakeRunner()
	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		return "", "", 0, nil
	}

	var out, errOut bytes.Buffer
	code := OptimizeCommand([]string{target, "--wait", "--resolution", "720p"}, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, err: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "enqueued job") {
		t.Errorf("expected enqueued message, got %s", out.String())
	}
}
