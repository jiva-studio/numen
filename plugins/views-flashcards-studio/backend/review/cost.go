package review

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// LongestAnswer is the most one answer is counted at. A card left on the screen
// while a person answered the door stands there for an hour, and the hour is
// not review.
const LongestAnswer = time.Minute

// GetCountedTime is how long this answer counts for, wherever one is counted:
func (a Answer) GetCountedTime() time.Duration { return min(a.Took, LongestAnswer) }

// ShortestAnswer is the shortest a kind of answer is costed at. A card graded
// before it could be read is a key hit and not review.
const ShortestAnswer = time.Second

// LeastAnswers is how many answers of a kind a history holds before it says
// default.
const LeastAnswers = 10

// AnswerCost is how long an answer takes: one of a card the scheduler is still
// putting into memory, and one of a card that comes round in days.
//
// ReadNew and ReadReview say which halves the history answered. A half it does
// a person says which of the two it is showing.
type AnswerCost struct {
	New, Review                     time.Duration
	ShouldReadNew, ShouldReadReview bool
}

// DefaultCost is what a vault holding no answer times is projected at.
var DefaultCost = AnswerCost{New: 20 * time.Second, Review: 8 * time.Second}

// GetCostUnder is how long an answer takes under each preset, by the path the
// review cards are grouped under.
//
// A preset is costed from the answers to its own review cards: a preset of long
// cards and one of short cards turn the same minutes into different counts. A
// review card nothing groups is left out, and a kind of answer a preset holds too
func GetCostUnder(scheduler Scheduler, answers []Answer, presetMap map[ReviewCardID]string) map[string]AnswerCost {
	target := make(map[string]*answerTimes)
	walkAnswers(scheduler, answers, func(before Schedule, a Answer) {
		path, groups := presetMap[a.ReviewCard]
		if !groups {
			return
		}
		one := target[path]
		if one == nil {
			one = &answerTimes{}
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

// addAnswer counts one answer. An answer carrying no time at all says nothing
// about how long its kind takes.
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

// cost is what these answers say a kind of answer takes, each half standing at
// the default where the history is too short to say.
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

// getMedian is the middle of these answers, and whether there are enough of
// them to have one.
//
// Answer times are a right tail: a person answers the door, and the card stands
// on the screen while they do. The middle is where half the answers fall either
// side of it, and one long answer moves it by one place.
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

// DivideBudget is a budget handed out in proportion to what each deck owes of it.
//
// A deck owing nine times another's takes nine times the share, a deck owing
// nothing takes nothing, and no deck takes more than it owes — the remainder is
// handed out a card at a time. What the proportions leave over goes by the
// largest fraction, and where the fractions stand equal the deck owing more
// takes it; decks owing the same take it in the order they are given.
//
// The proportion itself is worked out in floating point: a budget of
// nanoseconds times what one deck owes of them overflows an integer. Every
// share handed back is whole.
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
