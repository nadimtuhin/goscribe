package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

type Sidecar struct {
	Pid   int    `json:"pid"`
	Exe   string `json:"exe"`
	Drain bool   `json:"drain,omitempty"`
}

type AudioPart struct {
	Pid  int    `json:"pid"`
	File string `json:"file"`
}

type RecordingState struct {
	Pid             int            `json:"pid"`
	Exe             string         `json:"exe,omitempty"`
	Engine          string         `json:"engine,omitempty"`
	File            string         `json:"file"`
	VideoFile       string         `json:"videoFile"`
	VideoHasMic     bool           `json:"videoHasMic"`
	AudioOnly       bool           `json:"audioOnly,omitempty"`
	SystemAudio     *AudioPart     `json:"systemAudio,omitempty"`
	SystemAudioFile string         `json:"systemAudioFile,omitempty"`
	Mic             *AudioPart     `json:"mic,omitempty"`
	StartedAt       string         `json:"startedAt"`
	PluginOptions   map[string]any `json:"pluginOptions,omitempty"`
	Sidecars        []Sidecar      `json:"sidecars,omitempty"`
	Processes       []int          `json:"processes,omitempty"`
}

func exeOrDefault(exe string) string {
	if exe == "" {
		return "ffmpeg"
	}
	return exe
}

// WriteState writes recording state atomically by writing to a temporary file then renaming.
func WriteState(path string, state *RecordingState) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(temp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

// ReadState reads and unmarshals the recording state from disk.
func ReadState(path string) (*RecordingState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state RecordingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// ClearState removes the state file, ignoring not-found errors.
func ClearState(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReadActive returns state if present and primary process is running, without mutating files or processes.
func ReadActive(path string, r runner.Runner) *RecordingState {
	if r == nil {
		r = runner.DefaultRunner
	}
	state, err := ReadState(path)
	if err != nil || state == nil {
		return nil
	}
	if r.IsRunning(state.Pid, exeOrDefault(state.Exe)) {
		return state
	}
	return nil
}

// ActiveRecording returns state if running; if stale, it interrupts alive sidecars and clears the state file.
func ActiveRecording(path string, r runner.Runner) *RecordingState {
	if r == nil {
		r = runner.DefaultRunner
	}
	state, err := ReadState(path)
	if err != nil || state == nil {
		return nil
	}
	if r.IsRunning(state.Pid, exeOrDefault(state.Exe)) {
		return state
	}
	// ponytail: kill sidecars with SIGINT; add escalation to SIGKILL after timeout if sidecars linger.
	for _, sc := range state.Sidecars {
		if r.IsAlive(sc.Pid) {
			_ = r.Kill(sc.Pid, syscall.SIGINT)
		}
	}
	_ = ClearState(path)
	return nil
}
