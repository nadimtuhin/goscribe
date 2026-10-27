package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/nadimtuhin/goscribe/pkg/optimize"
	"github.com/nadimtuhin/goscribe/pkg/paths"
)

func QueueCommand(args []string, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("queue", flag.ContinueOnError)
	fs.SetOutput(errOut)
	clearFlag := fs.Bool("clear", false, "Clear completed and failed jobs")
	jsonFlag := fs.Bool("json", false, "Output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	qPath := paths.OptimizeQueuePath()
	if *clearFlag {
		optimize.ClearQueue(qPath)
		if *jsonFlag {
			fmt.Fprintln(out, `{"cleared":true}`)
		} else {
			fmt.Fprintln(out, "queue cleared")
		}
		return 0
	}

	data := optimize.ReadQueue(qPath)
	if *jsonFlag {
		b, _ := json.Marshal(data)
		fmt.Fprintf(out, "%s\n", b)
		return 0
	}

	if len(data.Jobs) == 0 {
		fmt.Fprintln(out, "queue is empty")
		return 0
	}

	for _, j := range data.Jobs {
		fmt.Fprintf(out, "[%s] %s -> %s (%s)\n", j.ID, j.Input, j.Output, j.Status)
	}
	return 0
}
