package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	nonLetterDigitMark   = regexp.MustCompile(`[^\p{L}\p{M}\p{N}]+`)
	timestampSlugRegex   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}_.+`)
	recordingExtensions  = []string{".mp4", ".m4a", ".transcript.md", ".notes.md", ".live.txt", ".manual.md"}
)

func resolveHome(home ...string) string {
	if len(home) > 0 && home[0] != "" {
		return home[0]
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// DefaultRecordingsDir returns Movies/goscribe under given or current home.
func DefaultRecordingsDir(home ...string) string {
	return filepath.Join(resolveHome(home...), "Movies", "goscribe")
}

// StateFilePath returns .goscribe/state.json under given or current home.
func StateFilePath(home ...string) string {
	return filepath.Join(resolveHome(home...), ".goscribe", "state.json")
}

// OptimizeQueuePath returns .goscribe/optimize-queue.json under given or current home.
func OptimizeQueuePath(home ...string) string {
	return filepath.Join(resolveHome(home...), ".goscribe", "optimize-queue.json")
}

// OptimizeLogPath returns .goscribe/optimize-worker.log under given or current home.
func OptimizeLogPath(home ...string) string {
	return filepath.Join(resolveHome(home...), ".goscribe", "optimize-worker.log")
}

// Slugify lowercases, trims, collapses non-alphanumeric/mark unicode into hyphens, trims hyphens, caps at 50 chars.
func Slugify(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = nonLetterDigitMark.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	// ponytail: cap by unicode runes at 50; upgrade to grapheme clusters if needed.
	if len(runes) > 50 {
		runes = runes[:50]
	}
	return string(runes)
}

// TimestampedFilename formats date as 2006-01-02_15-04-05[_slug].ext.
func TimestampedFilename(date time.Time, name string, ext string) string {
	if ext == "" {
		ext = "mp4"
	}
	ext = strings.TrimPrefix(ext, ".")
	formatted := date.Format("2006-01-02_15-04-05")
	if name != "" {
		slug := Slugify(name)
		if slug != "" {
			return fmt.Sprintf("%s_%s.%s", formatted, slug, ext)
		}
	}
	return fmt.Sprintf("%s.%s", formatted, ext)
}

// WithMp4Extension replaces existing extension with .mp4.
func WithMp4Extension(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".mp4"
}

// WithAudioExtension returns path unchanged if .mp4 or .m4a, else replaces extension with .m4a.
func WithAudioExtension(path string) string {
	if strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".m4a") {
		return path
	}
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".m4a"
}

// WithRecordingExtension delegates to WithAudioExtension or WithMp4Extension.
func WithRecordingExtension(path string, audioOnly bool) string {
	if audioOnly {
		return WithAudioExtension(path)
	}
	return WithMp4Extension(path)
}

// UniqueBase resolves a unique base path, appending slug and counter suffix on conflict.
func UniqueBase(base string, slug string, isExplicit bool, exists func(string) bool) string {
	name := filepath.Base(base)
	if strings.HasSuffix(name, "_"+slug) {
		return base
	}
	if !isExplicit && timestampSlugRegex.MatchString(name) {
		return base
	}
	if exists == nil {
		exists = func(p string) bool {
			_, err := os.Stat(p)
			return err == nil
		}
	}
	dir := filepath.Dir(base)
	target := name + "_" + slug
	candidate := target
	counter := 1
	for exists(filepath.Join(dir, candidate+".mp4")) || exists(filepath.Join(dir, candidate+".m4a")) {
		candidate = fmt.Sprintf("%s-%d", target, counter)
		counter++
	}
	return filepath.Join(dir, candidate)
}

// RenameRecordingFiles renames recording artifacts from oldBase to newBase.
func RenameRecordingFiles(
	oldBase string,
	newBase string,
	exists func(string) bool,
	rename func(string, string) error,
) ([]string, error) {
	if oldBase == newBase {
		return nil, nil
	}
	if exists == nil {
		exists = func(p string) bool {
			_, err := os.Stat(p)
			return err == nil
		}
	}
	if rename == nil {
		rename = os.Rename
	}

	var renamed []string
	for _, ext := range recordingExtensions {
		oldPath := oldBase + ext
		newPath := newBase + ext
		if exists(oldPath) {
			if err := rename(oldPath, newPath); err != nil {
				return renamed, err
			}
			renamed = append(renamed, newPath)
		}
	}
	return renamed, nil
}
