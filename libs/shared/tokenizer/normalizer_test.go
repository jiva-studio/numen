package tokenizer_test

import "testing"

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
