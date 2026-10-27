package optimize

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func TestParseResolution(t *testing.T) {
	tests := []struct {
		input  string
		wantW  int
		wantH  int
		hasErr bool
	}{
		{"720p", 0, 720, false},
		{"1080", 0, 1080, false},
		{"4k", 0, 2160, false},
		{"1920x1080", 1920, 1080, false},
		{"1280:720", 1280, 720, false},
		{"invalid", 0, 0, true},
	}
	for _, tt := range tests {
		w, h, err := ParseResolution(tt.input)
		if (err != nil) != tt.hasErr {
			t.Errorf("ParseResolution(%q) err = %v, wantErr = %v", tt.input, err, tt.hasErr)
		}
		if w != tt.wantW || h != tt.wantH {
			t.Errorf("ParseResolution(%q) = (%d, %d), want (%d, %d)", tt.input, w, h, tt.wantW, tt.wantH)
		}
	}
}

func TestBuildScaleFilter(t *testing.T) {
	f, _ := BuildScaleFilter("720p")
	if f != "scale=-2:720" {
		t.Errorf("got %q, want scale=-2:720", f)
	}

	f2, _ := BuildScaleFilter("1920x1080")
	if !strings.Contains(f2, "force_original_aspect_ratio=decrease") {
		t.Errorf("unexpected custom dimension filter: %q", f2)
	}
}

func TestQueuePersistence(t *testing.T) {
	tmp := t.TempDir()
	qPath := filepath.Join(tmp, "queue.json")

	job := EnqueueJob(qPath, "in.mp4", "out.mp4", "720p", 24, 1000)
	if job.Status != StatusQueued {
		t.Fatalf("expected queued job, got %s", job.Status)
	}

	next := GetNextQueuedJob(qPath)
	if next == nil || next.ID != job.ID {
		t.Fatalf("expected next queued job %s, got %v", job.ID, next)
	}

	UpdateJob(qPath, job.ID, func(j *OptimizeJob) {
		j.Status = StatusProcessing
	})

	RecoverStaleJobs(qPath)
	data := ReadQueue(qPath)
	if len(data.Jobs) != 1 || data.Jobs[0].Status != StatusQueued {
		t.Fatalf("expected job recovered to queued, got %v", data.Jobs)
	}

	UpdateJob(qPath, job.ID, func(j *OptimizeJob) {
		j.Status = StatusCompleted
	})
	ClearQueue(qPath)
	data = ReadQueue(qPath)
	if len(data.Jobs) != 0 {
		t.Fatalf("expected completed job cleared, got %d jobs", len(data.Jobs))
	}
}

func TestProcessQueue(t *testing.T) {
	tmp := t.TempDir()
	qPath := filepath.Join(tmp, "queue.json")
	job := EnqueueJob(qPath, "in.mp4", "out.mp4", "720p", 24, 1000)

	fake := runner.NewFakeRunner()
	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		return "", "", 0, nil
	}

	count, err := ProcessQueue(context.Background(), qPath, fake, nil)
	if err != nil {
		t.Fatalf("ProcessQueue err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 job processed, got %d", count)
	}

	data := ReadQueue(qPath)
	if len(data.Jobs) != 1 || data.Jobs[0].ID != job.ID || data.Jobs[0].Status != StatusCompleted {
		t.Fatalf("expected job completed, got %v", data.Jobs)
	}
}
