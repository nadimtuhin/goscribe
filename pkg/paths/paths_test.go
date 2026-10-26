package paths

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestDefaultLocations(t *testing.T) {
	t.Run("with explicit home", func(t *testing.T) {
		home := "/Users/ada"
		if got, want := DefaultRecordingsDir(home), filepath.Join(home, "Movies", "goscribe"); got != want {
			t.Errorf("DefaultRecordingsDir() = %q, want %q", got, want)
		}
		if got, want := StateFilePath(home), filepath.Join(home, ".goscribe", "state.json"); got != want {
			t.Errorf("StateFilePath() = %q, want %q", got, want)
		}
		if got, want := OptimizeQueuePath(home), filepath.Join(home, ".goscribe", "optimize-queue.json"); got != want {
			t.Errorf("OptimizeQueuePath() = %q, want %q", got, want)
		}
		if got, want := OptimizeLogPath(home), filepath.Join(home, ".goscribe", "optimize-worker.log"); got != want {
			t.Errorf("OptimizeLogPath() = %q, want %q", got, want)
		}
	})

	t.Run("with current user home", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("os.UserHomeDir() error: %v", err)
		}
		if got, want := DefaultRecordingsDir(), filepath.Join(home, "Movies", "goscribe"); got != want {
			t.Errorf("DefaultRecordingsDir() = %q, want %q", got, want)
		}
		if got, want := StateFilePath(), filepath.Join(home, ".goscribe", "state.json"); got != want {
			t.Errorf("StateFilePath() = %q, want %q", got, want)
		}
		if got, want := OptimizeQueuePath(), filepath.Join(home, ".goscribe", "optimize-queue.json"); got != want {
			t.Errorf("OptimizeQueuePath() = %q, want %q", got, want)
		}
		if got, want := OptimizeLogPath(), filepath.Join(home, ".goscribe", "optimize-worker.log"); got != want {
			t.Errorf("OptimizeLogPath() = %q, want %q", got, want)
		}
	})
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercases and collapses punctuation and whitespace",
			input: "  Hello, World! -- Let's Go -- ",
			want:  "hello-world-let-s-go",
		},
		{
			name:  "supports unicode letters",
			input: "বাংলা কথা",
			want:  "বাংলা-কথা",
		},
		{
			name:  "caps length at 50 runes",
			input: strings.Repeat("a", 100),
			want:  strings.Repeat("a", 50),
		},
		{
			name:  "empty and punctuation only",
			input: "   --- !!!   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if utf8.RuneCountInString(got) > 50 {
				t.Errorf("Slugify(%q) length %d > 50", tt.input, utf8.RuneCountInString(got))
			}
		})
	}
}

