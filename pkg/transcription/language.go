package transcription

import (
	"fmt"
	"strings"
)

type LanguageMeta struct {
	Name    string
	Aliases []string
}

var Languages = map[string]LanguageMeta{
	"en": {Name: "English", Aliases: []string{"english"}},
	"bn": {Name: "Bangla", Aliases: []string{"bangla", "bengali"}},
	"hi": {Name: "Hindi", Aliases: []string{"hindi"}},
	"ur": {Name: "Urdu", Aliases: []string{"urdu"}},
	"ar": {Name: "Arabic", Aliases: []string{"arabic"}},
	"es": {Name: "Spanish", Aliases: []string{"spanish"}},
}

const DefaultLanguage = "en"

func IsLanguageToken(raw string) bool {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "auto" {
		return true
	}
	for code, meta := range Languages {
		if code == key {
			return true
		}
		for _, a := range meta.Aliases {
			if a == key {
				return true
			}
		}
	}
	return false
}

func parseSingleLanguage(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "auto" {
		return "auto", nil
	}
	for code, meta := range Languages {
		if code == key {
			return code, nil
		}
		for _, a := range meta.Aliases {
			if a == key {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("--lang expects a language name or code such as english, bangla, hindi, urdu, arabic, spanish (or en, bn, hi) or auto, got %q", raw)
}

func ParseLanguage(input string) (string, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(parts) == 0 {
		return DefaultLanguage, nil
	}
	var codes []string
	seen := make(map[string]bool)
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		c, err := parseSingleLanguage(trimmed)
		if err != nil {
			return "", err
		}
		if c == "auto" {
			return "auto", nil
		}
		if !seen[c] {
			seen[c] = true
			codes = append(codes, c)
		}
	}
	if len(codes) == 0 {
		return DefaultLanguage, nil
	}
	return strings.Join(codes, ","), nil
}

func LanguageInstruction(code string) string {
	if code == "auto" {
		return "Reply in the language of the conversation."
	}
	codes := strings.Split(code, ",")
	if len(codes) == 1 {
		name := codes[0]
		if meta, ok := Languages[codes[0]]; ok {
			name = meta.Name
		}
		return fmt.Sprintf("The conversation is in %s. Reply in %s.", name, name)
	}
	var names []string
	for _, c := range codes {
		name := c
		if meta, ok := Languages[c]; ok {
			name = meta.Name
		}
		names = append(names, name)
	}
	joined := strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	return fmt.Sprintf("The conversation is bilingual in %s. Reply in the languages of the conversation.", joined)
}
