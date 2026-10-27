package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/notes"
	"github.com/nadimtuhin/goscribe/pkg/transcription"
)

func NotesCommand(args []string, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("notes", flag.ContinueOnError)
	fs.SetOutput(errOut)
	lang := fs.String("lang", "", "Conversation language")
	name := fs.String("name", "", "Conversation name")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if len(fs.Args()) == 0 {
		fmt.Fprintln(errOut, "notes requires a recording or transcript path")
		return 1
	}

	target := fs.Args()[0]
	parsedLang := ""
	if *lang != "" {
		l, err := transcription.ParseLanguage(*lang)
		if err != nil {
			fmt.Fprintf(errOut, "%v\n", err)
			return 1
		}
		parsedLang = l
	}

	base := target
	for _, ext := range []string{".mp4", ".m4a", ".transcript.md"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}

	meta := notes.NotesMeta{
		File: target,
		Lang: parsedLang,
		Name: *name,
	}

	fmt.Fprintln(out, "notes: sending the transcript to Hermes (it may reach a cloud model)")
	realHermes := func(ctx context.Context, hArgs []string, cwd string) (string, string, int, error) {
		bin, err := exec.LookPath("hermes")
		if err != nil {
			home, _ := os.UserHomeDir()
			bin = filepath.Join(home, ".local", "bin", "hermes")
		}
		cmd := exec.CommandContext(ctx, bin, hArgs...)
		cmd.Dir = cwd
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		code := 0
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		return stdout.String(), stderr.String(), code, err
	}

	res, err := notes.GenerateNotes(context.Background(), base, meta, realHermes)
	if err != nil {
		fmt.Fprintf(errOut, "notes: %v\n", err)
		return 1
	}
	fmt.Fprintln(out, res.Message)
	return 0
}
