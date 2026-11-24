package runner

import (
	"path/filepath"
	"testing"
	"time"
)

// A detached child that exits must be reaped, otherwise it lingers as a
// zombie and IsAlive (kill -0) keeps reporting it alive, which defeats the
// recorder's "exited on startup" check.
func TestRealRunner_SpawnDetachedReapsExitedChild(t *testing.T) {
	r := &RealRunner{}
	pid, err := r.SpawnDetached("true", nil, filepath.Join(t.TempDir(), "log"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !r.IsAlive(pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid %d exited but IsAlive still reports true (zombie not reaped)", pid)
}
