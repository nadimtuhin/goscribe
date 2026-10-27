package commands

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

const captionsPollInterval = 500 * time.Millisecond

// CaptionsCommand tails the live captions file of the active recording until stopped or interrupted.
func CaptionsCommand(statePath string, r runner.Runner, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if r == nil {
		r = runner.DefaultRunner
	}

	st := state.ActiveRecording(statePath, r)
	if st == nil {
		fmt.Fprintln(errOut, "not recording")
		return 1
	}

	base := strings.TrimSuffix(st.File, filepath.Ext(st.File))
	liveFile := base + ".live.txt"

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	var offset int
	drain := func() {
		data, err := os.ReadFile(liveFile)
		if err != nil {
			return
		}
		if len(data) > offset {
			_, _ = out.Write(data[offset:])
			offset = len(data)
		}
	}

	drain()

	for state.ActiveRecording(statePath, r) != nil {
		select {
		case <-sigCh:
			drain()
			return 0
		default:
		}
		r.Sleep(captionsPollInterval)
		drain()
	}

	drain()
	return 0
}
