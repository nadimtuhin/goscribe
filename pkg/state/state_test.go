package state

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func sampleState(pid int) *RecordingState {
	return &RecordingState{
		Pid:             pid,
		Exe:             "ffmpeg",
		Engine:          "ffmpeg",
		File:            "/tmp/rec.mp4",
		VideoFile:       "/tmp/rec.mp4",
		VideoHasMic:     true,
		AudioOnly:       false,
		SystemAudio:     &AudioPart{Pid: 200, File: "/tmp/sys.m4a"},
		SystemAudioFile: "/tmp/sys.m4a",
		Mic:             &AudioPart{Pid: 201, File: "/tmp/mic.m4a"},
		StartedAt:       "2026-04-18T10:00:00Z",
		PluginOptions:   map[string]any{"transcribe": true},
		Sidecars: []Sidecar{
			{Pid: 300, Exe: "livesys", Drain: false},
			{Pid: 301, Exe: "drainer", Drain: true},
		},
		Processes: []int{pid, 200, 201},
	}
}

func TestWriteAndReadState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	initial := sampleState(100)

	if err := WriteState(path, initial); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	read, err := ReadState(path)
	if err != nil {
		t.Fatalf("ReadState() error = %v", err)
	}

	if !reflect.DeepEqual(initial, read) {
		t.Errorf("ReadState() = %+v, want %+v", read, initial)
	}
}

func TestWriteStateAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "sub", "state.json")
	st := sampleState(101)

	if err := WriteState(path, st); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("parent directory not created: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file missing: %v", err)
	}

	tempPath := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s still exists", tempPath)
	}
}

func TestReadStateMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing.json")

	st, err := ReadState(missingPath)
	if err == nil || st != nil {
		t.Errorf("ReadState(missing) expected error, got state=%v, err=%v", st, err)
	}

	corruptPath := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corruptPath, []byte("not valid json {"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	st, err = ReadState(corruptPath)
	if err == nil || st != nil {
		t.Errorf("ReadState(corrupt) expected error, got state=%v, err=%v", st, err)
	}
}

func TestClearState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := WriteState(path, sampleState(102)); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}
	if err := ClearState(path); err != nil {
		t.Fatalf("ClearState() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists after ClearState()")
	}

	// Clearing already removed or non-existent file should ignore IsNotExist
	if err := ClearState(path); err != nil {
		t.Errorf("ClearState() on non-existent file returned error: %v", err)
	}
}

func TestReadActiveWithLiveAndDeadProcess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	fake := runner.NewFakeRunner()

	t.Run("returns state when process is running default ffmpeg", func(t *testing.T) {
		st := sampleState(1001)
		st.Exe = ""
		fake.Alive[1001] = "ffmpeg"
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		active := ReadActive(path, fake)
		if active == nil || active.Pid != 1001 {
			t.Fatalf("ReadActive() = %v, want Pid 1001", active)
		}
	})

	t.Run("returns state when process is running custom exe", func(t *testing.T) {
		st := sampleState(1002)
		st.Exe = "custom-capture"
		fake.Alive[1002] = "custom-capture"
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		active := ReadActive(path, fake)
		if active == nil || active.Pid != 1002 {
			t.Fatalf("ReadActive() = %v, want Pid 1002", active)
		}
	})

	t.Run("returns nil when process is dead", func(t *testing.T) {
		st := sampleState(1003)
		st.Exe = "ffmpeg"
		delete(fake.Alive, 1003)
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		active := ReadActive(path, fake)
		if active != nil {
			t.Fatalf("ReadActive() = %v, want nil", active)
		}
	})

	t.Run("returns nil when process name does not match", func(t *testing.T) {
		st := sampleState(1004)
		st.Exe = "ffmpeg"
		fake.Alive[1004] = "other-process"
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		active := ReadActive(path, fake)
		if active != nil {
			t.Fatalf("ReadActive() = %v, want nil", active)
		}
	})

	t.Run("returns nil when file does not exist", func(t *testing.T) {
		active := ReadActive(filepath.Join(dir, "nonexistent.json"), fake)
		if active != nil {
			t.Fatalf("ReadActive() = %v, want nil", active)
		}
	})
}

func TestActiveRecordingCleansDeadStateAndSidecars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	t.Run("returns state if running and leaves sidecars alone", func(t *testing.T) {
		fake := runner.NewFakeRunner()
		fake.Alive[2001] = "ffmpeg"
		fake.Alive[2002] = "sidecar1"

		st := sampleState(2001)
		st.Sidecars = []Sidecar{{Pid: 2002, Exe: "sidecar1"}}
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		got := ActiveRecording(path, fake)
		if got == nil || got.Pid != 2001 {
			t.Fatalf("ActiveRecording() = %v, want Pid 2001", got)
		}
		if len(fake.KillCalls) != 0 {
			t.Fatalf("KillCalls = %v, want none", fake.KillCalls)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("state file should still exist: %v", err)
		}
	})

	t.Run("cleans dead state and kills alive sidecars", func(t *testing.T) {
		fake := runner.NewFakeRunner()
		// Main process 3001 is dead; sidecar 3002 alive, sidecar 3003 dead
		fake.Alive[3002] = "sidecar1"

		st := sampleState(3001)
		st.Sidecars = []Sidecar{
			{Pid: 3002, Exe: "sidecar1"},
			{Pid: 3003, Exe: "sidecar2"},
		}
		if err := WriteState(path, st); err != nil {
			t.Fatalf("WriteState() error = %v", err)
		}

		got := ActiveRecording(path, fake)
		if got != nil {
			t.Fatalf("ActiveRecording() = %v, want nil", got)
		}
		if len(fake.KillCalls) != 1 || fake.KillCalls[0].Pid != 3002 || fake.KillCalls[0].Sig != syscall.SIGINT {
			t.Fatalf("KillCalls = %v, want 1 SIGINT call for PID 3002", fake.KillCalls)
		}
		if fake.IsAlive(3002) {
			t.Errorf("sidecar 3002 should have been killed")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("state file should have been cleared")
		}
	})

	t.Run("returns nil on missing state file", func(t *testing.T) {
		fake := runner.NewFakeRunner()
		got := ActiveRecording(filepath.Join(dir, "missing.json"), fake)
		if got != nil {
			t.Fatalf("ActiveRecording() = %v, want nil", got)
		}
	})
}
