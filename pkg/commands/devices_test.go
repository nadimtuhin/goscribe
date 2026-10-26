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

func TestDevicesCommandJSON(t *testing.T) {
	fake := runner.NewFakeRunner()
	fake.WhichFn = func(bin string) (string, error) {
		return "/usr/bin/" + bin, nil
	}

	source := "native/capture.swift"
	_ = os.MkdirAll("native", 0755)
	_ = os.WriteFile(source, []byte("// swift"), 0644)
	defer os.RemoveAll("native")

	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		if len(cmd) > 0 && strings.Contains(cmd[0], "swiftc") {
			bin := cmd[len(cmd)-1]
			_ = os.MkdirAll(filepath.Dir(bin), 0755)
			_ = os.WriteFile(bin, []byte("bin"), 0755)
			return "", "", 0, nil
		}
		if len(cmd) > 1 && cmd[1] == "--list" {
			return `{"displays":[{"id":1,"name":"Main","width":1920,"height":1080,"scale":2}],"windows":[{"id":100,"app":"Finder","title":"Downloads"}]}`, "", 0, nil
		}
		if len(cmd) > 0 && cmd[0] == "ffmpeg" {
			return "", "[AVFoundation indev @ 0x123] [0] Built-in Microphone\n", 0, nil
		}
		return "", "", 0, nil
	}

	var out, errOut bytes.Buffer
	code := DevicesCommand([]string{"--json"}, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, errOut: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"displays"`) || !strings.Contains(out.String(), `"Main"`) {
		t.Fatalf("unexpected json output: %s", out.String())
	}
}

func TestDevicesCommandText(t *testing.T) {
	fake := runner.NewFakeRunner()
	fake.WhichFn = func(bin string) (string, error) {
		return "/usr/bin/" + bin, nil
	}

	source := "native/capture.swift"
	_ = os.MkdirAll("native", 0755)
	_ = os.WriteFile(source, []byte("// swift"), 0644)
	defer os.RemoveAll("native")

	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		if len(cmd) > 0 && strings.Contains(cmd[0], "swiftc") {
			bin := cmd[len(cmd)-1]
			_ = os.MkdirAll(filepath.Dir(bin), 0755)
			_ = os.WriteFile(bin, []byte("bin"), 0755)
			return "", "", 0, nil
		}
		if len(cmd) > 1 && cmd[1] == "--list" {
			return `{"displays":[{"id":1,"name":"Main","width":1920,"height":1080,"scale":2}],"windows":[{"id":100,"app":"Finder","title":"Downloads"}]}`, "", 0, nil
		}
		return "", "", 0, nil
	}

	var out, errOut bytes.Buffer
	code := DevicesCommand([]string{}, fake, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, errOut: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Displays") || !strings.Contains(out.String(), "Windows") {
		t.Fatalf("unexpected text output: %s", out.String())
	}
}
