package tokenizer_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// The template names <s> and </s>, and its special_tokens give them the ids 5 and 7; their
// vocabulary positions are 1 and 2.
func TestEncodeUsesTheSpecialTokenIDsOfTheTemplate(t *testing.T) {
	checkEncode(t, loadFixture(t), "", []int{5, 7})
}

func TestEncodePairPlacesBothTextsInThePairTemplateWithItsTypeIDs(t *testing.T) {
	got, err := loadFixture(t).EncodePair("a", "b", 64)
	if err != nil {
		t.Fatal(err)
	}
	ids := slices.Concat([]int{bosID}, getPieceIDs(t, "▁", "a"), []int{eosID, eosID}, getPieceIDs(t, "▁", "b"), []int{eosID})
	if !slices.Equal(got.IDs, ids) {
		t.Errorf("EncodePair ids = %v, want %v", got.IDs, ids)
	}
	if types := []int{0, 0, 0, 0, 1, 1, 1, 1}; !slices.Equal(got.TypeIDs, types) {
		t.Errorf("EncodePair type ids = %v, want %v", got.TypeIDs, types)
	}
}

func TestEncodeWithoutAPostProcessorReturnsThePiecesAlone(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) { file["post_processor"] = nil })
	tok := loadTokenizer(t, path)
	checkEncode(t, tok, "abcd", getPieceIDs(t, "▁ab", "cd"))
	if _, err := tok.EncodePair("a", "b", 64); !errors.Is(err, tokenizer.ErrNoPairTemplate) {
		t.Errorf("EncodePair without a post-processor = %v, want ErrNoPairTemplate", err)
	}
}

func TestEncodePairWithoutAPairTemplateFails(t *testing.T) {
	path := writeVariant(t, func(file map[string]any) {
		delete(getObject(t, file["post_processor"]), "pair")
	})
	tok := loadTokenizer(t, path)
	checkEncode(t, tok, "abcd", getSingleIDs(t, "▁ab", "cd"))
	if _, err := tok.EncodePair("a", "b", 64); !errors.Is(err, tokenizer.ErrNoPairTemplate) {
		t.Errorf("EncodePair without a pair template = %v, want ErrNoPairTemplate", err)
	}
}

func TestGetPadIDReturnsTheAddedPadToken(t *testing.T) {
	if id, ok := loadFixture(t).GetPadID(); !ok || id != padID {
		t.Errorf("GetPadID() = %d, %v, want %d, true", id, ok, padID)
	}
	bracketed := writeVariant(t, func(file map[string]any) {
		file["added_tokens"] = []any{
			map[string]any{"id": 0, "content": "<unk>", "special": true},
			map[string]any{"id": 6, "content": "[PAD]", "special": true},
		}
	})
	if id, ok := loadTokenizer(t, bracketed).GetPadID(); !ok || id != 6 {
		t.Errorf("GetPadID() with [PAD] = %d, %v, want 6, true", id, ok)
	}
	absent := writeVariant(t, func(file map[string]any) {
		file["added_tokens"] = []any{map[string]any{"id": 0, "content": "<unk>", "special": true}}
	})
	if id, ok := loadTokenizer(t, absent).GetPadID(); ok {
		t.Errorf("GetPadID() without a pad token = %d, true, want false", id)
	}
}

// hftokenizer reads a pad token only among the special added tokens.
func TestGetPadIDIgnoresAPadTokenThatIsNotSpecial(t *testing.T) {
	for _, content := range []string{"<pad>", "[PAD]"} {
		path := writeVariant(t, func(file map[string]any) {
			file["added_tokens"] = []any{
				map[string]any{"id": 0, "content": "<unk>", "special": true},
				map[string]any{"id": 3, "content": content, "special": false},
			}
		})
		if id, ok := loadTokenizer(t, path).GetPadID(); ok {
			t.Errorf("GetPadID() with a %s that is not special = %d, true, want false", content, id)
		}
	}
}
