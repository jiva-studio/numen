package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/review"
)

// The four ratings are what a person says about a card, and they order the day
// it comes round again: the better it came back, the longer it is left.
func TestTheBetterACardCameBackTheLongerItIsLeft(t *testing.T) {
	by := review.NewFSRS()
	when := at("2026-08-29T09:00:00Z")

	// A card already spaced, so that the four answers are told apart by what
	// they do to it rather than by the first steps of learning.
	learnt := by.Next(by.Next(review.Schedule{}, when, review.Good),
		when.Add(10*24*time.Hour), review.Good)
	later := learnt.Due

	var last time.Time
	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		due := by.Next(learnt, later, r).Due
		if !due.After(last) {
			t.Errorf("%v is due %v, which is no later than the answer before it at %v", r, due, last)
		}
		last = due
	}
}

// A card that did not come back at all is a lapse, and it is counted.
func TestACardThatDidNotComeBackIsALapse(t *testing.T) {
	by := review.NewFSRS()
	when := at("2026-08-29T09:00:00Z")

	learnt := by.Next(by.Next(review.Schedule{}, when, review.Good),
		when.Add(10*24*time.Hour), review.Good)
	if learnt.Lapses != 0 {
		t.Fatalf("a card answered well twice has %d lapses", learnt.Lapses)
	}

	forgotten := by.Next(learnt, learnt.Due, review.Again)
	if forgotten.Lapses != 1 {
		t.Errorf("a card that did not come back has %d lapses, want 1", forgotten.Lapses)
	}
	if forgotten.Reps != learnt.Reps+1 {
		t.Errorf("answers counted %d, want %d", forgotten.Reps, learnt.Reps+1)
	}
}

// An answer leaves a card seen, which is what tells it from one nobody has
// reached yet.
func TestAnAnsweredCardIsSeen(t *testing.T) {
	by := review.NewFSRS()
	when := at("2026-08-29T09:00:00Z")

	if by.Next(review.Schedule{}, when, review.Good).Seen() != true {
		t.Error("a card answered once is not seen")
	}
	if got := by.Next(review.Schedule{}, when, review.Good).Last; !got.Equal(when) {
		t.Errorf("answered at %v, recorded at %v", when, got)
	}
}

// A schedule says which scheduler filled it, because the numbers one carries
// between answers are its own.
func TestASchedulerSaysWhichItIs(t *testing.T) {
	if got := review.NewFSRS().Name(); got != review.FSRSName {
		t.Errorf("named itself %q, want %q", got, review.FSRSName)
	}
}
