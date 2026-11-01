package engine

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"
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

func TestMockEmbedderBlocksABatchWithTheHoldWordUntilCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() {
		_, err := MockEmbedder{}.Embed(ctx, []string{"a note", "a Note with NumenMockHold in it"})
		ended <- err
	}()

	select {
	case err := <-ended:
		t.Fatalf("the held batch returned before its context ended: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	if err := <-ended; !errors.Is(err, context.Canceled) {
		t.Errorf("the held batch ended with %v, want context.Canceled", err)
	}

	vectors, err := MockEmbedder{}.Embed(t.Context(), []string{"a note"})
	if err != nil || len(vectors) != 1 {
		t.Errorf("a batch without the hold word answered %d vectors and %v", len(vectors), err)
	}
}

func TestEmbedMock_ParaphraseQuestionNearPassage(t *testing.T) {
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
