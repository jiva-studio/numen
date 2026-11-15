package tokenizer_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// Each hello and each world is one piece, so a text of n words holds n ids. The pair template adds
// four special tokens: maxTokens is the budget plus four.
func TestEncodePairTruncatesLongestFirst(t *testing.T) {
	hello, world := getPieceIDs(t, "▁hello")[0], getPieceIDs(t, "▁world")[0]
	tok := loadFixture(t)
	for _, one := range []struct {
		name               string
		first, second      int
		maxTokens          int
		keptFirst, keptSec int
	}{
		{"a pair that fits is not cut", 2, 3, 9, 2, 3},
		{"a pair under the budget is not cut", 2, 3, 64, 2, 3},
		{"only the longer is cut when the shorter fits half", 2, 10, 10, 2, 4},
		{"both are cut, the longer gets the odd id", 6, 8, 11, 3, 4},
		{"a longer first text is cut like a longer second", 10, 2, 10, 4, 2},
		{"a longer first text gets the odd id", 8, 6, 11, 4, 3},
		{"of two equal texts the second gets the odd id", 6, 6, 11, 3, 4},
		{"an empty first text leaves the budget to the second", 0, 10, 8, 0, 4},
		{"a budget of one goes to the second", 3, 3, 5, 0, 1},
		{"a budget of zero keeps the special tokens alone", 3, 3, 4, 0, 0},
	} {
		first := strings.Repeat("hello ", one.first)
		second := strings.Repeat("world ", one.second)
		got, err := tok.EncodePair(first, second, one.maxTokens)
		if err != nil {
			t.Errorf("%s: %v", one.name, err)
			continue
		}
		ids := slices.Concat([]int{bosID}, slices.Repeat([]int{hello}, one.keptFirst), []int{eosID, eosID}, slices.Repeat([]int{world}, one.keptSec), []int{eosID})
		types := slices.Concat(slices.Repeat([]int{0}, one.keptFirst+2), slices.Repeat([]int{1}, one.keptSec+2))
		if !slices.Equal(got.IDs, ids) || !slices.Equal(got.TypeIDs, types) {
			t.Errorf("%s: EncodePair(%d, %d words, %d) = %v %v, want %v %v", one.name, one.first, one.second, one.maxTokens, got.IDs, got.TypeIDs, ids, types)
		}
		if len(got.IDs) > one.maxTokens {
			t.Errorf("%s: %d ids for maxTokens %d", one.name, len(got.IDs), one.maxTokens)
		}
	}
}

func TestEncodePairRefusesABudgetBelowTheSpecialTokens(t *testing.T) {
	tok := loadFixture(t)
	for _, maxTokens := range []int{3, 0, -1} {
		if _, err := tok.EncodePair("hello", "world", maxTokens); !errors.Is(err, tokenizer.ErrBudgetTooSmall) {
			t.Errorf("EncodePair with maxTokens %d = %v, want ErrBudgetTooSmall", maxTokens, err)
		}
	}
}
