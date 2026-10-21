package review_test

import (
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// The four ratings are what a person says about a card, and they order the day
// it comes round again: the better it came back, the longer it is left.
func TestTheBetterACardCameBackTheLongerItIsLeft(t *testing.T) {
	by := review.NewFSRS()
	when := parseTime("2026-08-29T09:00:00Z")

	// A card already spaced, so that the four answers are told apart by what
	// they do to it and not by the first steps of learning.
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
	when := parseTime("2026-08-29T09:00:00Z")

	learnt := by.Next(by.Next(review.Schedule{}, when, review.Good),
		when.Add(10*24*time.Hour), review.Good)
	if learnt.Lapses != 0 {
		t.Fatalf("a card answered well twice has %d lapses", learnt.Lapses)
	}

	cleared := by.Next(learnt, learnt.Due, review.Again)
	if cleared.Lapses != 1 {
		t.Errorf("a card that did not come back has %d lapses, want 1", cleared.Lapses)
	}
	if cleared.Reps != learnt.Reps+1 {
		t.Errorf("answers counted %d, want %d", cleared.Reps, learnt.Reps+1)
	}
}

// An answer leaves a card seen, which is what tells it from one nobody has
// reached yet.
func TestAnAnsweredCardIsSeen(t *testing.T) {
	by := review.NewFSRS()
	when := parseTime("2026-08-29T09:00:00Z")

	if by.Next(review.Schedule{}, when, review.Good).IsNew() {
		t.Error("a card answered once is still new")
	}
	if got := by.Next(review.Schedule{}, when, review.Good).Last; !got.Equal(when) {
		t.Errorf("answered at %v, recorded at %v", when, got)
	}
}

// A scheduler carries nothing from one answer to the next, so one of them
// answers for many card faces at once and each answer is the answer it would
// have given on its own.
//
// Run under the race detector this is what holds the scheduler to carrying no
// state: one that wrote anything into itself while working an interval out
// could not be shared, and a projection of a preset shares one across every
// place of a curve.
func TestASchedulerAnswersForManyCardFacesAtOnce(t *testing.T) {
	by := review.NewFSRS()
	when := parseTime("2026-08-29T09:00:00Z")

	// A spread of card faces: one nobody has answered, ones the scheduler is
	// still putting into memory, and ones it has put into review.
	cards := []review.Schedule{{}}
	for _, r := range []review.Rating{review.Again, review.Good, review.Easy} {
		one := by.Next(review.Schedule{}, when.Add(-30*24*time.Hour), r)
		cards = append(cards, one, by.Next(one, one.Due, review.Good))
	}

	alone := make([]review.Schedule, len(cards))
	good := make([]review.Schedule, len(cards))
	again := make([]review.Schedule, len(cards))
	for i, c := range cards {
		alone[i] = by.Next(c, when, review.Good)
		good[i], again[i] = by.Endings(c, when)
	}

	// Every asking writes its own answer down, so what the detector sees is the
	// scheduler being shared and nothing this test does with the answers.
	const rounds = 8
	at := make([][]review.Schedule, rounds)
	ends := make([][]review.Schedule, rounds)
	fell := make([][]review.Schedule, rounds)
	var together sync.WaitGroup
	for round := range rounds {
		at[round] = make([]review.Schedule, len(cards))
		ends[round] = make([]review.Schedule, len(cards))
		fell[round] = make([]review.Schedule, len(cards))
		for i, c := range cards {
			together.Add(1)
			go func() {
				defer together.Done()
				at[round][i] = by.Next(c, when, review.Good)
				ends[round][i], fell[round][i] = by.Endings(c, when)
			}()
		}
	}
	together.Wait()

	for round := range rounds {
		for i := range cards {
			if at[round][i] != alone[i] {
				t.Errorf("card face %d answered %v at once and %v alone",
					i, at[round][i], alone[i])
			}
			if ends[round][i] != good[i] || fell[round][i] != again[i] {
				t.Errorf("card face %d ends %v and %v at once, %v and %v alone",
					i, ends[round][i], fell[round][i], good[i], again[i])
			}
		}
	}
}

// A schedule in review with no stability of its own still answers with a
// finite, positive stability, because data a scheduler never wrote can carry
// a review phase without one.
func TestAReviewScheduleWithNoStabilityAnswersWithoutPoisoning(t *testing.T) {
	by := review.NewFSRS()
	when := parseTime("2026-08-29T09:00:00Z")

	corrupt := review.Schedule{
		Last:      when.Add(-10 * 24 * time.Hour),
		Phase:     uint8(fsrs.Review),
		Stability: 0,
	}

	assertFinitePositive := func(t *testing.T, label string, got float64) {
		if math.IsNaN(got) || math.IsInf(got, 0) || got <= 0 {
			t.Errorf("%s left stability %v, want a finite positive number", label, got)
		}
	}

	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		assertFinitePositive(t, r.String(), by.Next(corrupt, when, r).Stability)
	}

	good, again := by.Endings(corrupt, when)
	assertFinitePositive(t, "Endings good", good.Stability)
	assertFinitePositive(t, "Endings again", again.Stability)
}

// A schedule says which scheduler filled it, because the numbers one carries
// between answers are its own.
func TestASchedulerSaysWhichItIs(t *testing.T) {
	got := review.NewFSRS().GetName()
	if !strings.HasPrefix(got, review.FSRSName+".") {
		t.Errorf("named itself %q, want the algorithm and what it is running on", got)
	}
	if got == review.FSRSName+"." {
		t.Error("named itself the algorithm and nothing about its parameters")
	}
	if again := review.NewFSRS().GetName(); again != got {
		t.Errorf("named itself %q and then %q", got, again)
	}
}
