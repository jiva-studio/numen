package embedding_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/embedding"
)

func sizes(batches [][]string) []int {
	out := make([]int, len(batches))
	for i, b := range batches {
		out[i] = len(b)
	}
	return out
}

func TestBatchesFillToTheBudget(t *testing.T) {
	texts := []string{"aaaa", "bbbb", "cccc", "dddd", "ee"}
	// Nine characters holds two texts of four and cuts before the third.
	got := embedding.Batches(texts, 9)
	if want := []int{2, 2, 1}; !slices.Equal(sizes(got), want) {
		t.Fatalf("got %v, want %v", sizes(got), want)
	}
	if !slices.Equal(slices.Concat(got...), texts) {
		t.Errorf("order changed: %v", got)
	}
}

func TestOneTextOverTheBudgetGoesAlone(t *testing.T) {
	texts := []string{"aa", strings.Repeat("b", 50), "cc"}
	got := embedding.Batches(texts, 10)
	if want := []int{1, 1, 1}; !slices.Equal(sizes(got), want) {
		t.Errorf("got %v, want %v", sizes(got), want)
	}
}

func TestNoTexts(t *testing.T) {
	if got := embedding.Batches(nil, 10); got != nil {
		t.Errorf("got %v", got)
	}
}

// The budget is characters, and Devanagari takes several bytes a character.
func TestANonLatinScriptIsBudgetedByCharacters(t *testing.T) {
	verse := strings.Repeat("बगीचे की बाड़ के पास एक पुराना शेड है ", 20)
	chars := len([]rune(verse))
	if chars >= len(verse) {
		t.Fatalf("expected multi-byte characters, got %d characters in %d bytes", chars, len(verse))
	}

	texts := make([]string, 8)
	for i := range texts {
		texts[i] = verse
	}
	got := embedding.Batches(texts, chars*3)
	if want := []int{3, 3, 2}; !slices.Equal(sizes(got), want) {
		t.Errorf("got %v, want %v", sizes(got), want)
	}
}

func TestATransliteratedTextIsBudgetedTheSame(t *testing.T) {
	// The characters are Latin; only the token cost is higher, which is what
	// the budget is set low enough to absorb.
	verse := strings.Repeat("udyāne pathaḥ dvāraṁ bījāni śākhāḥ jalaṁ ", 20)
	chars := len([]rune(verse))
	texts := []string{verse, verse, verse}
	if got := embedding.Batches(texts, chars*2); !slices.Equal(sizes(got), []int{2, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestABudgetOfZeroFallsBackToTheDefault(t *testing.T) {
	texts := []string{strings.Repeat("a", embedding.DefaultBatchCharacters), "b"}
	if got := embedding.Batches(texts, 0); !slices.Equal(sizes(got), []int{1, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}
