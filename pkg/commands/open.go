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

func OpenCommand(args []string, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(errOut)
	notesFlag := fs.Bool("notes", false, "Open notes file")
	txFlag := fs.Bool("transcribe", false, "Open transcript file")
	revealFlag := fs.Bool("R", false, "Reveal in Finder")
	fs.BoolVar(revealFlag, "reveal", false, "Reveal in Finder")

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

	if target == "" || !fileExists(target) {
		fmt.Fprintln(errOut, "no recordings found")
		return 1
	}

	base := target
	for _, ext := range []string{".mp4", ".m4a"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}

	fileToOpen := target
	if *notesFlag {
		notesPath := base + ".notes.md"
		if fileExists(notesPath) {
			fileToOpen = notesPath
		}
	} else if *txFlag {
		txPath := base + ".transcript.md"
		if fileExists(txPath) {
			fileToOpen = txPath
		}
	}

	openArgs := []string{}
	if *revealFlag {
		openArgs = append(openArgs, "-R")
	}
	openArgs = append(openArgs, fileToOpen)

	cmd := exec.Command("open", openArgs...)
	_ = cmd.Run()

	fmt.Fprintf(out, "opened %s\n", fileToOpen)
	return 0
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
