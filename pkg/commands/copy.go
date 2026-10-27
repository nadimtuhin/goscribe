package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/recorder"
)

func CopyCommand(args []string, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("copy", flag.ContinueOnError)
	fs.SetOutput(errOut)
	transcribeFlag := fs.Bool("transcribe", false, "Copy transcript instead of notes")
	if err := fs.Parse(args); err != nil {
		return 1
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
		fmt.Fprintln(errOut, "no recordings found")
		return 1
	}

	base := target
	for _, ext := range []string{".mp4", ".m4a", ".transcript.md", ".notes.md"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}

	notesPath := base + ".notes.md"
	txPath := base + ".transcript.md"

	var chosen string
	if *transcribeFlag {
		if _, err := os.Stat(txPath); err == nil {
			chosen = txPath
		}
	} else {
		if _, err := os.Stat(notesPath); err == nil {
			chosen = notesPath
		} else if _, err := os.Stat(txPath); err == nil {
			chosen = txPath
		}
	}

	if chosen == "" {
		fmt.Fprintf(errOut, "no notes or transcript found for %s\n", target)
		return 1
	}

	content, err := os.ReadFile(chosen)
	if err != nil {
		fmt.Fprintf(errOut, "failed to read %s: %v\n", chosen, err)
		return 1
	}

	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(string(content))
	_ = cmd.Run()

	fmt.Fprintf(out, "copied to clipboard: %s\n", chosen)
	return 0
}
