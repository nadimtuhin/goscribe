package optimize

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func TestEnqueueJobConcurrentDoesNotLoseJobs(t *testing.T) {
	qPath := filepath.Join(t.TempDir(), "queue.json")
	const n = 30
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EnqueueJob(qPath, "in.mp4", "out.mp4", "", 0, 0)
		}()
	}
	wg.Wait()
	if got := len(ReadQueue(qPath).Jobs); got != n {
		t.Fatalf("expected %d jobs, got %d (lost updates)", n, got)
	}
}

// Cancelling the worker mid-encode must not mark the job failed; it should go
// back to queued so the next worker picks it up.
func TestProcessQueueCancelRequeuesJob(t *testing.T) {
	qPath := filepath.Join(t.TempDir(), "queue.json")
	job := EnqueueJob(qPath, "in.mp4", "out.mp4", "", 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	fake := runner.NewFakeRunner()
	fake.RunFn = func(ctx context.Context, cmd []string) (string, string, int, error) {
		cancel()
		return "", "", 1, ctx.Err()
	}
	_, err := ProcessQueue(ctx, qPath, fake, nil)
	if err == nil {
		t.Fatalf("expected context error")
	}
	for _, j := range ReadQueue(qPath).Jobs {
		if j.ID == job.ID && j.Status != StatusQueued {
			t.Fatalf("expected job requeued after cancel, got %s (%q)", j.Status, j.Error)
		}
	}
}
