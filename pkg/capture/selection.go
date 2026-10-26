package capture

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/state"
)

type Display struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float64 `json:"scale"`
}

type WindowInfo struct {
	ID    int    `json:"id"`
	App   string `json:"app"`
	Title string `json:"title"`
}

type CaptureSources struct {
	Displays []Display    `json:"displays"`
	Windows  []WindowInfo `json:"windows"`
}

type Area struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type SelectionFlags struct {
	Window  string
	Area    string
	Display string
}

func ParseArea(text string) (*Area, error) {
	parts := strings.Split(text, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("--area expects x,y,w,h in points (x,y >= 0, w,h > 0), got %q", text)
	}
	nums := make([]int, 4)
	for i, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			return nil, fmt.Errorf("--area expects x,y,w,h in points (x,y >= 0, w,h > 0), got %q", text)
		}
		val, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("--area expects x,y,w,h in points (x,y >= 0, w,h > 0), got %q", text)
		}
		nums[i] = val
	}
	x, y, w, h := nums[0], nums[1], nums[2], nums[3]
	if x < 0 || y < 0 || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("--area expects x,y,w,h in points (x,y >= 0, w,h > 0), got %q", text)
	}
	return &Area{X: x, Y: y, W: w, H: h}, nil
}

func ResolveWindow(windows []WindowInfo, query string) (*WindowInfo, error) {
	if id, err := strconv.Atoi(query); err == nil {
		for _, w := range windows {
			if w.ID == id {
				return &w, nil
			}
		}
		return nil, fmt.Errorf("no window with id %s. Run: goscribe devices", query)
	}
	needle := strings.ToLower(query)
	for _, w := range windows {
		if strings.ToLower(w.App) == needle {
			return &w, nil
		}
	}
	var matches []WindowInfo
	for _, w := range windows {
		full := strings.ToLower(w.App + " " + w.Title)
		if strings.Contains(full, needle) {
			matches = append(matches, w)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no window matches %q. Run: goscribe devices", query)
	}
	if len(matches) > 1 {
		lines := make([]string, len(matches))
		for i, m := range matches {
			lines[i] = fmt.Sprintf("  %d  %s - %s", m.ID, m.App, m.Title)
		}
		return nil, fmt.Errorf("%q is ambiguous, pick one by id:\n%s", query, strings.Join(lines, "\n"))
	}
	return &matches[0], nil
}

func ResolveDisplay(displays []Display, text string) (*Display, error) {
	id, err := strconv.Atoi(text)
	if err != nil {
		return nil, fmt.Errorf("--display expects a display id, got %q", text)
	}
	for _, d := range displays {
		if d.ID == id {
			return &d, nil
		}
	}
	var ids []string
	for _, d := range displays {
		ids = append(ids, strconv.Itoa(d.ID))
	}
	return nil, fmt.Errorf("display %s not found. Available ids: %s", text, strings.Join(ids, ", "))
}

func ResolveSelection(flags SelectionFlags, sources CaptureSources) (*state.Selection, error) {
	if flags.Window != "" && (flags.Area != "" || flags.Display != "") {
		return nil, errors.New("use only one of --window, --area, --display (--display may accompany --area)")
	}
	if flags.Window != "" {
		w, err := ResolveWindow(sources.Windows, flags.Window)
		if err != nil {
			return nil, err
		}
		return &state.Selection{Kind: "window", WindowID: w.ID, App: w.App, Title: w.Title}, nil
	}

	var display *Display
	if flags.Display != "" {
		d, err := ResolveDisplay(sources.Displays, flags.Display)
		if err != nil {
			return nil, err
		}
		display = d
	}

	if flags.Area != "" {
		target := display
		if target == nil && len(sources.Displays) > 0 {
			target = &sources.Displays[0]
		}
		if target == nil {
			return nil, errors.New("no display found")
		}
		area, err := ParseArea(flags.Area)
		if err != nil {
			return nil, err
		}
		if area.X+area.W > target.Width || area.Y+area.H > target.Height {
			return nil, fmt.Errorf("--area is outside display %d (%dx%d points)", target.ID, target.Width, target.Height)
		}
		return &state.Selection{
			Kind:      "area",
			DisplayID: target.ID,
			X:         area.X,
			Y:         area.Y,
			W:         area.W,
			H:         area.H,
		}, nil
	}

	if display != nil {
		return &state.Selection{Kind: "display", DisplayID: display.ID}, nil
	}

	return nil, nil
}

func CaptureArgs(selection *state.Selection, out string, systemAudio string, fps int) []string {
	var target []string
	if selection != nil {
		switch selection.Kind {
		case "display":
			target = []string{"--display", strconv.Itoa(selection.DisplayID)}
		case "window":
			target = []string{"--window", strconv.Itoa(selection.WindowID)}
		case "area":
			target = []string{"--area", fmt.Sprintf("%d,%d,%d,%d", selection.X, selection.Y, selection.W, selection.H), "--display", strconv.Itoa(selection.DisplayID)}
		}
	}
	args := append(target, "--fps", strconv.Itoa(fps), "--out", out)
	if systemAudio != "" {
		args = append(args, "--system-audio", systemAudio)
	}
	return args
}

func ParsePick(stdout string) (*SelectionFlags, error) {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil, errors.New("selection cancelled")
	}
	var pick struct {
		Window    *int `json:"window"`
		DisplayID *int `json:"displayId"`
		X         *int `json:"x"`
		Y         *int `json:"y"`
		W         *int `json:"w"`
		H         *int `json:"h"`
	}
	if err := json.Unmarshal([]byte(trimmed), &pick); err != nil {
		return nil, err
	}
	if pick.Window != nil {
		return &SelectionFlags{Window: strconv.Itoa(*pick.Window)}, nil
	}
	if pick.X != nil && pick.Y != nil && pick.W != nil && pick.H != nil {
		dispStr := ""
		if pick.DisplayID != nil {
			dispStr = strconv.Itoa(*pick.DisplayID)
		}
		return &SelectionFlags{
			Area:    fmt.Sprintf("%d,%d,%d,%d", *pick.X, *pick.Y, *pick.W, *pick.H),
			Display: dispStr,
		}, nil
	}
	return nil, errors.New("selection cancelled")
}
