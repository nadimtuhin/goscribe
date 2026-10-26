package recorder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/ffmpeg"
	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

const (
	DefaultStartupCheckMs = 200 * time.Millisecond
	DefaultStopTimeoutMs  = 5 * time.Second
)

// EnsureHelper ensures the Swift helper binary is compiled and up to date.
func EnsureHelper(r runner.Runner, source, binary string) (string, error) {
	sBin, errBin := os.Stat(binary)
	sSrc, errSrc := os.Stat(source)
	if errBin == nil && errSrc == nil && (sBin.ModTime().After(sSrc.ModTime()) || sBin.ModTime().Equal(sSrc.ModTime())) {
		return binary, nil
	}
	if _, err := r.Which("swiftc"); err != nil {
		return "", errors.New("swiftc not found. Install Xcode command line tools: xcode-select --install")
	}
	if err := os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
		return "", err
	}
	_, stderr, exitCode, err := r.Run(context.Background(), []string{"swiftc", "-O", source, "-o", binary})
	if err != nil || exitCode != 0 {
		return "", fmt.Errorf("building %s failed:\n%s", filepath.Base(binary), stderr)
	}
	return binary, nil
}

type StartOptions struct {
	Out           string
	Name          string
	AudioOnly     bool
	Screen        *int
	Fps           int
	Mic           *int
	NoMic         bool
	NoSystemAudio bool
	Engine        string
	Selection     *state.Selection
	PluginOptions map[string]any
}

