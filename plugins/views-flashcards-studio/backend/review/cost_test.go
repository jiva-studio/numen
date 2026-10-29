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
		on := review.ReviewCardID{Card: fmt.Sprintf("card%06d", reviewCard), Template: "Recognise"}
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

// What an answer costs is the middle of the answers to its kind: half of them
// fall either side of it, so a person who answered the door once moves it by one
// place and not by minutes.
func TestWhatAnAnswerCostsIsTheMiddleOfTheAnswers(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Twelve review cards answered three times each: three seconds while a card
	// review card is being learned and nine once it comes round in days. One of them
	// stood on the screen for an hour every time.
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

// A history too short to say what a kind of answer costs leaves that kind at
// the default, and says it did.
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
// other at the default and says which of the two it is.
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
// day is never priced at more cards than a person could sit through.
func TestAKindOfAnswerIsNeverCostedBelowTheShortest(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	answers := makeAnswers(pos, 12, 3, func(int, int) time.Duration { return 20 * time.Millisecond })

	cost := getCost(delta, answers)
	if cost.New != review.ShortestAnswer || cost.Review != review.ShortestAnswer {
		t.Errorf("cost = %+v, want both halves at %v", cost, review.ShortestAnswer)
	}
}

// One answer nobody sat through is counted at what a card is worth, so an hour
// away from the screen is an hour of no review.
func TestAnAnswerNobodySatThroughIsCappedAtTheLongest(t *testing.T) {
	delta := review.NewFSRS()
	pos := time.Now().Add(-400 * 24 * time.Hour)

	// Every answer stood on the screen for an hour, so the middle of them is
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

// An answer carrying no time at all says nothing about how long its kind takes,
// so the answers that do carry one are the whole of what a cost is read from.
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
	underMap := make(map[review.ReviewCardID]string)
	for _, a := range answers {
		underMap[a.ReviewCard] = group
	}
	costs := review.GetCostUnder(delta, answers, underMap)
	if cost, ok := costs[group]; ok {
		return cost
	}
	return review.DefaultCost
}

// drawDivision generates what a preset's decks owe of one of its budgets, and
// the budget there is to divide between them. A budget larger than the whole of
// what is owed and one smaller than any single share both fall inside the
// range, so both ends of the division are generated as often as the middle.
func drawDivision(t *rapid.T) (int, []int) {
	owes := rapid.SliceOfN(rapid.IntRange(0, 400), 0, 8).Draw(t, "owes")
	return rapid.IntRange(0, 1200).Draw(t, "budget"), owes
}

// getTotalOwed is what a set of decks owes altogether.
func getTotalOwed(owes []int) int {
	out := 0
	for _, one := range owes {
		out += one
	}
	return out
}

// A day is handed out whole: what the decks take between them is the budget
// where the budget is the smaller, and everything owed where it is not. What no
// deck could use is not left on the day.
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

// No deck takes more of the day than it owes of it, and a deck owing nothing
// takes nothing: a share is what a deck owes, so there is nothing to hand a
// deck that owes nothing.
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
// A share is worked out from what its deck owes and from nothing else, so
// renaming a deck's file moves no card from one deck to another.
//
// The shares are proportional to what the decks owe, and where the fractions
// stand equal the deck owing more takes the remainder.
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

// Decks owing quite different amounts stand at the same fraction: half a day
// leaves a deck owing an odd number half a card over, whatever that number is,
// so a deck owing one and a deck owing seven are level. The two cards over go to
// the two decks owing most of those standing level, and not to the decks owing
// one card each.
//
// Where the fractions stand equal the deck owing more takes the remainder.
func TestDecksLevelOnTheFractionAreSplitByWhatTheyOwe(t *testing.T) {
	t.Parallel()
	owes := []int{0, 1, 1, 7, 239, 400}
	want := []int{0, 0, 0, 4, 120, 200}
	if got := review.DivideBudget(324, owes); !slices.Equal(got, want) {
		t.Fatalf("half a day over %v was handed out as %v, want %v",
			owes, got, want)
	}
}
