package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/commands"
	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

const helpText = `goscribe - local screen recorder

  goscribe start [--name <name>] [--out file.mp4] [--audio-only] [--screen N] [--fps 30] [--mic N] [--no-mic] [--no-system-audio]
                   [--window <id|name>] [--area x,y,w,h] [--display N] [--engine sck|ffmpeg]
                   [--transcribe] [--live] [--notes] [--draw] [-d|--detach] [--notify]
                   [--lang <language>]   (conversation language: english, bangla, hindi, urdu, arabic, spanish, ... or multiple like "en bn", "en,bn"; default en; Bangla uses ggml-large-v3-turbo-q5_0.bin, other non-English ggml-small.bin)
                   [--role candidate|interviewer|meeting] [--context <file>]...   (with --live; type "?question" in the live view to ask Hermes)
                   (--live opens the live view here; quitting it stops the recording. -d runs in the background)
  goscribe live        (reattach the live view to a recording started with --live -d)
  goscribe stop        [--notify]
  goscribe open        [target] [--notes] [--transcribe] [-R|--reveal]
  goscribe copy        [target] [--transcribe]
  goscribe last        [--json]
  goscribe status      [--json]
  goscribe list
  goscribe devices     [--json]
  goscribe captions
  goscribe notes <mp4|transcript.md> [--lang <language>] [--name <name>]
  goscribe optimize [target] [--resolution <res>] [--crf N] [--out file.mp4] [--wait] [--status] [--clear]
  goscribe queue    [--clear] [--json]`

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, helpText)
		return 1
	}

	cmd := args[0]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		fmt.Fprintln(stdout, helpText)
		return 0
	}

	for _, arg := range args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(stdout, helpText)
			return 0
		}
	}

	statePath := paths.StateFilePath()
	r := runner.DefaultRunner

	switch cmd {
	case "start":
		return commands.StartCommand(args[1:], statePath, r, stdout, stderr)

	case "stop":
		return commands.StopCommand(args[1:], statePath, r, stdout, stderr)

	case "status":
		jsonOutput := false
		for _, arg := range args[1:] {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		return commands.StatusCommand(statePath, r, jsonOutput, time.Now, stdout)

	case "list":
		return commands.ListCommand(paths.DefaultRecordingsDir(), stdout)

	case "devices":
		return commands.DevicesCommand(args[1:], r, stdout, stderr)

	default:
		fmt.Fprintf(stderr, "goscribe %s: not implemented\n", cmd)
		return 1
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
