package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/recorder"
	"github.com/nadimtuhin/goscribe/pkg/runner"
	"github.com/nadimtuhin/goscribe/pkg/state"
)

type statusInactiveJSON struct {
	Recording bool `json:"recording"`
}

type statusActiveJSON struct {
	Recording bool   `json:"recording"`
	File      string `json:"file"`
	StartedAt string `json:"startedAt"`
	ElapsedMs int64  `json:"elapsedMs"`
}

// StatusCommand writes recording status and returns 0 if active, 1 if inactive.
func StatusCommand(statePath string, r runner.Runner, jsonOutput bool, now func() time.Time, out io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	st := state.ActiveRecording(statePath, r)
	if st == nil {
		if jsonOutput {
			data, _ := json.Marshal(statusInactiveJSON{Recording: false})
			fmt.Fprintf(out, "%s\n", data)
		} else {
			fmt.Fprintln(out, "not recording")
		}
		return 1
	}

	if now == nil {
		now = time.Now
	}

	startedAt, err := time.Parse(time.RFC3339, st.StartedAt)
	if err != nil {
		startedAt = now()
	}

	elapsedMs := now().Sub(startedAt).Milliseconds()
	if elapsedMs < 0 {
		elapsedMs = 0
	}

	if jsonOutput {
		data, _ := json.Marshal(statusActiveJSON{
			Recording: true,
			File:      st.File,
			StartedAt: st.StartedAt,
			ElapsedMs: elapsedMs,
		})
		fmt.Fprintf(out, "%s\n", data)
	} else {
		fmt.Fprintf(out, "recording %s (%s)\n", st.File, recorder.FormatDuration(elapsedMs))
	}
	return 0
}
