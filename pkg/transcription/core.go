package transcription

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type WhisperSegment struct {
	From int64  `json:"from"`
	To   int64  `json:"to"`
	Text string `json:"text"`
}

func FormatTimestamp(ms int64) string {
	totalSeconds := ms / 1000
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	return fmt.Sprintf("%02d:%02d", totalSeconds/60, totalSeconds%60)
}

var hallucinations = map[string]bool{
	"you":        true,
	"thank you":  true,
	"thanks":     true,
	"bye":        true,
}

var nonSpeechBracketRegex = regexp.MustCompile(`^[\[\(].*[\]\)]$`)

func cleanPunct(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func IsNonSpeech(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return true
	}
	if nonSpeechBracketRegex.MatchString(t) {
		return true
	}
	clean := strings.ToLower(cleanPunct(t))
	return hallucinations[clean]
}

func DevanagariToBengali(text string) string {
	var out strings.Builder
	runes := []rune(text)
	for _, r := range runes {
		if r == 'व' {
			out.WriteRune('ব')
		} else if r == 0x0964 || r == 0x0965 { // danda, double danda
			out.WriteRune(r)
		} else if r >= 0x0901 && r <= 0x096F {
			bcp := r + 0x80
			if bcp >= 0x0981 && bcp <= 0x09FA {
				out.WriteRune(bcp)
			} else {
				out.WriteRune(r)
			}
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func SegmentsToMarkdown(segments []WhisperSegment, lang string) string {
	isBangla := strings.Contains(","+lang+",", ",bn,")
	var sb strings.Builder
	for _, seg := range segments {
		trimmed := strings.TrimSpace(seg.Text)
		text := trimmed
		if isBangla {
			text = DevanagariToBengali(trimmed)
		}
		if !IsNonSpeech(text) {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", FormatTimestamp(seg.From), text))
		}
	}
	return sb.String()
}

func BuildExtractAudioArgs(input string, wav string) []string {
	return []string{"-y", "-i", input, "-vn", "-ac", "1", "-ar", "16000", wav}
}

func WhisperCliLanguage(lang string) string {
	codes := strings.Split(lang, ",")
	for _, c := range codes {
		if c == "bn" {
			return "bn"
		}
	}
	if len(codes) == 1 {
		return codes[0]
	}
	return "auto"
}

func BuildWhisperArgs(model, wav, prefix, lang, prompt string) []string {
	codes := strings.Split(lang, ",")
	hasBn := false
	for _, c := range codes {
		if c == "bn" {
			hasBn = true
			break
		}
	}
	cliLang := WhisperCliLanguage(lang)
	p := prompt
	if p == "" {
		if lang == "bn" {
			p = "বাংলা ভাষা।"
		} else if hasBn {
			p = "বাংলা এবং English মিশ্রিত আলোচনা ও মিটিং।"
		}
	}
	args := []string{"-m", model, "-f", wav, "-l", cliLang, "-oj", "-of", prefix}
	if p != "" {
		args = append(args, "--prompt", p)
	}
	if hasBn {
		args = append(args, "--suppress-nst")
	}
	return args
}

func WhisperModelPath(env map[string]string, lang string) string {
	if custom, ok := env["LOCALSCRIBE_WHISPER_MODEL"]; ok && custom != "" {
		return custom
	}
	home, _ := os.UserHomeDir()
	modelsDir := filepath.Join(home, ".hermes", "models", "whisper")
	if lang == "" {
		lang = DefaultLanguage
	}
	codes := strings.Split(lang, ",")
	for _, c := range codes {
		if c == "bn" {
			return filepath.Join(modelsDir, "ggml-large-v3-turbo-q5_0.bin")
		}
	}
	if len(codes) == 1 && codes[0] == "en" {
		return filepath.Join(modelsDir, "ggml-base.en.bin")
	}
	return filepath.Join(modelsDir, "ggml-small.bin")
}

func RequireWhisperModel(env map[string]string, lang string, exists func(string) bool) (string, error) {
	model := WhisperModelPath(env, lang)
	if exists(model) {
		return model, nil
	}
	modelURL := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"
	fix := fmt.Sprintf(". Download it with: curl -L -o %s %s/%s", model, modelURL, filepath.Base(model))
	if custom, ok := env["LOCALSCRIBE_WHISPER_MODEL"]; ok && custom != "" {
		fix = " (set LOCALSCRIBE_WHISPER_MODEL)"
	}
	return "", fmt.Errorf("whisper model not found: %s%s", model, fix)
}
