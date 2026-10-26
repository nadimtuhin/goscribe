package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/nadimtuhin/goscribe/pkg/capture"
	"github.com/nadimtuhin/goscribe/pkg/ffmpeg"
	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/recorder"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func DevicesCommand(args []string, r runner.Runner, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("devices", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonFlag := fs.Bool("json", false, "Output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	ffmpeg.EnsureToolPath()
	binDir := filepath.Dir(paths.StateFilePath())
	helper, err := recorder.EnsureHelper(r, "native/capture.swift", filepath.Join(binDir, "capture"))
	if err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		return 1
	}

	report, err := capture.ListDevices(context.Background(), r, helper)
	if err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		return 1
	}

	if *jsonFlag {
		data, _ := json.Marshal(report)
		fmt.Fprintf(out, "%s\n", data)
	} else {
		fmt.Fprintln(out, capture.FormatDevicesText(report))
	}
	return 0
}
