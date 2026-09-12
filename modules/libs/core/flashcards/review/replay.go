package review

import (
	"slices"
	"strings"
)

// Replay works out where a history leaves every card face it names, each placed
// on its day by the preset a deck naming none is scheduled by. Day is when a day
// of review begins.
//
// The answers arrive in whatever order the files were read, and the files
// arrive in whatever order they were synchronised, so they are put in the order
// they were given first. A schedule depends on that order.
//
// An answer some line takes back is left out, and both lines stay in the file.
// One identifier is one answer however many lines carry it: a synchroniser that
// met a conflict leaves a second copy of a run beside the first, and a person
// restoring a backup puts one there by hand.
func Replay(d Day, by Scheduler, answers []Answer) map[CardFaceID]Schedule {
	return Give(answers).Replay(d, ScheduleBy(by))
}

// SchedulingPolicy is what one card face is worked out under: the scheduler
// that spaces it, and the preset that says which day it lands on.
type SchedulingPolicy struct {
	By     Scheduler
	Preset Preset
}

// Assignment is how one card face is scheduled. A card face is scheduled by the
// preset its deck points at, and two presets asking for different shares of the
// cards send the same card away for different lengths of time.
type Assignment func(CardFaceID) SchedulingPolicy

// ScheduleBy is one scheduler for every card face, on the preset a deck naming
// none is scheduled by.
func ScheduleBy(s Scheduler) Assignment {
	return func(CardFaceID) SchedulingPolicy { return SchedulingPolicy{By: s, Preset: Defaults()} }
}

// History is a vault's answers in the order they were given: nothing a
// line takes back, one line to an identifier, earliest first.
//
// One request asks several things of one history — where it leaves each card,
// what each day came to, what a preset spent. The order is worked out once and
// handed to each of them.
type History []Answer

// Give puts a vault's answers in the order they were given.
func Give(answers []Answer) History { return getGivenAnswers(answers) }

// Replay works out where this history leaves every card face, each under the
// scheduler its own preset asks for and on the day its own preset puts it.
//
// The days the answers have already filled are what the next card is placed
// against.
func (h History) Replay(d Day, by Assignment) map[CardFaceID]Schedule {
	out := make(map[CardFaceID]Schedule)
	on := NewDueByDay(d)
	for _, a := range h {
		one := by(a.CardFace)
		next := one.By.Next(out[a.CardFace], a.At, a.Rating)
		next.Due = one.Preset.Places(on, a.At, next.Due)
		out[a.CardFace] = next
	}
	return out
}

// RecallTally is how much of what came round in days came back, over one day:
// the answers given to spaced card faces, and how many of those came back at
// all.
//
// A card face the scheduler is still putting into memory is in neither. What is
// asked of one is whether it comes back after ten minutes, which says nothing
// about memory.
type RecallTally struct {
	// Asked is the answers given to spaced card faces, and Recalled the ones
	// among them that were not Again.
	Asked    int
	Recalled int
}

// GetRetained is what came back on each day, by the name of the day.
//
// It is worked out with the replay: whether a card face was spaced is a thing
// only the answers before it can say.
func GetRetained(by Scheduler, d Day, answers []Answer) map[string]RecallTally {
	return Give(answers).GetRetained(by, d)
}

// GetRetained is the same over a history already in order.
func (h History) GetRetained(by Scheduler, d Day) map[string]RecallTally {
	out := make(map[string]RecallTally)
	h.walkAnswers(by, func(before Schedule, a Answer) {
		if !by.IsSpaced(before) {
			return
		}
		day := d.GetName(a.At)
		one := out[day]
		one.Asked++
		if a.Rating != Again {
			one.Recalled++
		}
		out[day] = one
	})
	return out
}

// walkAnswers walks the answers in the order they were given, telling each one
// to the caller with the schedule the card face stood at before it.
func walkAnswers(
	by Scheduler, answers []Answer, each func(before Schedule, a Answer),
) map[CardFaceID]Schedule {
	return Give(answers).walkAnswers(by, each)
}

func (h History) walkAnswers(
	by Scheduler, each func(before Schedule, a Answer),
) map[CardFaceID]Schedule {
	out := make(map[CardFaceID]Schedule)
	for _, a := range h {
		before := out[a.CardFace]
		if each != nil {
			each(before, a)
		}
		out[a.CardFace] = by.Next(before, a.At, a.Rating)
	}
	return out
}

// getGivenAnswers is the answers that count, in the order they were given.
//
// An answer some line takes back is left out, and one identifier is one answer
// however many lines carry it. The files arrive in whatever order they were
// synchronised, and what a card face has been through is the order of the
// answers themselves.
func getGivenAnswers(answers []Answer) []Answer {
	taken := make(map[string]bool)
	for _, a := range answers {
		if a.TakesBack() {
			taken[a.Undoes] = true
		}
	}

	seen := make(map[string]bool, len(answers))
	out := make([]Answer, 0, len(answers))
	for _, a := range answers {
		if a.TakesBack() || taken[a.ID] || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		out = append(out, a)
	}
	slices.SortStableFunc(out, byWhen)
	return out
}

// byWhen puts answers in the order they were given.
//
// Two answers of one millisecond are put in the order of their identifiers,
// which is one order and the same at every launch. Which of them was given
// first is not known: an identifier carries the millisecond and then
// randomness.
func byWhen(a, b Answer) int {
	if !a.At.Equal(b.At) {
		return a.At.Compare(b.At)
	}
	return strings.Compare(a.ID, b.ID)
}
