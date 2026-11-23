package vector_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"slices"
	"strings"
	"testing"
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
	got := vector.SplitBatches(texts, 9)
	if want := []int{2, 2, 1}; !slices.Equal(sizes(got), want) {
		t.Fatalf("got %v, want %v", sizes(got), want)
	}
	if !slices.Equal(slices.Concat(got...), texts) {
		t.Errorf("order changed: %v", got)
	}
}

func TestOneTextOverTheBudgetGoesAlone(t *testing.T) {
	texts := []string{"aa", strings.Repeat("b", 50), "cc"}
	got := vector.SplitBatches(texts, 10)
	if want := []int{1, 1, 1}; !slices.Equal(sizes(got), want) {
		t.Errorf("got %v, want %v", sizes(got), want)
	}
}

func TestNoTexts(t *testing.T) {
	if got := vector.SplitBatches(nil, 10); got != nil {
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
	got := vector.SplitBatches(texts, chars*3)
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
	if got := vector.SplitBatches(texts, chars*2); !slices.Equal(sizes(got), []int{2, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestABudgetOfZeroFallsBackToTheDefault(t *testing.T) {
	texts := []string{strings.Repeat("a", vector.DefaultBatchCharacters), "b"}
	if got := vector.SplitBatches(texts, 0); !slices.Equal(sizes(got), []int{1, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestTheDefaultBudgetContainsExactlyItsCharacters(t *testing.T) {
	half := vector.DefaultBatchCharacters / 2
	fits := []string{strings.Repeat("a", half), strings.Repeat("b", half)}
	if got := vector.SplitBatches(fits, 0); !slices.Equal(sizes(got), []int{2}) {
		t.Errorf("at the budget: got %v", sizes(got))
	}
	overVal := []string{strings.Repeat("a", half), strings.Repeat("b", half+1)}
	if got := vector.SplitBatches(overVal, 0); !slices.Equal(sizes(got), []int{1, 1}) {
		t.Errorf("one past the budget: got %v", sizes(got))
	}
}

func TestANegativeBudgetFallsBackToTheDefault(t *testing.T) {
	if got := vector.SplitBatches([]string{"a", "b", "c"}, -1); !slices.Equal(sizes(got), []int{3}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestABudgetOfOneCharacterContainsOneCharacter(t *testing.T) {
	if got := vector.SplitBatches([]string{"a", "b", "c"}, 1); !slices.Equal(sizes(got), []int{1, 1, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestAFirstTextOverTheBudgetIsNotPrecededByAnEmptyGroup(t *testing.T) {
	got := vector.SplitBatches([]string{"aaaaaa", "b"}, 3)
	if !slices.Equal(sizes(got), []int{1, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}

func TestATextThatFillsTheBudgetExactlyStaysInTheGroup(t *testing.T) {
	got := vector.SplitBatches([]string{"aa", "bb", "c"}, 4)
	if !slices.Equal(sizes(got), []int{2, 1}) {
		t.Errorf("got %v", sizes(got))
	}
}
