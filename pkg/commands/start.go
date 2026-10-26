package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/nadimtuhin/goscribe/pkg/recorder"
	"github.com/nadimtuhin/goscribe/pkg/runner"
)

func StartCommand(args []string, statePath string, r runner.Runner, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(errOut)

	name := fs.String("name", "", "Name of the recording")
	outFile := fs.String("out", "", "Output file path")
	audioOnly := fs.Bool("audio-only", false, "Record audio only")
	screen := fs.Int("screen", -1, "Screen index")
	fps := fs.Int("fps", 30, "Recording FPS")
	mic := fs.Int("mic", -1, "Microphone index")
	noMic := fs.Bool("no-mic", false, "Disable microphone")
	noSysAudio := fs.Bool("no-system-audio", false, "Disable system audio")
	engine := fs.String("engine", "", "Recording engine (ffmpeg or sck)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts := recorder.StartOptions{
		Out:           *outFile,
		Name:          *name,
		AudioOnly:     *audioOnly,
		Fps:           *fps,
		NoMic:         *noMic,
		NoSystemAudio: *noSysAudio,
		Engine:        *engine,
	}
	if *screen >= 0 {
		opts.Screen = screen
	}
	if *mic >= 0 {
		opts.Mic = mic
	}

	st, err := recorder.Start(opts, statePath, r)
	if err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		return 1
	}

	fmt.Fprintf(out, "recording to %s\n", st.File)
	return 0
}
