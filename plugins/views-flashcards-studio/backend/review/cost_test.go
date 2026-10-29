package review_test

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// makeAnswers is a history of as many review cards, each answered as many times,
// at the time the caller gives for each answer.
//
// The first two answers to a review card are of one still being learned and every
// answer after them is of one that comes round in days, so a history of card
// review cards answered three times each holds twice as many of the first kind as of
// the second.
func makeAnswers(
	pos time.Time, reviewCards, each int, took func(reviewCard, step int) time.Duration,
) []review.Answer {
	var out []review.Answer
	for reviewCard := range reviewCards {
		on := review.CardID{Card: fmt.Sprintf("card%06d", reviewCard), Template: "Recognise"}
		for step := range each {
			out = append(out, review.Answer{
				ID: fmt.Sprintf("%06d", len(out)), ReviewCard: on,
				At:     pos.Add(time.Duration(len(out)) * time.Hour),
				Rating: review.Good, Took: took(reviewCard, step),
			})
		}
	}
	return out
}

// What an answer costs is the median of the answers of its kind: half of them
// fall on either side of it, so a single outlier moves it by one position and
// not by minutes.
func TestWhatAnAnswerCostsIsTheMiddleOfTheAnswers(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Twelve review cards answered three times each: three seconds while a card
	// review card is being learned and nine once it is scheduled in days. One of them
	// stayed on the screen for an hour every time.
	answers := makeAnswers(pos, 12, 3, func(reviewCard, step int) time.Duration {
		switch {
		case reviewCard == 0:
			return time.Hour
		case step < 2:
			return 3 * time.Second
		default:
			return 9 * time.Second
		}
	})

	cost := getCost(delta, answers)
	if cost.New != 3*time.Second {
		t.Errorf("a card being learned costs %v, want 3s", cost.New)
	}
	if cost.Review != 9*time.Second {
		t.Errorf("a review costs %v, want 9s", cost.Review)
	}
	if !cost.ShouldReadNew || !cost.ShouldReadReview {
		t.Errorf("cost = %+v, want both halves read from the answers", cost)
	}
}

// A history too short to determine what one answer kind costs leaves that kind
// at the default and reports it.
func TestAHistoryTooShortToSayExistsAtTheDefault(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// One review card answered once, for fifty-five seconds.
	answers := makeAnswers(pos, 1, 1, func(int, int) time.Duration { return 55 * time.Second })

	cost := getCost(delta, answers)
	if cost != review.DefaultCost {
		t.Errorf("cost = %+v, want the default %+v", cost, review.DefaultCost)
	}
	if cost.ShouldReadNew || cost.ShouldReadReview {
		t.Errorf("cost = %+v, want neither half read from the answers", cost)
	}
}

// The two kinds of answer are counted apart: a history of one kind leaves the
// other at the default and reports which of the two it is.
func TestACostOfOneKindPreservesTheOtherAtTheDefault(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Twelve review cards answered twice each, which is a vault holding no answer
	// to a card that comes round in days.
	answers := makeAnswers(pos, 12, 2, func(int, int) time.Duration { return 30 * time.Second })

	cost := getCost(delta, answers)
	if cost.New != 30*time.Second || !cost.ShouldReadNew {
		t.Errorf("a card being learned costs %v, read %t, want 30s read", cost.New, cost.ShouldReadNew)
	}
	if cost.Review != review.DefaultCost.Review || cost.ShouldReadReview {
		t.Errorf("a review costs %v, read %t, want the default %v unread",
			cost.Review, cost.ShouldReadReview, review.DefaultCost.Review)
	}
}

// A history of key hits is costed at the shortest an answer is costed at, so a
// day is never priced at more cards than can be answered.
func TestAKindOfAnswerIsNeverCostedBelowTheShortest(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	answers := makeAnswers(pos, 12, 3, func(int, int) time.Duration { return 20 * time.Millisecond })

	cost := getCost(delta, answers)
	if cost.New != review.ShortestAnswer || cost.Review != review.ShortestAnswer {
		t.Errorf("cost = %+v, want both halves at %v", cost, review.ShortestAnswer)
	}
}

// An abandoned answer is capped at the longest cost, so an hour away from the
// screen does not count as an hour of review.
func TestAnAnswerNobodySatThroughIsCappedAtTheLongest(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Every answer stayed on the screen for an hour, so the median of them is
	// what one answer is capped at.
	answers := makeAnswers(pos, 12, 3, func(int, int) time.Duration { return time.Hour })

	cost := getCost(delta, answers)
	if cost.New != review.LongestAnswer || cost.Review != review.LongestAnswer {
		t.Errorf("cost = %+v, want both halves at %v", cost, review.LongestAnswer)
	}
}

// A vault holding no answer times is projected at the default, and not at
// nothing a minute.
func TestAVaultContainingNoAnswerTimesIsProjectedAtTheDefault(t *testing.T) {
	if got := getCost(review.NewFSRS(), nil); got != review.DefaultCost {
		t.Errorf("cost = %+v, want the default %+v", got, review.DefaultCost)
	}
}

// An answer carrying no time reveals nothing about how long its kind takes, so
// only the answers that carry a time determine a cost.
func TestAnAnswerCarryingNoTimeReturnsNothingAboutItsKind(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Twenty-four review cards answered three times each, at nine seconds an
	// answer. Half the review cards were answered with no time recorded at all.
	answers := makeAnswers(pos, 24, 3, func(reviewCard, _ int) time.Duration {
		if reviewCard%2 == 0 {
			return 0
		}
		return 9 * time.Second
	})

	cost := getCost(delta, answers)
	if cost.New != 9*time.Second || cost.Review != 9*time.Second {
		t.Errorf("cost = %+v, want both halves at 9s", cost)
	}
}

