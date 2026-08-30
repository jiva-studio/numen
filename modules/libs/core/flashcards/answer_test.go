package flashcards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// Each of the four is said in the word a person uses for it, which is what a
// log line and anything read by a person carry.
func TestEachRatingIsSaidInItsOwnWord(t *testing.T) {
	for _, one := range []struct {
		r    flashcards.Rating
		want string
	}{
		{flashcards.Again, "again"},
		{flashcards.Hard, "hard"},
		{flashcards.Good, "good"},
		{flashcards.Easy, "easy"},
		{0, "unknown"},
		{flashcards.Easy + 1, "unknown"},
	} {
		if got := one.r.String(); got != one.want {
			t.Errorf("%d is said as %q, want %q", one.r, got, one.want)
		}
	}
}

// Only the four are ratings. A number out of a file that is not one of them is
// a line nobody can act on, and saying so is what leaves it out.
func TestOnlyTheFourAreRatings(t *testing.T) {
	for _, r := range []flashcards.Rating{
		flashcards.Again, flashcards.Hard, flashcards.Good, flashcards.Easy,
	} {
		if !r.Valid() {
			t.Errorf("%s is not one of the four", r)
		}
	}
	for _, r := range []flashcards.Rating{0, flashcards.Easy + 1, 255} {
		if r.Valid() {
			t.Errorf("%d passed as one of the four", r)
		}
	}
}
