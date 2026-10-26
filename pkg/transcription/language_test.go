package transcription

import (
	"testing"
)

func TestIsLanguageToken(t *testing.T) {
	for _, valid := range []string{"en", "english", "bn", "bangla", "bengali", "hi", "hindi", "auto"} {
		if !IsLanguageToken(valid) {
			t.Errorf("expected %q to be valid language token", valid)
		}
	}
	for _, invalid := range []string{"foo", "bar", "123", ""} {
		if IsLanguageToken(invalid) {
			t.Errorf("expected %q to be invalid language token", invalid)
		}
	}
}

func TestParseLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   bool
	}{
		{"", "en", false},
		{"english", "en", false},
		{"EN", "en", false},
		{"Bangla", "bn", false},
		{"en bn", "en,bn", false},
		{"en,bn", "en,bn", false},
		{"auto", "auto", false},
		{"klingon", "", true},
	}
	for _, tt := range tests {
		got, err := ParseLanguage(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseLanguage(%q) err = %v, want err = %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("ParseLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLanguageInstruction(t *testing.T) {
	if got := LanguageInstruction("auto"); got != "Reply in the language of the conversation." {
		t.Errorf("unexpected auto instruction: %q", got)
	}
	if got := LanguageInstruction("en"); got != "The conversation is in English. Reply in English." {
		t.Errorf("unexpected en instruction: %q", got)
	}
	if got := LanguageInstruction("en,bn"); got != "The conversation is bilingual in English and Bangla. Reply in the languages of the conversation." {
		t.Errorf("unexpected en,bn instruction: %q", got)
	}
}
