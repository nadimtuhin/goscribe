package recorder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

// If a later process fails to spawn, the ffmpeg already started (detached, in
// its own process group) must be killed, not orphaned.
func TestStartKillsEarlierProcessWhenLaterSpawnFails(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	var ffmpegPid int
	fake.SpawnDetachedFn = func(bin string, args []string, log string) (int, error) {
		if bin == "ffmpeg" {
			ffmpegPid = 4242
			fake.Alive[ffmpegPid] = "ffmpeg"
			return ffmpegPid, nil
		}
		return 0, errors.New("spawn sysaudio failed")
	}
	// Make EnsureHelper's up-to-date check pass without swiftc.
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(tmp)
	_ = os.MkdirAll("native", 0755)
	_ = os.WriteFile("native/sysaudio.swift", []byte("x"), 0644)
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes("native/sysaudio.swift", old, old)
	_ = os.WriteFile(filepath.Join(tmp, "sysaudio"), []byte("bin"), 0755)

	_, err := Start(StartOptions{Out: filepath.Join(tmp, "rec.mp4")}, statePath, fake)
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, alive := fake.Alive[ffmpegPid]; alive {
		t.Fatalf("ffmpeg pid %d left running after failed start", ffmpegPid)
	}
}

// A mux that fails without stderr (e.g. ffmpeg missing or killed) must still
// be reported as a failure instead of "stopped ... (0 B)".
func TestStopMuxFailureWithoutStderrIsReported(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	fake.Alive[10] = "ffmpeg"
	videoPart := filepath.Join(tmp, "rec.video.mp4")
	sysPart := filepath.Join(tmp, "rec.system.m4a")
	_ = os.WriteFile(videoPart, []byte("v"), 0644)
	_ = os.WriteFile(sysPart, []byte("a"), 0644)
	_ = state.WriteState(statePath, &state.RecordingState{
		Pid: 10, File: filepath.Join(tmp, "rec.mp4"), VideoFile: videoPart,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		SystemAudio: &state.AudioPart{Pid: 20, File: sysPart},
	})
	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		return "", "", -1, errors.New("exec: \"ffmpeg\": executable file not found")
	}
	res, err := Stop(statePath, fake)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if res.MixError == "" {
		t.Fatalf("mux failed but MixError is empty; caller would report success")
	}
}
