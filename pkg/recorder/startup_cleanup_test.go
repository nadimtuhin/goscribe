package recorder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

// When one process dies during the startup check, Start must kill every
// sibling it already spawned (video and audio-only modes).
func assertStartupFailureKillsSiblings(t *testing.T, dead string) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	fake := runner.NewFakeRunner()
	pids := map[string]int{}
	next := 5000
	fake.SpawnDetachedFn = func(bin string, args []string, log string) (int, error) {
		next++
		name := filepath.Base(bin)
		pids[name] = next
		fake.Alive[next] = name
		return next, nil
	}
	fake.SleepFn = func(time.Duration) { delete(fake.Alive, pids[dead]) }
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.Chdir(tmp)
	_ = os.MkdirAll("native", 0755)
	_ = os.WriteFile("native/sysaudio.swift", []byte("x"), 0644)
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes("native/sysaudio.swift", old, old)
	_ = os.WriteFile(filepath.Join(tmp, "sysaudio"), []byte("bin"), 0755)
	for _, ao := range []bool{false, true} {
		_, err := Start(StartOptions{Out: filepath.Join(tmp, "rec"+map[bool]string{true: "a", false: "v"}[ao]+".mp4"), AudioOnly: ao}, statePath, fake)
		if err == nil {
			t.Fatalf("audioOnly=%v: expected startup error", ao)
		}
		if len(fake.Alive) != 0 {
			t.Errorf("orphan left: %v", fake.Alive)
		}
	}
}
func TestStartupCheckKillsFfmpegWhenSysaudioDies(t *testing.T) {
	assertStartupFailureKillsSiblings(t, "sysaudio")
}

func TestStartupCheckKillsSysaudioWhenFfmpegDies(t *testing.T) {
	assertStartupFailureKillsSiblings(t, "ffmpeg")
}
