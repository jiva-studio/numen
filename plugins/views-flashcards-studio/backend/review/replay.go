package review

import (
	"slices"
	"strings"
)

// SchedulingPolicy is what one review card is computed under: the scheduler
// that spaces it, and the preset that determines which day it falls on.
type SchedulingPolicy struct {
	By     Scheduler
	Preset Preset
}

// Assignment is how one review card is scheduled. A review card is scheduled by the
// preset its deck points at, and two presets asking for different shares of the
// cards send the same card away for different lengths of time.
type Assignment func(CardID) SchedulingPolicy

// ScheduleBy is one scheduler for every review card, on the preset a deck naming
// none is scheduled by.
func ScheduleBy(s Scheduler) Assignment {
	return func(CardID) SchedulingPolicy { return SchedulingPolicy{By: s, Preset: Defaults()} }
}

// History is a vault's answers in the order they were given: no revoked lines,
// one line per identifier, earliest first.
//
// One request queries several things from one history: the schedule of each
// card, the tally of each day, and what a preset spent. The order is computed
// once and shared by all of them.
type History []Answer

// OrderAnswers puts a vault's answers in the order they were given.
func OrderAnswers(answers []Answer) History { return getActiveAnswers(answers) }

// Replay computes the schedule of every review card in this history, each under
// the scheduler its own preset requests and on the day its own preset assigns.
//
// The days already filled by earlier answers are what the next card is placed
// against.
func (h History) Replay(d Day, assignment Assignment) map[CardID]Schedule {
	out := make(map[CardID]Schedule)
	on := NewDueByDay(d)
	for _, a := range h {
		one := assignment(a.ReviewCard)
		next := one.By.Next(out[a.ReviewCard], a.At, a.Rating)
		next.Due = one.Preset.AssignDueDay(on, a.At, next.Due)
		out[a.ReviewCard] = next
	}
	return out
}

// RecallTally is how many of the cards that came due in days were recalled,
// over one day: the answers given to spaced review cards, and how many of those
// were recalled.
//
// A review card still in the learning phase is in neither count. The question
// for such a card is whether it is recalled after ten minutes, which gives no
// information about long-term memory.
type RecallTally struct {
	// Tested is the answers given to spaced review cards, and Recalled the subset
	// of them that were not Again.
	Tested   int
	Recalled int
}

// GetRetained is what came back on each day, by the name of the day.
//
// It is worked out with the replay: whether a review card was spaced is a thing
// only the answers before it can say.
func (h History) GetRetained(scheduler Scheduler, d Day) map[string]RecallTally {
	out := make(map[string]RecallTally)
	h.walkAnswers(scheduler, func(before Schedule, a Answer) {
		if !scheduler.IsSpaced(before) {
			return
		}
		day := d.GetName(a.At)
		one := out[day]
		one.Tested++
		if a.Rating != Again {
			one.Recalled++
		}
		out[day] = one
	})
	return out
}

// walkAnswers iterates over the answers in the order they were given and passes each
// one to the caller with the schedule the review card had before it.
func walkAnswers(
	delta Scheduler, answers []Answer, each func(before Schedule, a Answer),
) map[CardID]Schedule {
	return OrderAnswers(answers).walkAnswers(delta, each)
}

func (h History) walkAnswers(
	delta Scheduler, each func(before Schedule, a Answer),
) map[CardID]Schedule {
	out := make(map[CardID]Schedule)
	for _, a := range h {
		before := out[a.ReviewCard]
		if each != nil {
			each(before, a)
		}
		out[a.ReviewCard] = delta.Next(before, a.At, a.Rating)
	}
	return out
}

// getActiveAnswers is the answers that count, in the order they were given.
//
// An answer some line takes back is left out, and one identifier is one answer
// however many lines carry it. The files arrive in whatever order they were
// synchronised, and what a review card has been through is the order of the
// answers themselves.
func getActiveAnswers(answers []Answer) []Answer {
	undoTargets := make(map[string]bool)
	for _, a := range answers {
		if a.IsUndo() {
			undoTargets[a.Undoes] = true
		}
	}

	observed := make(map[string]bool, len(answers))
	out := make([]Answer, 0, len(answers))
	for _, a := range answers {
		if a.IsUndo() || undoTargets[a.ID] || observed[a.ID] {
			continue
		}
		observed[a.ID] = true
		out = append(out, a)
	}
	slices.SortStableFunc(out, compareAnswersByTime)
	return out
}

// compareAnswersByTime puts answers in the order they were given.
//
// Two answers of one millisecond are put in the order of their identifiers,
// which is one order and the same at every launch. Which of them was given
// first is not known: an identifier carries the millisecond and then
// randomness.
func compareAnswersByTime(a, b Answer) int {
	if !a.At.Equal(b.At) {
		return a.At.Compare(b.At)
	}
	return strings.Compare(a.ID, b.ID)
}