// getCost is the cost of the answers taken as one group.
func getCost(delta review.Scheduler, answers []review.Answer) review.AnswerCost {
	const group = "all"
	underMap := make(map[review.CardID]string)
	for _, a := range answers {
		underMap[a.ReviewCard] = group
	}
	costs := review.GetCostUnder(delta, answers, underMap)
	if cost, ok := costs[group]; ok {
		return cost
	}
	return review.DefaultCost
}

// drawDivision generates the amount each of a preset's decks has due and the
// budget to divide between them. The budget range covers values above the total
// due and below any single share, so both ends of the division are generated as
// often as the middle.
func drawDivision(t *rapid.T) (int, []int) {
	owes := rapid.SliceOfN(rapid.IntRange(0, 400), 0, 8).Draw(t, "owes")
	return rapid.IntRange(0, 1200).Draw(t, "budget"), owes
}

// getTotalOwed returns the total due across a set of decks.
func getTotalOwed(owes []int) int {
	out := 0
	for _, one := range owes {
		out += one
	}
	return out
}

// A day is distributed whole: the decks together take the budget when it is
// smaller than the total due, and the whole total due when it is not. No
// unusable remainder is left on the day.
//
// A preset's day is divided over the decks it schedules.
func TestTheSharesOfADayAreHandedOutWhole(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := drawDivision(t)
		got := getTotalOwed(review.DivideBudget(budget, owes))
		want := min(budget, getTotalOwed(owes))
		if got != want {
			t.Fatalf("a budget of %v over %v handed out %v, want %v",
				budget, owes, got, want)
		}
	})
}

// No deck takes more of the day than it has due, and a deck with nothing due
// takes nothing: a share is bounded by what the deck has due, so a deck with
// nothing due receives no share.
//
// The shares are proportional to what the decks owe.
func TestDivideBudget_ShareNeverExceedsOwed(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := drawDivision(t)
		for item, share := range review.DivideBudget(budget, owes) {
			if share < 0 || share > owes[item] {
				t.Fatalf("a deck owing %v took %v of a budget of %v over %v",
					owes[item], share, budget, owes)
			}
		}
	})
}

// the other: three decks owing the same take a third each, and a deck owing
// nine times another's takes nine times the share. What the proportions leave
// over is under one card a deck, so no share is a whole card away from the
// proportion it is drawn from.
//
// The shares are proportional, and what they leave over goes by the largest
// fraction.
func TestDivideBudget_SharesAreProportional(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := drawDivision(t)
		total := getTotalOwed(owes)
		if total <= 0 || budget >= total {
			return
		}
		for item, share := range review.DivideBudget(budget, owes) {
			exact := float64(budget) * float64(owes[item]) / float64(total)
			if float64(share) < math.Floor(exact) || float64(share) > math.Ceil(exact) {
				t.Fatalf("a deck owing %v of %v took %v of a budget of %v, "+
					"where its proportion of it is %v",
					owes[item], total, share, budget, exact)
			}
		}
	})
}

// A deck owing more of the day never takes less of it than a deck owing less,
// and decks owing the same take what is left over in the order they are given.
//
// A deck owing nine times another's takes nine times the share, and where the
// fractions stand equal the deck owing more takes the remainder.
func TestADeckOwingMoreNeverConsumesLess(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := drawDivision(t)
		out := review.DivideBudget(budget, owes)
		for i := range owes {
			for j := range owes {
				ahead := owes[i] > owes[j] || (owes[i] == owes[j] && i < j)
				if ahead && out[i] < out[j] {
					t.Fatalf("a deck owing %v took %v where one owing %v took %v, "+
						"out of a budget of %v over %v",
						owes[i], out[i], owes[j], out[j], budget, owes)
				}
			}
		}
	})
}

// The decks take the same shares between them whatever order they are given in.
// A share is computed from the amount its deck has due and nothing else, so renaming a
// deck's file moves no card from one deck to another.
//
// The shares are proportional to what the decks owe, and where the fractions
// are equal the deck owing more takes the remainder.
func TestTheOrderDecksAreGivenInDoesNotChangeTheShares(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := drawDivision(t)
		places := make([]int, len(owes))
		for item := range places {
			places[item] = item
		}
		otherwise := make([]int, len(owes))
		for entry, one := range rapid.Permutation(places).Draw(t, "order") {
			otherwise[entry] = owes[one]
		}

		was, now := review.DivideBudget(budget, owes), review.DivideBudget(budget, otherwise)
		slices.Sort(was)
		slices.Sort(now)
		if !slices.Equal(was, now) {
			t.Fatalf("a budget of %v over %v was handed out as %v, and as %v "+
				"with the decks given in another order", budget, owes, was, now)
		}
	})
}

// Decks owing different amounts have the same fraction: half a day leaves a
// deck owing an odd number half a card over, whatever that number is, so a deck
// owing one and a deck owing seven are equal. The two remaining cards go to the
// two decks owing the most among those with equal fractions, and not to the
// decks owing one card each.
//
// Where the fractions are equal the deck owing more takes the remainder.
func TestDecksLevelOnTheFractionAreSplitByWhatTheyOwe(t *testing.T) {
	t.Parallel()
	owes := []int{0, 1, 1, 7, 239, 400}
	want := []int{0, 0, 0, 4, 120, 200}
	if got := review.DivideBudget(324, owes); !slices.Equal(got, want) {
		t.Fatalf("half a day over %v was handed out as %v, want %v",
			owes, got, want)
	}
}
