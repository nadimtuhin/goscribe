package recorder

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListNonExistentDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	files, err := List(nonExistent)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected empty files slice, got %d items", len(files))
	}
}

func TestListFilesFilteringAndSorting(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().Truncate(time.Second)
	t1 := now.Add(-3 * time.Minute)
	t2 := now.Add(-2 * time.Minute)
	t3 := now.Add(-1 * time.Minute)

	filesToCreate := []struct {
		name    string
		content []byte
		modTime time.Time
	}{
		{"old.mp4", []byte("old-mp4-data"), t1},
		{"mid.m4a", []byte("mid-m4a-data-123"), t2},
		{"new.mp4", []byte("new-mp4"), t3},
		{"active.part.mp4", []byte("ignored-part-mp4"), now},
		{"active.part.m4a", []byte("ignored-part-m4a"), now},
		{"notes.txt", []byte("ignored-text"), now},
	}

	for _, tc := range filesToCreate {
		p := filepath.Join(dir, tc.name)
		if err := os.WriteFile(p, tc.content, 0o644); err != nil {
			t.Fatalf("failed to write test file %s: %v", tc.name, err)
		}
		if err := os.Chtimes(p, tc.modTime, tc.modTime); err != nil {
			t.Fatalf("failed to chtimes for test file %s: %v", tc.name, err)
		}
	}

	files, err := List(dir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	expectedNames := []string{"new.mp4", "mid.m4a", "old.mp4"}
	expectedBytes := []int64{int64(len("new-mp4")), int64(len("mid-m4a-data-123")), int64(len("old-mp4-data"))}

	for i, expected := range expectedNames {
		if files[i].Name != expected {
			t.Errorf("at index %d: expected name %q, got %q", i, expected, files[i].Name)
		}
		expectedPath := filepath.Join(dir, expected)
		if files[i].Path != expectedPath {
			t.Errorf("at index %d: expected path %q, got %q", i, expectedPath, files[i].Path)
		}
		if files[i].Bytes != expectedBytes[i] {
			t.Errorf("at index %d: expected %d bytes, got %d", i, expectedBytes[i], files[i].Bytes)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0m00s"},
		{-500, "0m00s"},
		{500, "0m01s"},
		{5000, "0m05s"},
		{65000, "1m05s"},
		{3661000, "61m01s"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q; want %q", tt.ms, got, tt.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0.0 MB"},
		{524288, "0.5 MB"},
		{1048576, "1.0 MB"},
		{15728640, "15.0 MB"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q; want %q", tt.bytes, got, tt.want)
		}
	}
}
