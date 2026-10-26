package ffmpeg

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestParseScreens(t *testing.T) {
	output := `[AVFoundation indev @ 0x1] AVFoundation video devices:
[AVFoundation indev @ 0x1] [0] FaceTime HD Camera
[AVFoundation indev @ 0x1] [1] Capture screen 0
[AVFoundation indev @ 0x1] [2] Capture screen 1
[AVFoundation indev @ 0x1] AVFoundation audio devices:
[AVFoundation indev @ 0x1] [0] MacBook Pro Microphone`

	got := ParseScreens(output)
	want := []Device{
		{Index: 1, Name: "Capture screen 0"},
		{Index: 2, Name: "Capture screen 1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseScreens() = %+v, want %+v", got, want)
	}

	emptyGot := ParseScreens("AVFoundation audio devices:\n[0] Mic")
	if len(emptyGot) != 0 {
		t.Fatalf("ParseScreens() for no screens = %+v, want empty", emptyGot)
	}
}

func TestParseMicrophones(t *testing.T) {
	output := `[x] AVFoundation video devices:
[x] [0] FaceTime HD Camera
[x] [1] Capture screen 0
[x] AVFoundation audio devices:
[x] [0] MacBook Pro Microphone
[x] [1] AirPods Pro`

	got := ParseMicrophones(output)
	want := []Device{
		{Index: 0, Name: "MacBook Pro Microphone"},
		{Index: 1, Name: "AirPods Pro"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseMicrophones() = %+v, want %+v", got, want)
	}
}

func TestBuildArgs(t *testing.T) {
	t.Run("without mic", func(t *testing.T) {
		args := BuildArgs(1, nil, 24, "/tmp/x.mp4")

		if !slices.Contains(args, "1:none") {
			t.Errorf("BuildArgs() missing screen input 1:none, got %v", args)
		}
		fpsIdx := slices.Index(args, "-framerate")
		if fpsIdx == -1 || fpsIdx+1 >= len(args) || args[fpsIdx+1] != "24" {
			t.Errorf("BuildArgs() framerate != 24, got %v", args)
		}
		if slices.Contains(args, "-c:a") {
			t.Errorf("BuildArgs() without mic should not include audio codec, got %v", args)
		}
		if args[len(args)-1] != "/tmp/x.mp4" {
			t.Errorf("BuildArgs() last arg = %q, want /tmp/x.mp4", args[len(args)-1])
		}
	})

	t.Run("with mic", func(t *testing.T) {
		mic := 0
		args := BuildArgs(1, &mic, 30, "/tmp/x.mp4")

		if !slices.Contains(args, "1:0") {
			t.Errorf("BuildArgs() missing screen:mic 1:0, got %v", args)
		}
		fpsIdx := slices.Index(args, "-framerate")
		if fpsIdx == -1 || fpsIdx+1 >= len(args) || args[fpsIdx+1] != "30" {
			t.Errorf("BuildArgs() framerate != 30, got %v", args)
		}
		codecIdx := slices.Index(args, "-c:a")
		if codecIdx == -1 || codecIdx+1 >= len(args) || args[codecIdx+1] != "aac" {
			t.Errorf("BuildArgs() -c:a != aac, got %v", args)
		}
		if args[len(args)-1] != "/tmp/x.mp4" {
			t.Errorf("BuildArgs() last arg = %q, want /tmp/x.mp4", args[len(args)-1])
		}
	})
}

func TestBuildMicArgs(t *testing.T) {
	args := BuildMicArgs(2, "/r/a.mic.m4a")

	if !slices.Contains(args, "none:2") {
		t.Errorf("BuildMicArgs() missing none:2, got %v", args)
	}
	if slices.Contains(args, "libx264") {
		t.Errorf("BuildMicArgs() should not contain libx264, got %v", args)
	}
	codecIdx := slices.Index(args, "-c:a")
	if codecIdx == -1 || codecIdx+1 >= len(args) || args[codecIdx+1] != "aac" {
		t.Errorf("BuildMicArgs() -c:a != aac, got %v", args)
	}
	if args[len(args)-1] != "/r/a.mic.m4a" {
		t.Errorf("BuildMicArgs() last arg = %q, want /r/a.mic.m4a", args[len(args)-1])
	}
}

func TestBuildMuxArgs(t *testing.T) {
	t.Run("with videoHasMic true", func(t *testing.T) {
		args := BuildMuxArgs("v.mp4", true, "s.m4a", "o.mp4")
		joined := strings.Join(args, " ")

		if !strings.Contains(joined, "-i v.mp4 -i s.m4a") {
			t.Errorf("BuildMuxArgs() missing input files, got %s", joined)
		}
		if !strings.Contains(joined, "[0:a][1:a]amix=inputs=2:duration=first:normalize=0[mixed]") {
			t.Errorf("BuildMuxArgs() missing amix filter, got %s", joined)
		}
		if !slices.Contains(args, "[mixed]") || !slices.Contains(args, "copy") {
			t.Errorf("BuildMuxArgs() missing map or copy codec, got %v", args)
		}
		if slices.Contains(args, "-shortest") {
			t.Errorf("BuildMuxArgs() should not contain -shortest")
		}
		if args[len(args)-1] != "o.mp4" {
			t.Errorf("BuildMuxArgs() last arg = %q, want o.mp4", args[len(args)-1])
		}
	})

	t.Run("with videoHasMic false", func(t *testing.T) {
		args := BuildMuxArgs("v.mp4", false, "s.m4a", "o.mp4")
		joined := strings.Join(args, " ")

		if !strings.Contains(joined, "-i v.mp4 -i s.m4a") {
			t.Errorf("BuildMuxArgs() missing input files, got %s", joined)
		}
		if !strings.Contains(joined, "-map 0:v -map 1:a -c copy") {
			t.Errorf("BuildMuxArgs() missing expected stream mapping, got %s", joined)
		}
		if slices.Contains(args, "-shortest") {
			t.Errorf("BuildMuxArgs() should not contain -shortest")
		}
		if args[len(args)-1] != "o.mp4" {
			t.Errorf("BuildMuxArgs() last arg = %q, want o.mp4", args[len(args)-1])
		}
	})
}

func TestBuildSckMuxArgs(t *testing.T) {
	t.Run("1 audio input", func(t *testing.T) {
		args := BuildSckMuxArgs("v.mp4", []string{"s.m4a"}, "o.mp4")
		joined := strings.Join(args, " ")

		if !strings.Contains(joined, "-i v.mp4 -i s.m4a") {
			t.Errorf("BuildSckMuxArgs() missing inputs, got %s", joined)
		}
		if !strings.Contains(joined, "-map 0:v -map 1:a -c copy") {
			t.Errorf("BuildSckMuxArgs() missing stream map/copy, got %s", joined)
		}
		if args[len(args)-1] != "o.mp4" {
			t.Errorf("BuildSckMuxArgs() last arg = %q, want o.mp4", args[len(args)-1])
		}
	})

	t.Run("2 audio inputs", func(t *testing.T) {
		args := BuildSckMuxArgs("v.mp4", []string{"s.m4a", "m.m4a"}, "o.mp4")
		joined := strings.Join(args, " ")

		if !strings.Contains(joined, "-i v.mp4 -i s.m4a -i m.m4a") {
			t.Errorf("BuildSckMuxArgs() missing inputs, got %s", joined)
		}
		if !strings.Contains(joined, "[1:a][2:a]amix=inputs=2:duration=longest:normalize=0[mixed]") {
			t.Errorf("BuildSckMuxArgs() missing filter complex, got %s", joined)
		}
		if !slices.Contains(args, "[mixed]") || !slices.Contains(args, "copy") {
			t.Errorf("BuildSckMuxArgs() missing map or copy codec, got %v", args)
		}
		if slices.Contains(args, "-shortest") {
			t.Errorf("BuildSckMuxArgs() should not contain -shortest")
		}
		if args[len(args)-1] != "o.mp4" {
			t.Errorf("BuildSckMuxArgs() last arg = %q, want o.mp4", args[len(args)-1])
		}
	})
}

func TestBuildAudioMixArgs(t *testing.T) {
	args := BuildAudioMixArgs([]string{"/r/s.m4a", "/r/m.m4a"}, "/r/out.m4a")
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-i /r/s.m4a -i /r/m.m4a") {
		t.Errorf("BuildAudioMixArgs() missing inputs, got %s", joined)
	}
	if !strings.Contains(joined, "[0:a][1:a]amix=inputs=2:duration=longest:normalize=0[mixed]") {
		t.Errorf("BuildAudioMixArgs() missing filter complex, got %s", joined)
	}
	if !slices.Contains(args, "[mixed]") || !slices.Contains(args, "aac") || !slices.Contains(args, "192k") {
		t.Errorf("BuildAudioMixArgs() missing audio codec args, got %v", args)
	}
	if args[len(args)-1] != "/r/out.m4a" {
		t.Errorf("BuildAudioMixArgs() last arg = %q, want /r/out.m4a", args[len(args)-1])
	}
}

func TestWithToolPaths(t *testing.T) {
	if got, want := WithToolPaths("/usr/bin:/bin"), "/usr/bin:/bin:/opt/homebrew/bin:/usr/local/bin"; got != want {
		t.Errorf("WithToolPaths(/usr/bin:/bin) = %q, want %q", got, want)
	}
	if got, want := WithToolPaths("/opt/homebrew/bin:/usr/bin"), "/opt/homebrew/bin:/usr/bin:/usr/local/bin"; got != want {
		t.Errorf("WithToolPaths(/opt/homebrew/bin:/usr/bin) = %q, want %q", got, want)
	}
	if got, want := WithToolPaths(""), "/opt/homebrew/bin:/usr/local/bin"; got != want {
		t.Errorf("WithToolPaths(\"\") = %q, want %q", got, want)
	}
}

func TestEnsureToolPath(t *testing.T) {
	orig := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", orig) }()

	_ = os.Setenv("PATH", "/usr/bin:/bin")
	EnsureToolPath()

	want := "/usr/bin:/bin:/opt/homebrew/bin:/usr/local/bin"
	if got := os.Getenv("PATH"); got != want {
		t.Errorf("EnsureToolPath() set PATH = %q, want %q", got, want)
	}
}
