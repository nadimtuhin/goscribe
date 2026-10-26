package capture

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/ffmpeg"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func TestParseHelperList(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		input := `{
			"displays": [
				{"id": 1, "name": "Built-in Retina", "width": 1920, "height": 1080, "scale": 2.0}
			],
			"windows": [
				{"id": 101, "app": "Terminal", "title": "zsh"}
			]
		}`
		sources, err := ParseHelperList(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &CaptureSources{
			Displays: []Display{
				{ID: 1, Name: "Built-in Retina", Width: 1920, Height: 1080, Scale: 2.0},
			},
			Windows: []WindowInfo{
				{ID: 101, App: "Terminal", Title: "zsh"},
			},
		}
		if !reflect.DeepEqual(sources, want) {
			t.Fatalf("got %+v, want %+v", sources, want)
		}
	})

	t.Run("invalid json returns error with truncated sample", func(t *testing.T) {
		input := strings.Repeat("error info here! ", 20)
		_, err := ParseHelperList(input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.HasPrefix(err.Error(), "capture helper returned unexpected output: ") {
			t.Fatalf("unexpected error message prefix: %v", err)
		}
		sample := strings.TrimPrefix(err.Error(), "capture helper returned unexpected output: ")
		if len(sample) > 200 {
			t.Fatalf("expected sample length <= 200, got %d", len(sample))
		}
	})

	t.Run("missing windows or displays returns error", func(t *testing.T) {
		input := `{"displays": [{"id": 1, "name": "D", "width": 100, "height": 100, "scale": 1}]}`
		_, err := ParseHelperList(input)
		if err == nil || !strings.Contains(err.Error(), "capture helper returned unexpected output") {
			t.Fatalf("expected error for missing windows, got: %v", err)
		}
	})
}

func TestBuildDevicesReport(t *testing.T) {
	helperJSON := `{
		"displays": [
			{"id": 1, "name": "Color LCD", "width": 1512, "height": 982, "scale": 2.0}
		],
		"windows": [
			{"id": 50, "app": "Notes", "title": "Checklist"}
		]
	}`

	ffmpegOutput := `[AVFoundation indev @ 0x1] AVFoundation video devices:
[AVFoundation indev @ 0x1] [0] FaceTime HD Camera
[AVFoundation indev @ 0x1] AVFoundation audio devices:
[AVFoundation indev @ 0x1] [0] MacBook Pro Microphone
[AVFoundation indev @ 0x1] [1] Studio Display Microphone`

	report, err := BuildDevicesReport(helperJSON, ffmpegOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := &DevicesReport{
		Displays: []Display{
			{ID: 1, Name: "Color LCD", Width: 1512, Height: 982, Scale: 2.0},
		},
		Windows: []WindowInfo{
			{ID: 50, App: "Notes", Title: "Checklist"},
		},
		Microphones: []ffmpeg.Device{
			{Index: 0, Name: "MacBook Pro Microphone"},
			{Index: 1, Name: "Studio Display Microphone"},
		},
	}

	if !reflect.DeepEqual(report, want) {
		t.Fatalf("got %+v, want %+v", report, want)
	}

	_, err = BuildDevicesReport("not json", ffmpegOutput)
	if err == nil {
		t.Fatal("expected error on invalid helper stdout")
	}
}

func TestFormatDevicesText(t *testing.T) {
	report := &DevicesReport{
		Displays: []Display{
			{ID: 1, Name: "Color LCD", Width: 1512, Height: 982, Scale: 2.0},
		},
		Windows: []WindowInfo{
			{ID: 50, App: "Notes", Title: "Checklist"},
		},
		Microphones: []ffmpeg.Device{
			{Index: 0, Name: "MacBook Pro Microphone"},
		},
	}

	got := FormatDevicesText(report)
	want := `Displays (id  name  size in points)
  1  Color LCD  1512x982 @2x
Windows (id  app - title, frontmost first)
  50  Notes - Checklist
Microphones (--mic index  name)
  0  MacBook Pro Microphone`

	if got != want {
		t.Fatalf("FormatDevicesText() =\n%q\nwant:\n%q", got, want)
	}
}

func TestListDevices(t *testing.T) {
	ctx := context.Background()

	t.Run("helper execution failed", func(t *testing.T) {
		fake := runner.NewFakeRunner()
		fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
			if strings.Contains(cmd[0], "helper") {
				return "", "permission denied", 1, nil
			}
			return "", "", 0, nil
		}

		_, err := ListDevices(ctx, fake, "/bin/helper")
		if err == nil || !strings.Contains(err.Error(), "listing displays and windows failed (is Screen Recording allowed?):\npermission denied") {
			t.Fatalf("expected permission error, got: %v", err)
		}
	})

	t.Run("helper execution successful", func(t *testing.T) {
		fake := runner.NewFakeRunner()
		fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
			if strings.Contains(cmd[0], "helper") {
				return `{"displays": [{"id": 1, "name": "LCD", "width": 100, "height": 100, "scale": 1}], "windows": []}`, "", 0, nil
			}
			if cmd[0] == "ffmpeg" {
				return "", "[AVFoundation indev @ 0x1] AVFoundation audio devices:\n[AVFoundation indev @ 0x1] [0] Default Mic\n", 0, nil
			}
			return "", "", 0, nil
		}

		report, err := ListDevices(ctx, fake, "/bin/helper")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Displays) != 1 || report.Displays[0].ID != 1 {
			t.Fatalf("unexpected displays: %+v", report.Displays)
		}
		if len(report.Microphones) != 1 || report.Microphones[0].Name != "Default Mic" {
			t.Fatalf("unexpected mics: %+v", report.Microphones)
		}
	})
}
