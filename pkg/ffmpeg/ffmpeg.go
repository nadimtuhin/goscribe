package ffmpeg

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var devicePattern = regexp.MustCompile(`\] \[(\d+)\] (.+)`)
var toolDirs = []string{"/opt/homebrew/bin", "/usr/local/bin"}

// Device represents an AVFoundation capture device.
type Device struct {
	Index int
	Name  string
}

// ParseDevices parses AVFoundation device lines matching `] [index] name`.
func ParseDevices(section string) []Device {
	devices := []Device{}
	matches := devicePattern.FindAllStringSubmatch(section, -1)
	for _, m := range matches {
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		devices = append(devices, Device{
			Index: idx,
			Name:  strings.TrimSpace(m[2]),
		})
	}
	return devices
}

// SplitDeviceList splits avfoundation device stderr into video and audio sections.
func SplitDeviceList(listOutput string) (video, audio string) {
	video, audio, _ = strings.Cut(listOutput, "AVFoundation audio devices")
	return video, audio
}

// ParseScreens returns all screen capture video devices.
func ParseScreens(listOutput string) []Device {
	video, _ := SplitDeviceList(listOutput)
	screens := []Device{}
	for _, d := range ParseDevices(video) {
		if strings.HasPrefix(d.Name, "Capture screen") {
			screens = append(screens, d)
		}
	}
	return screens
}

// ParseMicrophones returns all audio devices.
func ParseMicrophones(listOutput string) []Device {
	_, audio := SplitDeviceList(listOutput)
	return ParseDevices(audio)
}

// BuildArgs constructs ffmpeg CLI arguments for screen capture (and optional microphone).
func BuildArgs(screenDevice int, micDevice *int, fps int, file string) []string {
	audioInput := "none"
	var audioCodec []string
	if micDevice != nil {
		audioInput = fmt.Sprint(*micDevice)
		audioCodec = []string{"-c:a", "aac", "-b:a", "160k"}
	}
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "avfoundation", "-capture_cursor", "1",
		"-framerate", fmt.Sprint(fps),
		"-i", fmt.Sprintf("%d:%s", screenDevice, audioInput),
		"-c:v", "libx264", "-preset", "veryfast", "-pix_fmt", "yuv420p",
	}
	args = append(args, audioCodec...)
	args = append(args, "-movflags", "+frag_keyframe+empty_moov+default_base_moof", file)
	return args
}

// BuildMicArgs constructs ffmpeg CLI arguments to record audio only.
func BuildMicArgs(micDevice int, file string) []string {
	return []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "avfoundation", "-i", fmt.Sprintf("none:%d", micDevice),
		"-c:a", "aac", "-b:a", "160k",
		file,
	}
}

// BuildMuxArgs constructs ffmpeg CLI arguments to combine screen capture and system audio.
func BuildMuxArgs(video string, videoHasMic bool, systemAudio string, out string) []string {
	inputs := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", video, "-i", systemAudio}
	if videoHasMic {
		return append(inputs,
			"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=first:normalize=0[mixed]",
			"-map", "0:v", "-map", "[mixed]",
			"-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
			out,
		)
	}
	return append(inputs, "-map", "0:v", "-map", "1:a", "-c", "copy", out)
}

// BuildSckMuxArgs constructs ffmpeg CLI arguments to mux capture helper video with audio tracks.
func BuildSckMuxArgs(video string, audio []string, out string) []string {
	inputs := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", video}
	for _, a := range audio {
		inputs = append(inputs, "-i", a)
	}
	if len(audio) == 1 {
		return append(inputs, "-map", "0:v", "-map", "1:a", "-c", "copy", out)
	}
	return append(inputs,
		"-filter_complex", "[1:a][2:a]amix=inputs=2:duration=longest:normalize=0[mixed]",
		"-map", "0:v", "-map", "[mixed]",
		"-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
		out,
	)
}

// BuildAudioMixArgs constructs ffmpeg CLI arguments to mix multiple audio inputs without video.
func BuildAudioMixArgs(inputs []string, out string) []string {
	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	for _, in := range inputs {
		args = append(args, "-i", in)
	}
	return append(args,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=longest:normalize=0[mixed]",
		"-map", "[mixed]",
		"-c:a", "aac", "-b:a", "192k",
		out,
	)
}

// WithToolPaths appends common Homebrew binary paths if missing.
// ponytail: unix colon separator; upgrade to filepath.ListSeparator if Windows supported.
func WithToolPaths(path string) string {
	var present []string
	for _, p := range strings.Split(path, ":") {
		if p != "" {
			present = append(present, p)
		}
	}
	for _, dir := range toolDirs {
		if !slices.Contains(present, dir) {
			present = append(present, dir)
		}
	}
	return strings.Join(present, ":")
}

// EnsureToolPath updates process PATH to include tool directories.
func EnsureToolPath() {
	_ = os.Setenv("PATH", WithToolPaths(os.Getenv("PATH")))
}
