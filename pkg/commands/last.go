package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nadimtuhin/goscribe/pkg/paths"
	"github.com/nadimtuhin/goscribe/pkg/recorder"
)

func summaryOf(notes string) string {
	idx := strings.Index(notes, "## Summary")
	if idx == -1 {
		return ""
	}
	rest := notes[idx+len("## Summary"):]
	end := strings.Index(rest, "\n## ")
	if end != -1 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func LastCommand(args []string, out, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	fs := flag.NewFlagSet("last", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonFlag := fs.Bool("json", false, "Output JSON format")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	files, err := recorder.List(paths.DefaultRecordingsDir())
	if err != nil || len(files) == 0 {
		fmt.Fprintln(errOut, "no recordings yet")
		return 1
	}

	newest := files[0]
	base := strings.TrimSuffix(strings.TrimSuffix(newest.Path, ".mp4"), ".m4a")
	notesPath := base + ".notes.md"
	txPath := base + ".transcript.md"

	var notesPtr, txPtr *string
	if _, err := os.Stat(notesPath); err == nil {
		notesPtr = &notesPath
	}
	if _, err := os.Stat(txPath); err == nil {
		txPtr = &txPath
	}

	if *jsonFlag {
		type lastJSON struct {
			File       string  `json:"file"`
			Name       string  `json:"name"`
			Bytes      int64   `json:"bytes"`
			Modified   string  `json:"modified"`
			Notes      *string `json:"notes"`
			Transcript *string `json:"transcript"`
		}
		data, _ := json.Marshal(lastJSON{
			File:       newest.Path,
			Name:       filepath.Base(newest.Path),
			Bytes:      newest.Bytes,
			Modified:   newest.Modified.UTC().Format("2006-01-02T15:04:05.000Z"),
			Notes:      notesPtr,
			Transcript: txPtr,
		})
		fmt.Fprintf(out, "%s\n", data)
		return 0
	}

	fmt.Fprintf(out, "%s  %s  %s\n", filepath.Base(newest.Path), recorder.FormatSize(newest.Bytes), newest.Modified.Local().Format("01/02/2006, 3:04:05 PM"))
	fmt.Fprintf(out, "video: %s\n", newest.Path)
	if notesPtr != nil {
		fmt.Fprintf(out, "notes: %s\n", *notesPtr)
	}
	if txPtr != nil {
		fmt.Fprintf(out, "transcript: %s\n", *txPtr)
	}
	if notesPtr != nil {
		if b, err := os.ReadFile(*notesPtr); err == nil {
			if s := summaryOf(string(b)); s != "" {
				fmt.Fprintf(out, "\n%s\n", s)
			}
		}
	}
	return 0
}
