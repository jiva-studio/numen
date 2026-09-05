package review

import (
	"slices"
	"time"
)

// LongestAnswer is the most one answer is counted at. A card left on the screen
// while a person answered the door stands there for an hour, and the hour is
// not review.
const LongestAnswer = time.Minute

// ShortestAnswer is the shortest a kind of answer is costed at. A card graded
// before it could be read is a key hit and not review.
const ShortestAnswer = time.Second

// LeastAnswers is how many answers of a kind a history holds before it says
// what that kind costs. A kind the history holds fewer of stands at the
// default.
const LeastAnswers = 10

// AnswerCost is how long an answer takes: one of a card the scheduler is still
// putting into memory, and one of a card that comes round in days.
//
// ReadNew and ReadReview say which halves the history answered. A half it does
// not answer stands at the default, and a caller putting the number in front of
// a person says which of the two it is showing.
type AnswerCost struct {
	New, Review         time.Duration
	ReadNew, ReadReview bool
}

// DefaultCost is what a vault holding no answer times is projected at.
var DefaultCost = AnswerCost{New: 20 * time.Second, Review: 8 * time.Second}

// Costed is how long an answer takes in this history, from the times the
// answers themselves carry. A kind of answer the history holds too few of
// stands at the default.
func Costed(by Scheduler, answers []Answer) AnswerCost {
	var took answerTimes
	replayed(by, answers, func(before Schedule, a Answer) {
		took.holds(by.Spaced(before), a.Took)
	})
	return took.cost()
}

// CostedUnder is how long an answer takes under each preset, by the path the
// card faces are grouped under.
//
// A preset is costed from the answers to its own card faces: a preset of long
// cards and one of short cards turn the same minutes into different counts. A
// card face nothing groups is left out, and a kind of answer a preset holds too
// few of stands at the default.
func CostedUnder(by Scheduler, answers []Answer, under map[CardFaceID]string) map[string]AnswerCost {
	held := make(map[string]*answerTimes)
	replayed(by, answers, func(before Schedule, a Answer) {
		path, groups := under[a.CardFace]
		if !groups {
			return
		}
		one := held[path]
		if one == nil {
			one = &answerTimes{}
			held[path] = one
		}
		one.holds(by.Spaced(before), a.Took)
	})

	out := make(map[string]AnswerCost, len(held))
	for path, one := range held {
		out[path] = one.cost()
	}
	return out
}

// answerTimes is how long the answers of each kind took, one entry an answer.
type answerTimes struct{ begun, spaced []time.Duration }

// holds counts one answer, capped at LongestAnswer. An answer carrying no time
// at all says nothing about how long its kind takes.
func (t *answerTimes) holds(spaced bool, took time.Duration) {
	took = min(took, LongestAnswer)
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
	if middle, read := middling(t.begun); read {
		out.New, out.ReadNew = middle, true
	}
	if middle, read := middling(t.spaced); read {
		out.Review, out.ReadReview = middle, true
	}
	return out
}

// middling is the middle of these answers, and whether there are enough of them
// to have one.
//
// Answer times are a right tail: a person answers the door, and the card stands
// on the screen while they do. The middle is where half the answers fall either
// side of it, and one long answer moves it by one place.
func middling(took []time.Duration) (time.Duration, bool) {
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
