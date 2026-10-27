package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/optimize"
	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/recorder"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func OptimizeCommand(args []string, r runner.Runner, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("optimize", flag.ContinueOnError)
	fs.SetOutput(errOut)

	res := fs.String("resolution", "", "Target resolution (e.g. 720p, 1080p, 4k)")
	fs.StringVar(res, "res", "", "Target resolution alias")
	crf := fs.Int("crf", 24, "x264 Constant Rate Factor")
	outFile := fs.String("out", "", "Custom output file path")
	wait := fs.Bool("wait", false, "Wait for optimization to complete")
	fs.BoolVar(wait, "w", false, "Wait alias")
	statusFlag := fs.Bool("status", false, "Show queue status")
	clearFlag := fs.Bool("clear", false, "Clear completed/failed jobs")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	qPath := paths.OptimizeQueuePath()

	if *clearFlag {
		optimize.ClearQueue(qPath)
		fmt.Fprintln(out, "queue cleared")
		return 0
	}

	if *statusFlag {
		return QueueCommand([]string{}, out, errOut)
	}

	target := ""
	if len(fs.Args()) > 0 {
		target = fs.Args()[0]
	} else {
		files, _ := recorder.List(paths.DefaultRecordingsDir())
		if len(files) > 0 {
			target = files[0].Path
		}
	}

	if target == "" {
		fmt.Fprintln(errOut, "no recordings found to optimize")
		return 1
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(errOut, "target file not found: %s\n", target)
		return 1
	}

	dest := *outFile
	if dest == "" {
		base := strings.TrimSuffix(target, filepath.Ext(target))
		suffix := "_opt"
		if *res != "" {
			suffix = fmt.Sprintf("_%s", *res)
		}
		dest = fmt.Sprintf("%s%s.mp4", base, suffix)
	}

	job := optimize.EnqueueJob(qPath, target, dest, *res, *crf, info.Size())
	fmt.Fprintf(out, "enqueued job %s: %s -> %s\n", job.ID, target, dest)

	if *wait {
		fmt.Fprintln(out, "waiting for optimization to finish...")
		_, err := optimize.ProcessQueue(context.Background(), qPath, r, func(s string) {
			fmt.Fprintln(out, s)
		})
		if err != nil {
			fmt.Fprintf(errOut, "error processing queue: %v\n", err)
			return 1
		}
	}

	return 0
}
