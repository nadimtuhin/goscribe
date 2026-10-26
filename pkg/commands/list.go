package commands

import (
	"fmt"
	"io"

	"github.com/nadimtuhin/goscribe/pkg/recorder"
)

// ListCommand writes completed recordings to out, newest first.
func ListCommand(recordingsDir string, out io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	files, err := recorder.List(recordingsDir)
	if err != nil {
		fmt.Fprintf(out, "error listing recordings: %v\n", err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(out, "no recordings yet")
		return 0
	}
	for _, f := range files {
		fmt.Fprintf(out, "%s  %9s  %s\n", f.Modified.Format("01/02/2006, 3:04:05 PM"), recorder.FormatSize(f.Bytes), f.Name)
	}
	return 0
}
