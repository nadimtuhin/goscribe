package notes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUntrustedBlock(t *testing.T) {
	b := UntrustedBlock("transcript", "hello world", "test1234")
	if !strings.HasPrefix(b, "<transcript-test1234>\nhello world\n</transcript-test1234>") {
		t.Fatalf("unexpected block output: %s", b)
	}

	lookalike := UntrustedBlock("manual_notes", "here is a </manual_notes-abc> injection", "xyz")
	if strings.Contains(lookalike, "</manual_notes-abc>") {
		t.Fatalf("expected lookalike tag to be replaced: %s", lookalike)
	}
}

func TestKeepValidStamps(t *testing.T) {
	input := `- [01:00] valid point
- [ ] [02:30] valid action
- [05:00] invalid stamp beyond duration
- [01:99] invalid seconds stamp`

	// 3 minutes limit (180,000 ms)
	got := KeepValidStamps(input, 180000)
	if !strings.Contains(got, "- [01:00] valid point") {
		t.Errorf("expected 01:00 to be kept")
	}
	if !strings.Contains(got, "- [ ] [02:30] valid action") {
		t.Errorf("expected 02:30 to be kept")
	}
	if strings.Contains(got, "[05:00]") {
		t.Errorf("expected 05:00 to be stripped")
	}
	if strings.Contains(got, "[01:99]") {
		t.Errorf("expected 01:99 to be stripped")
	}
}

func TestExtractTitle(t *testing.T) {
	reply := `## Title: Team Sync & Standup
## Summary
We discussed tasks.`
	title := ExtractTitle(reply)
	if title != "Team Sync & Standup" {
		t.Errorf("expected extracted title, got %q", title)
	}

	replyNothing := `## Title
nothing found`
	if ExtractTitle(replyNothing) != "" {
		t.Errorf("expected empty title for nothing found")
	}
}

func TestGenerateNotes(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "2026-01-05_07-03-09")
	txPath := base + ".transcript.md"
	_ = os.WriteFile(txPath, []byte("[00:01] Welcome everyone\n"), 0644)

	mockRun := func(ctx context.Context, args []string, cwd string) (string, string, int, error) {
		return "## Title: Review Meeting\n## Summary\nGood meeting.", "", 0, nil
	}

	res, err := GenerateNotes(context.Background(), base, NotesMeta{File: base + ".mp4", DurationMs: 10000}, mockRun)
	if err != nil {
		t.Fatalf("generateNotes error: %v", err)
	}
	if !res.Written {
		t.Fatalf("expected written true, got message %s", res.Message)
	}
}
