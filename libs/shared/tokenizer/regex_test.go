package tokenizer_test

import "testing"

func writeReplaceRegex(t *testing.T, pattern string) string {
	t.Helper()
	return writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"Regex": pattern}, "content": "▁"}
	})
}

func TestLoadRefusesAReplaceRegexWithANegatedUnicodeClass(t *testing.T) {
	for _, pattern := range []string{`\S`, `\W+`, `[^\D]`, `x\B`, `\\\s`} {
		if _, err := loadWithError(t, writeReplaceRegex(t, pattern)); err == nil {
			t.Errorf("Load with the Replace regex %s succeeded, want an error", pattern)
		}
	}
}

func TestLoadAcceptsAReplaceRegexWithAnEscapedClassLetter(t *testing.T) {
	for _, pattern := range []string{`\\s`, `\\w\\d`, `a\\`, `\.\+`, `\Q\s\E`} {
		if _, err := loadWithError(t, writeReplaceRegex(t, pattern)); err != nil {
			t.Errorf("Load with the Replace regex %s: %v, want no error", pattern, err)
		}
	}
}

// A colon in a class or an escaped bracket is no POSIX class, so RE2 and Hugging Face agree.
func TestLoadAcceptsAReplaceRegexWithAColonThatIsNoPOSIXClass(t *testing.T) {
	for _, pattern := range []string{`[:]`, `\[:space:\]`} {
		if _, err := loadWithError(t, writeReplaceRegex(t, pattern)); err != nil {
			t.Errorf("Load with the Replace regex %s: %v, want no error", pattern, err)
		}
	}
}
