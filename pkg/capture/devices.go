package capture

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/ffmpeg"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

type DevicesReport struct {
	Displays    []Display       `json:"displays"`
	Windows     []WindowInfo    `json:"windows"`
	Microphones []ffmpeg.Device `json:"microphones"`
}

func ParseHelperList(helperStdout string) (*CaptureSources, error) {
	var parsed struct {
		Displays []Display    `json:"displays"`
		Windows  []WindowInfo `json:"windows"`
	}
	if err := json.Unmarshal([]byte(helperStdout), &parsed); err == nil && parsed.Displays != nil && parsed.Windows != nil {
		return &CaptureSources{Displays: parsed.Displays, Windows: parsed.Windows}, nil
	}
	sample := helperStdout
	if len(sample) > 200 {
		sample = sample[:200]
	}
	return nil, fmt.Errorf("capture helper returned unexpected output: %s", sample)
}

func BuildDevicesReport(helperStdout string, ffmpegDeviceListing string) (*DevicesReport, error) {
	sources, err := ParseHelperList(helperStdout)
	if err != nil {
		return nil, err
	}
	mics := ffmpeg.ParseMicrophones(ffmpegDeviceListing)
	return &DevicesReport{
		Displays:    sources.Displays,
		Windows:     sources.Windows,
		Microphones: mics,
	}, nil
}

func FormatDevicesText(report *DevicesReport) string {
	var lines []string
	lines = append(lines, "Displays (id  name  size in points)")
	for _, d := range report.Displays {
		lines = append(lines, fmt.Sprintf("  %d  %s  %dx%d @%.0fx", d.ID, d.Name, d.Width, d.Height, d.Scale))
	}
	lines = append(lines, "Windows (id  app - title, frontmost first)")
	for _, w := range report.Windows {
		lines = append(lines, fmt.Sprintf("  %d  %s - %s", w.ID, w.App, w.Title))
	}
	lines = append(lines, "Microphones (--mic index  name)")
	for _, m := range report.Microphones {
		lines = append(lines, fmt.Sprintf("  %d  %s", m.Index, m.Name))
	}
	return strings.Join(lines, "\n")
}

func ListDevices(ctx context.Context, r runner.Runner, captureHelper string) (*DevicesReport, error) {
	stdout, stderr, code, _ := r.Run(ctx, []string{captureHelper, "--list", "--json"})
	if code != 0 {
		return nil, fmt.Errorf("listing displays and windows failed (is Screen Recording allowed?):\n%s", strings.TrimSpace(stderr))
	}
	_, ffmpegStderr, _, _ := r.Run(ctx, []string{"ffmpeg", "-hide_banner", "-f", "avfoundation", "-list_devices", "true", "-i", ""})
	return BuildDevicesReport(stdout, ffmpegStderr)
}
