package review_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// timed is a history of as many card faces, each answered as many times, at
// the time the caller gives for each answer.
//
// The first two answers to a card face are of one still being learned and every
// answer after them is of one that comes round in days, so a history of card
// faces answered three times each holds twice as many of the first kind as of
// the second.
func timed(
	at time.Time, faces, each int, took func(face, step int) time.Duration,
) []review.Answer {
	var out []review.Answer
	for face := range faces {
		on := review.CardFaceID{Card: fmt.Sprintf("card%06d", face), Face: "Recognise"}
		for step := range each {
			out = append(out, review.Answer{
				ID: fmt.Sprintf("%06d", len(out)), CardFace: on,
				At:     at.Add(time.Duration(len(out)) * time.Hour),
				Rating: review.Good, Took: took(face, step),
			})
		}
	}
	return out
}

// What an answer costs is the middle of the answers to its kind: half of them
// fall either side of it, so a person who answered the door once moves it by one
// place and not by minutes.
func TestWhatAnAnswerCostsIsTheMiddleOfTheAnswers(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	// Twelve card faces answered three times each: three seconds while a card
	// face is being learned and nine once it comes round in days. One of them
	// stood on the screen for an hour every time.
	answers := timed(at, 12, 3, func(face, step int) time.Duration {
		switch {
		case face == 0:
			return time.Hour
		case step < 2:
			return 3 * time.Second
		default:
			return 9 * time.Second
		}
	})

	cost := review.Costed(by, answers)
	if cost.New != 3*time.Second {
		t.Errorf("a card being learned costs %v, want 3s", cost.New)
	}
	if cost.Review != 9*time.Second {
		t.Errorf("a review costs %v, want 9s", cost.Review)
	}
	if !cost.ReadNew || !cost.ReadReview {
		t.Errorf("cost = %+v, want both halves read from the answers", cost)
	}
}

// A history too short to say what a kind of answer costs leaves that kind at
// the default, and says it did.
func TestAHistoryTooShortToSayStandsAtTheDefault(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	// One card face answered once, for fifty-five seconds.
	answers := timed(at, 1, 1, func(int, int) time.Duration { return 55 * time.Second })

	cost := review.Costed(by, answers)
	if cost != review.DefaultCost {
		t.Errorf("cost = %+v, want the default %+v", cost, review.DefaultCost)
	}
	if cost.ReadNew || cost.ReadReview {
		t.Errorf("cost = %+v, want neither half read from the answers", cost)
	}
}

// The two kinds of answer are counted apart: a history of one kind leaves the
// other at the default and says which of the two it is.
func TestACostOfOneKindLeavesTheOtherAtTheDefault(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	// Twelve card faces answered twice each, which is a vault holding no answer
	// to a card that comes round in days.
	answers := timed(at, 12, 2, func(int, int) time.Duration { return 30 * time.Second })

	cost := review.Costed(by, answers)
	if cost.New != 30*time.Second || !cost.ReadNew {
		t.Errorf("a card being learned costs %v, read %t, want 30s read", cost.New, cost.ReadNew)
	}
	if cost.Review != review.DefaultCost.Review || cost.ReadReview {
		t.Errorf("a review costs %v, read %t, want the default %v unread",
			cost.Review, cost.ReadReview, review.DefaultCost.Review)
	}
}

// A history of key hits is costed at the shortest an answer is costed at, so a
// day is never priced at more cards than a person could sit through.
func TestAKindOfAnswerIsNeverCostedBelowTheShortest(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	answers := timed(at, 12, 3, func(int, int) time.Duration { return 20 * time.Millisecond })

	cost := review.Costed(by, answers)
	if cost.New != review.ShortestAnswer || cost.Review != review.ShortestAnswer {
		t.Errorf("cost = %+v, want both halves at %v", cost, review.ShortestAnswer)
	}
}

// One answer nobody sat through is counted at what a card is worth, so an hour
// away from the screen is an hour of no review.
func TestAnAnswerNobodySatThroughIsCappedAtTheLongest(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	// Every answer stood on the screen for an hour, so the middle of them is
	// what one answer is capped at.
	answers := timed(at, 12, 3, func(int, int) time.Duration { return time.Hour })

	cost := review.Costed(by, answers)
	if cost.New != review.LongestAnswer || cost.Review != review.LongestAnswer {
		t.Errorf("cost = %+v, want both halves at %v", cost, review.LongestAnswer)
	}
}

// A vault holding no answer times is projected at the default, and not at
// nothing a minute.
func TestAVaultHoldingNoAnswerTimesIsProjectedAtTheDefault(t *testing.T) {
	if got := review.Costed(review.NewFSRS(), nil); got != review.DefaultCost {
		t.Errorf("cost = %+v, want the default %+v", got, review.DefaultCost)
	}
}

// An answer carrying no time at all says nothing about how long its kind takes,
// so the answers that do carry one are the whole of what a cost is read from.
func TestAnAnswerCarryingNoTimeSaysNothingAboutItsKind(t *testing.T) {
	by := review.NewFSRS()
	at := time.Now().Add(-400 * 24 * time.Hour)

	// Twenty-four card faces answered three times each, at nine seconds an
	// answer. Half the card faces were answered with no time recorded at all.
	answers := timed(at, 24, 3, func(face, _ int) time.Duration {
		if face%2 == 0 {
			return 0
		}
		return 9 * time.Second
	})

	cost := review.Costed(by, answers)
	if cost.New != 9*time.Second || cost.Review != 9*time.Second {
		t.Errorf("cost = %+v, want both halves at 9s", cost)
	}
}