func Start(opts StartOptions, statePath string, r runner.Runner) (*state.RecordingState, error) {
	if _, err := r.Which("ffmpeg"); err != nil {
		return nil, errors.New("ffmpeg not found. Install it: brew install ffmpeg")
	}
	if existing := state.ActiveRecording(statePath, r); existing != nil {
		return nil, fmt.Errorf("already recording: %s", existing.File)
	}

	engine := opts.Engine
	if engine == "" {
		if opts.Selection != nil {
			engine = "sck"
		} else {
			engine = "ffmpeg"
		}
	}
	if engine == "ffmpeg" && opts.Selection != nil {
		return nil, errors.New("--window, --area and --display need --engine sck")
	}

	var targetFile string
	if opts.AudioOnly {
		if opts.Out != "" {
			targetFile = paths.WithAudioExtension(opts.Out)
		} else {
			targetFile = filepath.Join(paths.DefaultRecordingsDir(), paths.TimestampedFilename(time.Now(), opts.Name, "m4a"))
		}
	} else {
		if opts.Out != "" {
			targetFile = paths.WithMp4Extension(opts.Out)
		} else {
			targetFile = filepath.Join(paths.DefaultRecordingsDir(), paths.TimestampedFilename(time.Now(), opts.Name, "mp4"))
		}
	}

	recordsSystemAudio := !opts.NoSystemAudio
	recordsMic := !opts.NoMic
	partsBase := strings.TrimSuffix(strings.TrimSuffix(targetFile, ".mp4"), ".m4a")

	var videoFile string
	if opts.AudioOnly {
		videoFile = targetFile
	} else if recordsSystemAudio || engine == "sck" {
		videoFile = partsBase + ".video.mp4"
	} else {
		videoFile = targetFile
	}

	systemAudioFile := partsBase + ".system.m4a"
	micFile := partsBase + ".mic.m4a"

	toCheck := []string{targetFile}
	if !opts.AudioOnly && videoFile != targetFile {
		toCheck = append(toCheck, videoFile)
	}
	if recordsSystemAudio {
		toCheck = append(toCheck, systemAudioFile)
	}
	if recordsMic {
		toCheck = append(toCheck, micFile)
	}
	for _, f := range toCheck {
		if _, err := os.Stat(f); err == nil {
			return nil, fmt.Errorf("refusing to overwrite existing file: %s", f)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		return nil, err
	}
	logDir := filepath.Dir(statePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	ffmpegLog := filepath.Join(logDir, "ffmpeg.log")
	helperLog := filepath.Join(logDir, "sysaudio.log")

	fps := opts.Fps
	if fps <= 0 {
		fps = 30
	}

	recState := &state.RecordingState{
		File:          targetFile,
		VideoFile:     videoFile,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		Engine:        engine,
		AudioOnly:     opts.AudioOnly,
		PluginOptions: opts.PluginOptions,
		Selection:     opts.Selection,
	}

	type procItem struct {
		pid   int
		label string
		log   string
	}
	var procs []procItem

	if opts.AudioOnly {
		var micPart *state.AudioPart
		if recordsMic {
			micDev := 0
			if opts.Mic != nil {
				micDev = *opts.Mic
			}
			pid, err := r.SpawnDetached("ffmpeg", ffmpeg.BuildMicArgs(micDev, micFile), ffmpegLog)
			if err != nil {
				return nil, err
			}
			micPart = &state.AudioPart{Pid: pid, File: micFile}
			procs = append(procs, procItem{pid: pid, label: "ffmpeg", log: ffmpegLog})
		}
		var sysPart *state.AudioPart
		if recordsSystemAudio {
			helper, err := EnsureHelper(r, "native/sysaudio.swift", filepath.Join(logDir, "sysaudio"))
			if err != nil {
				return nil, err
			}
			pid, err := r.SpawnDetached(helper, []string{systemAudioFile}, helperLog)
			if err != nil {
				return nil, err
			}
			sysPart = &state.AudioPart{Pid: pid, File: systemAudioFile}
			procs = append(procs, procItem{pid: pid, label: "sysaudio", log: helperLog})
		}
		if micPart == nil && sysPart == nil {
			return nil, errors.New("cannot record audio-only with both --no-mic and --no-system-audio")
		}
		primary := micPart
		if primary == nil {
			primary = sysPart
		}
		recState.Pid = primary.Pid
		if micPart != nil {
			recState.Exe = "ffmpeg"
		} else {
			recState.Exe = "sysaudio"
		}
		recState.Mic = micPart
		recState.SystemAudio = sysPart
		recState.VideoHasMic = recordsMic
	} else if engine == "ffmpeg" {
		screenDev := 0
		if opts.Screen != nil {
			screenDev = *opts.Screen
		}
		var micDev *int
		if recordsMic {
			d := 0
			if opts.Mic != nil {
				d = *opts.Mic
			}
			micDev = &d
		}
		pid, err := r.SpawnDetached("ffmpeg", ffmpeg.BuildArgs(screenDev, micDev, fps, videoFile), ffmpegLog)
		if err != nil {
			return nil, err
		}
		procs = append(procs, procItem{pid: pid, label: "ffmpeg", log: ffmpegLog})
		recState.Pid = pid
		recState.Exe = "ffmpeg"
		recState.VideoHasMic = (micDev != nil)

		if recordsSystemAudio {
			helper, err := EnsureHelper(r, "native/sysaudio.swift", filepath.Join(logDir, "sysaudio"))
			if err != nil {
				return nil, err
			}
			spid, err := r.SpawnDetached(helper, []string{systemAudioFile}, helperLog)
			if err != nil {
				return nil, err
			}
			recState.SystemAudio = &state.AudioPart{Pid: spid, File: systemAudioFile}
			procs = append(procs, procItem{pid: spid, label: "sysaudio", log: helperLog})
		}
	}

	for _, p := range procs {
		recState.Processes = append(recState.Processes, p.pid)
	}

	if err := state.WriteState(statePath, recState); err != nil {
		return nil, err
	}

	r.Sleep(DefaultStartupCheckMs)
	for _, p := range procs {
		if !r.IsAlive(p.pid) {
			_ = state.ClearState(statePath)
			for _, other := range procs {
				if other.pid != p.pid && r.IsAlive(other.pid) {
					_ = r.Kill(other.pid, syscall.SIGKILL)
				}
			}
			return nil, fmt.Errorf("%s exited on startup", p.label)
		}
	}

	return recState, nil
}

type StopResult struct {
	File       string
	DurationMs int64
	Bytes      int64
	Forced     bool
	KeptParts  []string
	MixError   string
}

func hasContent(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}

func Stop(statePath string, r runner.Runner) (*StopResult, error) {
	st := state.ActiveRecording(statePath, r)
	if st == nil {
		return nil, errors.New("not recording")
	}

	forced := false
	pidsToKill := []int{st.Pid}
	if st.SystemAudio != nil && st.SystemAudio.Pid != st.Pid {
		pidsToKill = append(pidsToKill, st.SystemAudio.Pid)
	}
	if st.Mic != nil && st.Mic.Pid != st.Pid {
		pidsToKill = append(pidsToKill, st.Mic.Pid)
	}
	for _, sc := range st.Sidecars {
		pidsToKill = append(pidsToKill, sc.Pid)
	}

	for _, pid := range pidsToKill {
		if r.IsAlive(pid) {
			_ = r.Kill(pid, syscall.SIGINT)
		}
	}

	deadline := time.Now().Add(DefaultStopTimeoutMs)
	for time.Now().Before(deadline) {
		anyAlive := false
		for _, pid := range pidsToKill {
			if r.IsAlive(pid) {
				anyAlive = true
				break
			}
		}
		if !anyAlive {
			break
		}
		r.Sleep(100 * time.Millisecond)
	}

	for _, pid := range pidsToKill {
		if r.IsAlive(pid) {
			forced = true
			_ = r.Kill(pid, syscall.SIGKILL)
		}
	}

	_ = state.ClearState(statePath)

	durationMs := int64(0)
	if started, err := time.Parse(time.RFC3339, st.StartedAt); err == nil {
		durationMs = time.Since(started).Milliseconds()
		if durationMs < 0 {
			durationMs = 0
		}
	}

	res := &StopResult{
		File:       st.File,
		DurationMs: durationMs,
		Forced:     forced,
	}

	if st.AudioOnly {
		var audioParts []string
		if st.SystemAudio != nil {
			audioParts = append(audioParts, st.SystemAudio.File)
		}
		if st.Mic != nil {
			audioParts = append(audioParts, st.Mic.File)
		}
		var recorded []string
		for _, p := range audioParts {
			if hasContent(p) {
				recorded = append(recorded, p)
			}
		}

		if len(recorded) == 2 {
			_, stderr, code, _ := r.Run(context.Background(), append([]string{"ffmpeg"}, ffmpeg.BuildAudioMixArgs(recorded, st.File)...))
			if code == 0 {
				for _, p := range audioParts {
					_ = os.Remove(p)
				}
			} else {
				res.KeptParts = recorded
				res.MixError = strings.TrimSpace(stderr)
			}
		} else if len(recorded) == 1 {
			_ = os.Rename(recorded[0], st.File)
			for _, p := range audioParts {
				if p != recorded[0] {
					_ = os.Remove(p)
				}
			}
		} else {
			res.MixError = "no audio recorded"
		}
	} else if st.SystemAudio != nil {
		hasSys := hasContent(st.SystemAudio.File)
		hasVid := hasContent(st.VideoFile)

		if !hasVid {
			if hasSys {
				res.KeptParts = []string{st.SystemAudio.File}
			}
			res.MixError = fmt.Sprintf("video part missing: %s", st.VideoFile)
		} else if !hasSys {
			_ = os.Rename(st.VideoFile, st.File)
			_ = os.Remove(st.SystemAudio.File)
		} else {
			muxArgs := ffmpeg.BuildMuxArgs(st.VideoFile, st.VideoHasMic, st.SystemAudio.File, st.File)
			_, stderr, code, _ := r.Run(context.Background(), append([]string{"ffmpeg"}, muxArgs...))
			if code == 0 {
				_ = os.Remove(st.VideoFile)
				_ = os.Remove(st.SystemAudio.File)
			} else {
				res.KeptParts = []string{st.VideoFile, st.SystemAudio.File}
				res.MixError = strings.TrimSpace(stderr)
			}
		}
	}

	if info, err := os.Stat(st.File); err == nil {
		res.Bytes = info.Size()
	}

	return res, nil
}
