package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/recorder"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func StopCommand(args []string, statePath string, r runner.Runner, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(errOut)

	notifyFlag := fs.Bool("notify", false, "Post system notification")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	res, err := recorder.Stop(statePath, r)
	if err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		return 1
	}

	if res.MixError != "" {
		fmt.Fprintf(errOut, "failed to mix audio: %s\n", res.MixError)
		if len(res.KeptParts) > 0 {
			fmt.Fprintf(errOut, "kept raw parts: %s\n", strings.Join(res.KeptParts, ", "))
		}
		return 1
	}

	if res.Forced {
		fmt.Fprintln(errOut, "warning: recording process had to be forcefully killed, file may be truncated")
	}

	durStr := recorder.FormatDuration(res.DurationMs)
	sizeStr := recorder.FormatSize(res.Bytes)
	fmt.Fprintf(out, "stopped %s (%s, %s)\n", res.File, durStr, sizeStr)

	if *notifyFlag {
		msg := fmt.Sprintf("Saved %s (%s, %s)", res.File, durStr, sizeStr)
		_, _ = r.SpawnDetached("osascript", []string{"-e", fmt.Sprintf(`display notification %q with title "Goscribe"`, msg)}, "/dev/null")
	}

	return 0
}
