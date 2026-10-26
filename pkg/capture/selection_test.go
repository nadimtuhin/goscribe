package capture

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/state"
)

func TestParseArea(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Area
		wantErr bool
	}{
		{
			name:  "valid area",
			input: "10,20,300,400",
			want:  &Area{X: 10, Y: 20, W: 300, H: 400},
		},
		{
			name:  "valid area with spaces",
			input: " 0 , 5 , 1920 , 1080 ",
			want:  &Area{X: 0, Y: 5, W: 1920, H: 1080},
		},
		{
			name:    "too few parts",
			input:   "10,20,300",
			wantErr: true,
		},
		{
			name:    "too many parts",
			input:   "10,20,300,400,500",
			wantErr: true,
		},
		{
			name:    "empty part",
			input:   "10,,300,400",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			input:   "a,b,c,d",
			wantErr: true,
		},
		{
			name:    "negative x",
			input:   "-1,0,100,100",
			wantErr: true,
		},
		{
			name:    "negative y",
			input:   "0,-1,100,100",
			wantErr: true,
		},
		{
			name:    "zero width",
			input:   "0,0,0,100",
			wantErr: true,
		},
		{
			name:    "zero height",
			input:   "0,0,100,0",
			wantErr: true,
		},
		{
			name:    "negative dimension",
			input:   "0,0,-100,100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArea(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseArea(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseArea(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveWindow(t *testing.T) {
	windows := []WindowInfo{
		{ID: 101, App: "Terminal", Title: "bash - 80x24"},
		{ID: 202, App: "Google Chrome", Title: "Localscribe Docs"},
		{ID: 303, App: "Code", Title: "selection.go - goscribe"},
		{ID: 404, App: "Code", Title: "devices.go - goscribe"},
	}

	t.Run("resolve by id match", func(t *testing.T) {
		w, err := ResolveWindow(windows, "101")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != 101 || w.App != "Terminal" {
			t.Fatalf("unexpected window: %+v", w)
		}
	})

	t.Run("resolve by id not found", func(t *testing.T) {
		_, err := ResolveWindow(windows, "999")
		if err == nil || !strings.Contains(err.Error(), "no window with id 999. Run: goscribe devices") {
			t.Fatalf("expected not found error, got: %v", err)
		}
	})

	t.Run("resolve by exact app name", func(t *testing.T) {
		w, err := ResolveWindow(windows, "terminal")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != 101 {
			t.Fatalf("expected ID 101, got %d", w.ID)
		}
	})

	t.Run("resolve by substring in title", func(t *testing.T) {
		w, err := ResolveWindow(windows, "docs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != 202 {
			t.Fatalf("expected ID 202, got %d", w.ID)
		}
	})

	t.Run("resolve not found", func(t *testing.T) {
		_, err := ResolveWindow(windows, "Safari")
		if err == nil || !strings.Contains(err.Error(), "no window matches \"Safari\". Run: goscribe devices") {
			t.Fatalf("expected no window matches error, got: %v", err)
		}
	})

	t.Run("resolve ambiguous", func(t *testing.T) {
		_, err := ResolveWindow(windows, "goscribe")
		if err == nil || !strings.Contains(err.Error(), "is ambiguous, pick one by id:") {
			t.Fatalf("expected ambiguous error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "303") || !strings.Contains(err.Error(), "404") {
			t.Fatalf("expected listing matching ids, got: %v", err)
		}
	})
}

func TestResolveDisplay(t *testing.T) {
	displays := []Display{
		{ID: 1, Name: "Built-in Retina", Width: 1920, Height: 1080, Scale: 2.0},
		{ID: 2, Name: "External 4K", Width: 3840, Height: 2160, Scale: 1.0},
	}

	t.Run("valid display", func(t *testing.T) {
		d, err := ResolveDisplay(displays, "2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.ID != 2 || d.Name != "External 4K" {
			t.Fatalf("unexpected display: %+v", d)
		}
	})

	t.Run("non-numeric display", func(t *testing.T) {
		_, err := ResolveDisplay(displays, "retina")
		if err == nil || !strings.Contains(err.Error(), "--display expects a display id, got \"retina\"") {
			t.Fatalf("expected invalid format error, got: %v", err)
		}
	})

	t.Run("display not found", func(t *testing.T) {
		_, err := ResolveDisplay(displays, "3")
		if err == nil || !strings.Contains(err.Error(), "display 3 not found. Available ids: 1, 2") {
			t.Fatalf("expected display not found error, got: %v", err)
		}
	})
}

func TestResolveSelection(t *testing.T) {
	sources := CaptureSources{
		Displays: []Display{
			{ID: 1, Name: "Display 1", Width: 1920, Height: 1080, Scale: 2.0},
			{ID: 2, Name: "Display 2", Width: 1280, Height: 720, Scale: 1.0},
		},
		Windows: []WindowInfo{
			{ID: 10, App: "Notes", Title: "Todo"},
		},
	}

	t.Run("conflict window and display", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Window: "10", Display: "1"}, sources)
		if err == nil || !strings.Contains(err.Error(), "use only one of --window, --area, --display") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})

	t.Run("conflict window and area", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Window: "10", Area: "0,0,100,100"}, sources)
		if err == nil || !strings.Contains(err.Error(), "use only one of --window, --area, --display") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})

	t.Run("window selection", func(t *testing.T) {
		sel, err := ResolveSelection(SelectionFlags{Window: "10"}, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &state.Selection{Kind: "window", WindowID: 10, App: "Notes", Title: "Todo"}
		if !reflect.DeepEqual(sel, want) {
			t.Fatalf("got %+v, want %+v", sel, want)
		}
	})

	t.Run("window resolution error", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Window: "99"}, sources)
		if err == nil {
			t.Fatal("expected error for unknown window")
		}
	})

	t.Run("display selection", func(t *testing.T) {
		sel, err := ResolveSelection(SelectionFlags{Display: "2"}, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &state.Selection{Kind: "display", DisplayID: 2}
		if !reflect.DeepEqual(sel, want) {
			t.Fatalf("got %+v, want %+v", sel, want)
		}
	})

	t.Run("display resolution error", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Display: "5"}, sources)
		if err == nil {
			t.Fatal("expected error for unknown display")
		}
	})

	t.Run("area selection with explicit display", func(t *testing.T) {
		sel, err := ResolveSelection(SelectionFlags{Area: "10,20,500,400", Display: "2"}, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &state.Selection{Kind: "area", DisplayID: 2, X: 10, Y: 20, W: 500, H: 400}
		if !reflect.DeepEqual(sel, want) {
			t.Fatalf("got %+v, want %+v", sel, want)
		}
	})

	t.Run("area selection defaults to first display", func(t *testing.T) {
		sel, err := ResolveSelection(SelectionFlags{Area: "0,0,1000,800"}, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &state.Selection{Kind: "area", DisplayID: 1, X: 0, Y: 0, W: 1000, H: 800}
		if !reflect.DeepEqual(sel, want) {
			t.Fatalf("got %+v, want %+v", sel, want)
		}
	})

	t.Run("area selection no display found", func(t *testing.T) {
		emptySources := CaptureSources{}
		_, err := ResolveSelection(SelectionFlags{Area: "0,0,100,100"}, emptySources)
		if err == nil || !strings.Contains(err.Error(), "no display found") {
			t.Fatalf("expected 'no display found' error, got: %v", err)
		}
	})

	t.Run("area selection invalid format", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Area: "invalid"}, sources)
		if err == nil {
			t.Fatal("expected error for invalid area string")
		}
	})

	t.Run("area selection out of display bounds", func(t *testing.T) {
		_, err := ResolveSelection(SelectionFlags{Area: "1000,0,500,500", Display: "2"}, sources)
		if err == nil || !strings.Contains(err.Error(), "--area is outside display 2 (1280x720 points)") {
			t.Fatalf("expected out of bounds error, got: %v", err)
		}
	})

	t.Run("empty flags returns nil selection", func(t *testing.T) {
		sel, err := ResolveSelection(SelectionFlags{}, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sel != nil {
			t.Fatalf("expected nil selection, got: %+v", sel)
		}
	})
}

