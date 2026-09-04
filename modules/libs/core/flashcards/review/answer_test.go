package review_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// Each of the four is said in the word a person uses for it, which is what a
// log line and anything read by a person carry.
func TestEachRatingIsSaidInItsOwnWord(t *testing.T) {
	for _, one := range []struct {
		r    review.Rating
		want string
	}{
		{review.Again, "again"},
		{review.Hard, "hard"},
		{review.Good, "good"},
		{review.Easy, "easy"},
		{0, "unknown"},
		{review.Easy + 1, "unknown"},
	} {
		if got := one.r.String(); got != one.want {
			t.Errorf("%d is said as %q, want %q", one.r, got, one.want)
		}
	}
}

// Only the four are ratings. A number out of a file that is not one of them is
// a line nobody can act on, and saying so is what leaves it out.
func TestOnlyTheFourAreRatings(t *testing.T) {
	for _, r := range []review.Rating{
		review.Again, review.Hard, review.Good, review.Easy,
	} {
		if !r.Valid() {
			t.Errorf("%s is not one of the four", r)
		}
	}
	for _, r := range []review.Rating{0, review.Easy + 1, 255} {
		if r.Valid() {
			t.Errorf("%d passed as one of the four", r)
		}
	}
}
