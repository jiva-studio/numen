package review_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// The window draws a week's load and a curve of retention before any answer
// from here has landed, so it holds the same two ends this does. Both read the
// one corpus, and neither owns it.

func TestAWholeDaysLoadIsWhatTheCorpusSays(t *testing.T) {
	want := readPresetCorpus(t).FullLoad

	if review.FullLoad != want {
		t.Errorf("a whole day's load is %d, want %d", review.FullLoad, want)
	}
	if review.LoadBounds.Most != float64(want) {
		t.Errorf("a day is loaded up to %g, want %d", review.LoadBounds.Most, want)
	}
}

func TestRetentionGoesAsFarAsTheCorpusSays(t *testing.T) {
	want := readPresetCorpus(t).RetentionBounds

	if review.RetentionBounds.Least != want.Least || review.RetentionBounds.Most != want.Most {
		t.Errorf("retention runs %g to %g, want %g to %g",
			review.RetentionBounds.Least, review.RetentionBounds.Most, want.Least, want.Most)
	}
}