func TestTimestampedFilename(t *testing.T) {
	d := time.Date(2026, 1, 5, 7, 3, 9, 0, time.UTC)

	tests := []struct {
		name     string
		date     time.Time
		fileName string
		ext      string
		want     string
	}{
		{
			name:     "slugified name with default ext",
			date:     d,
			fileName: "Team Sync",
			ext:      "",
			want:     "2026-01-05_07-03-09_team-sync.mp4",
		},
		{
			name:     "explicit m4a extension",
			date:     d,
			fileName: "Voice Memo",
			ext:      "m4a",
			want:     "2026-01-05_07-03-09_voice-memo.m4a",
		},
		{
			name:     "empty name with m4a extension",
			date:     d,
			fileName: "",
			ext:      "m4a",
			want:     "2026-01-05_07-03-09.m4a",
		},
		{
			name:     "whitespace only name defaults to timestamp.mp4",
			date:     d,
			fileName: "   ",
			ext:      "",
			want:     "2026-01-05_07-03-09.mp4",
		},
		{
			name:     "ext with leading dot",
			date:     d,
			fileName: "Test",
			ext:      ".mp4",
			want:     "2026-01-05_07-03-09_test.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimestampedFilename(tt.date, tt.fileName, tt.ext)
			if got != tt.want {
				t.Errorf("TimestampedFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithMp4Extension(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/r/demo", "/r/demo.mp4"},
		{"/r/demo.mov", "/r/demo.mp4"},
		{"/r/demo.mp4", "/r/demo.mp4"},
	}

	for _, tt := range tests {
		if got := WithMp4Extension(tt.input); got != tt.want {
			t.Errorf("WithMp4Extension(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWithAudioExtension(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/r/audio.m4a", "/r/audio.m4a"},
		{"/r/video.mp4", "/r/video.mp4"},
		{"/r/audio", "/r/audio.m4a"},
		{"/r/audio.wav", "/r/audio.m4a"},
	}

	for _, tt := range tests {
		if got := WithAudioExtension(tt.input); got != tt.want {
			t.Errorf("WithAudioExtension(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWithRecordingExtension(t *testing.T) {
	tests := []struct {
		input     string
		audioOnly bool
		want      string
	}{
		{"/r/demo", false, "/r/demo.mp4"},
		{"/r/demo", true, "/r/demo.m4a"},
		{"/r/demo.mp4", true, "/r/demo.mp4"},
		{"/r/demo.m4a", true, "/r/demo.m4a"},
	}

	for _, tt := range tests {
		if got := WithRecordingExtension(tt.input, tt.audioOnly); got != tt.want {
			t.Errorf("WithRecordingExtension(%q, %v) = %q, want %q", tt.input, tt.audioOnly, got, tt.want)
		}
	}
}

func TestUniqueBase(t *testing.T) {
	t.Run("returns unchanged if base already ends in slug", func(t *testing.T) {
		got := UniqueBase("/r/2026-01-05_07-03-09_team-sync", "team-sync", false, nil)
		if want := "/r/2026-01-05_07-03-09_team-sync"; got != want {
			t.Errorf("UniqueBase() = %q, want %q", got, want)
		}
	})

	t.Run("returns unchanged when base already has a slug and name is not explicit", func(t *testing.T) {
		got := UniqueBase("/r/2026-01-05_07-03-09_old-slug", "new-slug", false, nil)
		if want := "/r/2026-01-05_07-03-09_old-slug"; got != want {
			t.Errorf("UniqueBase() = %q, want %q", got, want)
		}
	})

	t.Run("appends slug when explicit even if old slug present", func(t *testing.T) {
		got := UniqueBase("/r/2026-01-05_07-03-09_old-slug", "new-slug", true, func(string) bool { return false })
		if want := "/r/2026-01-05_07-03-09_old-slug_new-slug"; got != want {
			t.Errorf("UniqueBase() = %q, want %q", got, want)
		}
	})

	t.Run("increments counter on collision", func(t *testing.T) {
		call := 0
		exists := func(string) bool {
			res := call < 2
			call++
			return res
		}
		got := UniqueBase("/r/2026-01-05_07-03-09", "sync", false, exists)
		if want := "/r/2026-01-05_07-03-09_sync-2"; got != want {
			t.Errorf("UniqueBase() = %q, want %q", got, want)
		}
	})
}

func TestRenameRecordingFiles(t *testing.T) {
	t.Run("returns nil when oldBase equals newBase", func(t *testing.T) {
		renamed, err := RenameRecordingFiles("/r/same", "/r/same", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if renamed != nil {
			t.Errorf("RenameRecordingFiles() = %v, want nil", renamed)
		}
	})

	t.Run("renames existing recording files to newBase", func(t *testing.T) {
		existing := map[string]bool{
			"/r/old.m4a":           true,
			"/r/old.transcript.md": true,
			"/r/old.notes.md":      true,
		}
		var ops [][2]string
		exists := func(p string) bool {
			return existing[p]
		}
		rename := func(from, to string) error {
			ops = append(ops, [2]string{from, to})
			return nil
		}

		renamed, err := RenameRecordingFiles("/r/old", "/r/new", exists, rename)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantRenamed := []string{"/r/new.m4a", "/r/new.transcript.md", "/r/new.notes.md"}
		if !reflect.DeepEqual(renamed, wantRenamed) {
			t.Errorf("renamed = %v, want %v", renamed, wantRenamed)
		}

		wantOps := [][2]string{
			{"/r/old.m4a", "/r/new.m4a"},
			{"/r/old.transcript.md", "/r/new.transcript.md"},
			{"/r/old.notes.md", "/r/new.notes.md"},
		}
		if !reflect.DeepEqual(ops, wantOps) {
			t.Errorf("rename ops = %v, want %v", ops, wantOps)
		}
	})

	t.Run("returns error if rename fails", func(t *testing.T) {
		exists := func(string) bool { return true }
		renameErr := errors.New("disk full")
		rename := func(from, to string) error {
			return renameErr
		}

		renamed, err := RenameRecordingFiles("/r/old", "/r/new", exists, rename)
		if !errors.Is(err, renameErr) {
			t.Fatalf("expected error %v, got %v", renameErr, err)
		}
		if len(renamed) != 0 {
			t.Errorf("expected no successfully collected paths, got %v", renamed)
		}
	})
}
