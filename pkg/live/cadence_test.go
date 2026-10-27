package live

import (
	"testing"
)

func TestIsQuestionLine(t *testing.T) {
	questions := []string{
		"[01:00] them: is that right?",
		"[01:00] them: What is your biggest weakness",
		"[01:00] them: how did you scale it",
		"[01:00] them: Can you walk me through it",
		"[01:00] them: tell me about yourself",
		"[00:05] them: why did you choose that?",
		"[02:15] them: when will it release?",
		"[03:40] them: where is the config file?",
		"[04:10] them: who is working on it?",
		"[05:00] them: which option is better?",
		"[06:20] them: could you elaborate?",
		"[07:30] them: walk me through the architecture",
		"[08:00] them: describe your background",
		"[09:12] them: explain how this works",
	}
	for _, q := range questions {
		if !IsQuestionLine(q) {
			t.Errorf("expected IsQuestionLine(%q) to be true, got false", q)
		}
	}

	nonQuestions := []string{
		"[01:00] them: we shipped it last week",
		"[01:00] them: however we think so",
		"[01:00] me: what do you mean?",
		"[01:00] note: what about pricing?",
		"them: what do you think?",
		"[01:00] them:",
		"[01:00] them:    ",
		"[01:00] them: whatever happens",
	}
	for _, nq := range nonQuestions {
		if IsQuestionLine(nq) {
			t.Errorf("expected IsQuestionLine(%q) to be false, got true", nq)
		}
	}
}

func TestSuggestionCadenceTimerInterval(t *testing.T) {
	var currentClock int64
	now := func() int64 { return currentClock }
	opts := CadenceOptions{IntervalMs: 60000, RetryMs: 15000}
	cadence := NewSuggestionCadence(now, opts)

	// Not due before 60s of new speech
	cadence.TranscriptReached(59000)
	if cadence.Due() {
		t.Fatal("expected Due() to be false at 59s")
	}

	// Due once 60s of transcript accumulated
	cadence.TranscriptReached(60000)
	if !cadence.Due() {
		t.Fatal("expected Due() to be true at 60s")
	}

	// In flight: never due
	cadence.Begin()
	if cadence.Due() {
		t.Fatal("expected Due() to be false while in flight")
	}

	// Succeeded: measures from transcript time of last successful suggestion
	cadence.Succeeded()
	cadence.TranscriptReached(119000)
	if cadence.Due() {
		t.Fatal("expected Due() to be false before next 60s from mark")
	}
	cadence.TranscriptReached(120000)
	if !cadence.Due() {
		t.Fatal("expected Due() to be true after 60s since last mark")
	}

	// Failed: waits retry delay on wall clock, speech not marked as consumed
	cadence.Begin()
	currentClock = 1000
	cadence.Failed()

	currentClock = 15999
	if cadence.Due() {
		t.Fatal("expected Due() to be false during retry delay")
	}

	currentClock = 16000
	if !cadence.Due() {
		t.Fatal("expected Due() to be true after retry delay")
	}
}

func TestSuggestionCadenceQuestions(t *testing.T) {
	var currentClock int64 = 100000
	now := func() int64 { return currentClock }
	opts := CadenceOptions{IntervalMs: 60000, RetryMs: 15000}
	cadence := NewSuggestionCadence(now, opts)

	// Heard question makes it due before 60s
	cadence.TranscriptReached(10000)
	if cadence.Due() {
		t.Fatal("expected Due() to be false initially")
	}
	cadence.QuestionHeard()
	if !cadence.Due() {
		t.Fatal("expected Due() to be true after question heard")
	}

	// Begin and succeed consumes the question
	cadence.Begin()
	cadence.Succeeded()
	if cadence.Due() {
		t.Fatal("expected Due() to be false after question consumed")
	}

	// Debounced: heard again within 15s of previous call is not due until 15s elapsed
	currentClock = 105000
	cadence.QuestionHeard()
	if cadence.Due() {
		t.Fatal("expected Due() to be false within 15s debounce gap")
	}

	currentClock = 115000
	if !cadence.Due() {
		t.Fatal("expected Due() to be true after 15s debounce gap")
	}
}
