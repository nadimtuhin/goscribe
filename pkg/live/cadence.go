package live

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

const questionGapMs int64 = 15000

var (
	questionLineRegex   = regexp.MustCompile(`^\[\d+:\d{2}\] them:\s*(.*)$`)
	questionOpenerRegex = regexp.MustCompile(`(?i)^(what|how|why|when|where|who|which|can you|could you|tell me|walk me through|describe|explain)\b`)
)

// IsQuestionLine returns true if line is spoken by "them" and either ends with "?" or begins with a question phrase.
func IsQuestionLine(line string) bool {
	matches := questionLineRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return false
	}
	text := strings.TrimSpace(matches[1])
	if text == "" {
		return false
	}
	return strings.HasSuffix(text, "?") || questionOpenerRegex.MatchString(text)
}

type CadenceOptions struct {
	IntervalMs int64
	RetryMs    int64
}

// SuggestionCadence decides when suggestions are due based on elapsed transcript speech and heard questions.
type SuggestionCadence struct {
	mu              sync.Mutex
	now             func() int64
	options         CadenceOptions
	latestMs        int64
	markMs          int64
	callMarkMs      int64
	inFlight        bool
	retryAtMs       int64
	lastBeginAtMs   int64
	hasBegun        bool
	questionPending bool
}

func NewSuggestionCadence(now func() int64, opts CadenceOptions) *SuggestionCadence {
	if now == nil {
		now = func() int64 {
			return time.Now().UnixMilli()
		}
	}
	return &SuggestionCadence{
		now:     now,
		options: opts,
	}
}

func (c *SuggestionCadence) TranscriptReached(ms int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ms > c.latestMs {
		c.latestMs = ms
	}
}

func (c *SuggestionCadence) QuestionHeard() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.questionPending = true
}

func (c *SuggestionCadence) Due() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.inFlight || now < c.retryAtMs {
		return false
	}
	timerDue := c.latestMs-c.markMs >= c.options.IntervalMs
	questionDue := c.questionPending && (!c.hasBegun || now-c.lastBeginAtMs >= questionGapMs)
	return timerDue || questionDue
}

func (c *SuggestionCadence) Begin() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight = true
	c.callMarkMs = c.latestMs
	c.lastBeginAtMs = c.now()
	c.hasBegun = true
	c.questionPending = false
}

func (c *SuggestionCadence) Succeeded() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight = false
	c.markMs = c.callMarkMs
}

func (c *SuggestionCadence) Failed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight = false
	c.retryAtMs = c.now() + c.options.RetryMs
}
