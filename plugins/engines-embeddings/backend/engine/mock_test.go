package engine

import (
	"math"
	"slices"
	"testing"
)

// floor is the cosine a passage must reach to answer a search by meaning.
const floor = 0.5

func cosine(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

func TestAMockQuestionIsNearAPassageThatNamesItsConceptByAnotherWord(t *testing.T) {
	question := embedMock("boat")
	passage := embedMock("The ferry leaves the harbour at dawn with the morning post.")
	if got := cosine(question, passage); got < floor {
		t.Errorf("a passage about a ferry stands %.3f from a question about a boat, want at least %.2f", got, floor)
	}
}

func TestAMockQuestionIsFarFromAPassageSharingNoConcept(t *testing.T) {
	question := embedMock("boat")
	passage := embedMock("A measure of uncertainty. To Jaynes it is a property of the description, not of the system.")
	if got := cosine(question, passage); got >= floor {
		t.Errorf("a passage sharing no word with the question stands %.3f from it, want under %.2f", got, floor)
	}
}

func TestMockQuestionIsNearPassageWithItsWord(t *testing.T) {
	question := embedMock("Entropy")
	passage := embedMock("Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description, not of the system.")
	if got := cosine(question, passage); got < floor {
		t.Errorf("a passage holding the word stands %.3f from the question, want at least %.2f", got, floor)
	}
}

func TestAMockVectorIsUnitLengthAndTheSameEveryTime(t *testing.T) {
	text := "The ferry leaves the harbour at dawn."
	first, second := embedMock(text), embedMock(text)
	if !slices.Equal(first, second) {
		t.Error("one text gave two vectors")
	}
	if got := math.Sqrt(cosine(first, first)); math.Abs(got-1) > 1e-5 {
		t.Errorf("length %.6f, want 1", got)
	}
	if len(first) != GetMockModel().Dimensions {
		t.Errorf("got %d dimensions, want %d", len(first), GetMockModel().Dimensions)
	}
}
