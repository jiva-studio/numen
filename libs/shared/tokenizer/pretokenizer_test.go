package tokenizer_test

import "testing"

func writeWhitespaceSplitVariant(t *testing.T) string {
	t.Helper()
	return writeVariant(t, func(file map[string]any) {
		file["normalizer"] = nil
		file["pre_tokenizer"] = map[string]any{
			"type": "Sequence",
			"pretokenizers": []any{
				map[string]any{"type": "WhitespaceSplit"},
				map[string]any{"type": "Metaspace", "replacement": "▁", "prepend_scheme": "always", "split": true},
			},
		}
	})
}

// WhitespaceSplit cuts on every White_Space rune, a tab included, and drops the empty words; Metaspace
// puts ▁ before each word.
func TestEncodeSplitsWordsOnWhitespaceBeforeMetaspace(t *testing.T) {
	tok := loadTokenizer(t, writeWhitespaceSplitVariant(t))
	checkEncode(t, tok, "  hello\tworld ", getSingleIDs(t, "▁hello", "▁world"))
	checkEncode(t, tok, " \t\n ", []int{bosID, eosID})
}

// The piece ▁hello▁world is in the vocabulary, but Metaspace splits before every ▁, so no piece
// spans two words.
func TestEncodeStartsANewPieceAtEveryReplacementRune(t *testing.T) {
	tok := loadFixture(t)
	checkEncode(t, tok, "hello world", getSingleIDs(t, "▁hello", "▁world"))
	checkEncode(t, tok, "hello▁world", getSingleIDs(t, "▁hello", "▁world"))
}

// multilingual-e5-small writes the older add_prefix_space: true and leaves out prepend_scheme and split.
func TestEncodeAcceptsTheOlderAddPrefixSpaceSetting(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		file["pre_tokenizer"] = map[string]any{"type": "Metaspace", "replacement": "▁", "add_prefix_space": true}
	})
	checkEncode(t, loadTokenizer(t, path), "hello world", getSingleIDs(t, "▁hello", "▁world"))
}

func TestEncodeOfAnEmptyTextReturnsOnlyTheSpecialTokens(t *testing.T) {
	tok := loadFixture(t)
	checkEncode(t, tok, "", []int{bosID, eosID})
	checkEncode(t, tok, "   ", []int{bosID, eosID})
}
