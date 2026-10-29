package review

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// LongestAnswer is the maximum time one answer counts for. A card left displayed
// for an hour is capped, because the idle time is not review.
const LongestAnswer = time.Minute

// GetCountedTime returns how long this answer counts for, capped at LongestAnswer.
func (a Answer) GetCountedTime() time.Duration { return min(a.Took, LongestAnswer) }

// ShortestAnswer is the minimum time a kind of answer is costed at. A card graded
// before it could be read is an accidental key press and not review.
const ShortestAnswer = time.Second

// LeastAnswers is the number of answers of a kind that a history needs before its
// median replaces the default.
const LeastAnswers = 10

// AnswerCost is how long an answer takes: one for a card that the scheduler is
// still learning, and one for a card that returns after days.
//
// ShouldReadNew and ShouldReadReview report which halves the history supplied. A
// half without enough answers keeps the default value.
type AnswerCost struct {
	New, Review                     time.Duration
	ShouldReadNew, ShouldReadReview bool
}

// DefaultCost is what a vault holding no answer times is projected at.
var DefaultCost = AnswerCost{New: 20 * time.Second, Review: 8 * time.Second, ShouldReadNew: false, ShouldReadReview: false}

// GetCostUnder returns how long an answer takes under each preset, keyed by the
// path the review cards are grouped under.
//
// A preset is costed from the answers to its own review cards: a preset of long
// cards and a preset of short cards convert the same minutes into different
// counts. A review card that no preset groups is left out, and a kind of answer
// that a preset has too few answers for keeps the default cost.
func GetCostUnder(scheduler Scheduler, answers []Answer, presetMap map[CardID]string) map[string]AnswerCost {
	target := make(map[string]*answerTimes)
	walkAnswers(scheduler, answers, func(before Schedule, a Answer) {
		path, groups := presetMap[a.ReviewCard]
		if !groups {
			return
		}
		one := target[path]
		if one == nil {
			one = &answerTimes{begun: nil, spaced: nil}
			target[path] = one
		}
		one.addAnswer(scheduler.IsSpaced(before), a.GetCountedTime())
	})

	out := make(map[string]AnswerCost, len(target))
	for path, one := range target {
		out[path] = one.cost()
	}
	return out
}

// answerTimes is how long the answers of each kind took, one entry an answer.
type answerTimes struct{ begun, spaced []time.Duration }

// addAnswer counts one answer. An answer with no recorded time carries no
// information about how long its kind takes.
func (t *answerTimes) addAnswer(spaced bool, took time.Duration) {
	if took <= 0 {
		return
	}
	if spaced {
		t.spaced = append(t.spaced, took)
		return
	}
	t.begun = append(t.begun, took)
}

// cost returns the cost these answers imply for each kind of answer, with each
// half set to the default where the history is too short.
func (t *answerTimes) cost() AnswerCost {
	out := DefaultCost
	if middle, read := getMedian(t.begun); read {
		out.New, out.ShouldReadNew = middle, true
	}
	if middle, read := getMedian(t.spaced); read {
		out.Review, out.ShouldReadReview = middle, true
	}
	return out
}

// getMedian returns the median of these answers, and whether there are enough
// answers to compute one.
//
// Answer times have a long right tail: a card stays displayed while the user is
// interrupted. The median is the value with half the answers on each side, and
// one long answer shifts it by one position.
func getMedian(took []time.Duration) (time.Duration, bool) {
	if len(took) < LeastAnswers {
		return 0, false
	}
	slices.Sort(took)
	out := took[len(took)/2]
	if len(took)%2 == 0 {
		out = (took[len(took)/2-1] + out) / 2
	}
	return max(out, ShortestAnswer), true
}

// DivideBudget divides a budget among decks in proportion to what each deck has
// due.
//
// A deck with nine times another deck's due count receives nine times the share,
// a deck with nothing due receives nothing, and no deck receives more than it has
// due; the remainder is distributed one card at a time. The units left over by
// the proportions go to the largest fractions, with ties going to the deck with
// more due, and decks with equal due counts receive them in the order given.
//
// The proportion is computed in floating point: a budget in nanoseconds times
// one deck's due count overflows an integer. Every returned share is a whole
// number.
func DivideBudget(budget int, owes []int) []int {
	out := make([]int, len(owes))
	total := 0
	for _, one := range owes {
		total += one
	}
	if total <= 0 {
		return out
	}
	if budget >= total {
		copy(out, owes)
		return out
	}
	shares := make([]int, len(owes))
	parts := make([]float64, len(owes))
	remaining := budget
	for item, one := range owes {
		exact := float64(budget) * float64(one) / float64(total)
		out[item] = int(math.Floor(exact))
		parts[item] = exact - float64(out[item])
		remaining -= out[item]
		shares[item] = item
	}
	slices.SortStableFunc(shares, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(parts[b], parts[a]),
			cmp.Compare(owes[b], owes[a]),
		)
	})
	for _, entry := range shares[:min(len(shares), remaining)] {
		out[entry]++
	}
	return out
}
