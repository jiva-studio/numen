package tokenizer_test

import (
	"strconv"
	"strings"
	"testing"
)

// Strip trims both sides first, then Replace turns the run of spaces inside into one ▁. Without
// Strip a trailing ▁ piece would follow; without Replace a lone ▁ piece would sit between a and b.
func TestEncodeStripsTheTextThenReplacesRunsOfSpaces(t *testing.T) {
	checkEncode(t, loadFixture(t), "  a  b  ", getSingleIDs(t, "▁", "a", "▁", "b"))
}

func TestEncodeStripsOnlyTheSidesTheFileNames(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Strip", "strip_left": false, "strip_right": true}
	})
	// Kept on the left, the two spaces become ▁▁a: a lone ▁, then ▁ and a.
	checkEncode(t, loadTokenizer(t, path), "  a\u3000\t", getSingleIDs(t, "▁", "▁", "a"))
}

func TestEncodeReplacesEveryLiteralOccurrenceOfAStringPattern(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"String": "ʘ"}, "content": "a"}
	})
	checkEncode(t, loadTokenizer(t, path), "ʘxʘ", getSingleIDs(t, "▁", "a", "x", "a"))
}

// The content of a Regex pattern is inserted as it is: $1 is not a group reference.
func TestEncodeInsertsTheReplacementOfARegexLiterally(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"Regex": "(x)"}, "content": "$1"}
	})
	// x is a piece; $ and 1 are not, so they become one fused unknown after ▁.
	checkEncode(t, loadTokenizer(t, path), "x", getSingleIDs(t, "▁", "<unk>"))
}

// RE2 reads \s, \w, \d and \b as ASCII, Oniguruma in Hugging Face as Unicode: a no-break space
// would be replaced there and kept here, so the ids would differ.
func TestLoadRefusesAReplaceRegexWithAUnicodeClassAndNamesThePattern(t *testing.T) {
	for _, pattern := range []string{`\s+`, `\w`, `\d{2}`, `\bx`, `[\s,]`} {
		path := writeVariant(t, func(file map[string]any) {
			file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"Regex": pattern}, "content": "▁"}
		})
		_, err := loadWithError(t, path)
		if err == nil {
			t.Errorf("Load with the Replace regex %s succeeded, want an error", pattern)
			continue
		}
		message := err.Error()
		isPatternNamed := strings.Contains(message, pattern) || strings.Contains(message, strconv.Quote(pattern))
		if !strings.Contains(message, "Replace") || !isPatternNamed {
			t.Errorf("Load with the Replace regex %s: error %q, want one that names Replace and the pattern", pattern, message)
		}
	}
}

func TestLoadAcceptsAReplaceRegexOfPlainSpaces(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"Regex": " {2,}"}, "content": "▁"}
	})
	checkEncode(t, loadTokenizer(t, path), "a  b", getSingleIDs(t, "▁", "a", "▁", "b"))
}