func TestCaptureArgs(t *testing.T) {
	t.Run("nil selection", func(t *testing.T) {
		args := CaptureArgs(nil, "out.mp4", "", 30)
		want := []string{"--fps", "30", "--out", "out.mp4"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("CaptureArgs() = %v, want %v", args, want)
		}
	})

	t.Run("display selection with system audio", func(t *testing.T) {
		sel := &state.Selection{Kind: "display", DisplayID: 1}
		args := CaptureArgs(sel, "out.mp4", "sys.m4a", 60)
		want := []string{"--display", "1", "--fps", "60", "--out", "out.mp4", "--system-audio", "sys.m4a"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("CaptureArgs() = %v, want %v", args, want)
		}
	})

	t.Run("window selection", func(t *testing.T) {
		sel := &state.Selection{Kind: "window", WindowID: 42}
		args := CaptureArgs(sel, "out.mp4", "", 30)
		want := []string{"--window", "42", "--fps", "30", "--out", "out.mp4"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("CaptureArgs() = %v, want %v", args, want)
		}
	})

	t.Run("area selection", func(t *testing.T) {
		sel := &state.Selection{Kind: "area", DisplayID: 2, X: 10, Y: 20, W: 800, H: 600}
		args := CaptureArgs(sel, "out.mp4", "", 24)
		want := []string{"--area", "10,20,800,600", "--display", "2", "--fps", "24", "--out", "out.mp4"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("CaptureArgs() = %v, want %v", args, want)
		}
	})
}

func TestParsePick(t *testing.T) {
	t.Run("empty string returns cancelled error", func(t *testing.T) {
		_, err := ParsePick("   \n")
		if err == nil || err.Error() != "selection cancelled" {
			t.Fatalf("expected 'selection cancelled', got: %v", err)
		}
	})

	t.Run("invalid json returns unmarshal error", func(t *testing.T) {
		_, err := ParsePick("{not-json}")
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("window pick", func(t *testing.T) {
		got, err := ParsePick(`{"window": 1234}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &SelectionFlags{Window: "1234"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParsePick() = %+v, want %+v", got, want)
		}
	})

	t.Run("area pick with display", func(t *testing.T) {
		got, err := ParsePick(`{"x": 100, "y": 200, "w": 400, "h": 300, "displayId": 2}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &SelectionFlags{Area: "100,200,400,300", Display: "2"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParsePick() = %+v, want %+v", got, want)
		}
	})

	t.Run("area pick without display", func(t *testing.T) {
		got, err := ParsePick(`{"x": 0, "y": 0, "w": 500, "h": 500}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &SelectionFlags{Area: "0,0,500,500", Display: ""}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParsePick() = %+v, want %+v", got, want)
		}
	})

	t.Run("empty object returns cancelled error", func(t *testing.T) {
		_, err := ParsePick(`{}`)
		if err == nil || err.Error() != "selection cancelled" {
			t.Fatalf("expected 'selection cancelled', got: %v", err)
		}
	})
}
