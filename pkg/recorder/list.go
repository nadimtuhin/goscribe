package recorder

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RecordingFile represents a single recorded media file.
type RecordingFile struct {
	Name     string
	Path     string
	Bytes    int64
	Modified time.Time
}

// List scans dir for completed .mp4 and .m4a recordings, sorted newest first.
func List(dir string) ([]RecordingFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []RecordingFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".mp4") && !strings.HasSuffix(name, ".m4a") {
			continue
		}
		if strings.HasSuffix(name, ".part.mp4") || strings.HasSuffix(name, ".part.m4a") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		files = append(files, RecordingFile{
			Name:     name,
			Path:     filepath.Join(dir, name),
			Bytes:    info.Size(),
			Modified: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Modified.After(files[j].Modified)
	})

	return files, nil
}

// FormatDuration formats milliseconds into "XmYYs".
func FormatDuration(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	s := int64(math.Round(float64(ms) / 1000.0))
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

// FormatSize formats byte size into "X.Y MB".
func FormatSize(bytes int64) string {
	return fmt.Sprintf("%.1f MB", float64(bytes)/1024.0/1024.0)
}
