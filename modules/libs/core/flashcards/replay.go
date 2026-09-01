package flashcards

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
func Replay(d Day, by Scheduler, answers []Answer) map[CardFace]Schedule {
	return ReplayUnder(d, By(by), answers)
}

// Scheduling is how one card face is worked out: the scheduler that spaces it,
// and the preset that says which day it lands on.
type Scheduling struct {
	By     Scheduler
	Preset Preset
}

// Under is how one card face is scheduled. A card face is scheduled by the
// preset its deck points at, and two presets asking for different shares of the
// cards send the same card away for different lengths of time.
type Under func(CardFace) Scheduling

// By is one scheduler for every card face, on the preset a deck naming none is
// scheduled by.
func By(s Scheduler) Under {
	return func(CardFace) Scheduling { return Scheduling{By: s, Preset: Defaults()} }
}

// ReplayUnder works out where a history leaves every card face, each under the
// scheduler its own preset asks for and on the day its own preset puts it.
func ReplayUnder(d Day, by Under, answers []Answer) map[CardFace]Schedule {
	return Give(answers).Replay(d, by)
}

// Given is a vault's answers in the order they were given: nothing a line takes
// back, one line to an identifier, earliest first.
//
// One request asks several things of one history — where it leaves each card,
// what each day came to, what a preset spent. The order is worked out once and
// handed to each of them.
type Given []Answer

// Give puts a vault's answers in the order they were given.
func Give(answers []Answer) Given { return given(answers) }

// Replay works out where this history leaves every card face, each under the
// scheduler its own preset asks for and on the day its own preset puts it.
//
// The days the answers have already filled are what the next card is placed
// against.
func (g Given) Replay(d Day, by Under) map[CardFace]Schedule {
	out := make(map[CardFace]Schedule)
	on := Spreading(d)
	for _, a := range g {
		one := by(a.CardFace)
		next := one.By.Next(out[a.CardFace], a.At, a.Rating)
		next.Due = one.Preset.Places(on, a.At, next.Due)
		out[a.CardFace] = next
	}
	return out
}

// Retention is how much of what came round in days came back, over one day:
// the answers given to spaced card faces, and how many of those came back at
// all.
//
// A card face the scheduler is still putting into memory is in neither. What is
// asked of one is whether it comes back after ten minutes, which says nothing
// about memory.
type Retention struct {
	// Asked is the answers given to spaced card faces, and Recalled the ones
	// among them that were not Again.
	Asked    int
	Recalled int
}

// Retained is what came back on each day, by the name of the day.
//
// It is worked out with the replay: whether a card face was spaced is a thing
// only the answers before it can say.
func Retained(by Scheduler, d Day, answers []Answer) map[string]Retention {
	return Give(answers).Retained(by, d)
}

// Retained is the same over a history already in order.
func (g Given) Retained(by Scheduler, d Day) map[string]Retention {
	out := make(map[string]Retention)
	g.replayed(by, func(before Schedule, a Answer) {
		if !by.Spaced(before) {
			return
		}
		day := d.Names(a.At)
		one := out[day]
		one.Asked++
		if a.Rating != Again {
			one.Recalled++
		}
		out[day] = one
	})
	return out
}

// replayed walks the answers in the order they were given, telling each one to
// the caller with the schedule the card face stood at before it.
func replayed(
	by Scheduler, answers []Answer, each func(before Schedule, a Answer),
) map[CardFace]Schedule {
	return Give(answers).replayed(by, each)
}

func (g Given) replayed(
	by Scheduler, each func(before Schedule, a Answer),
) map[CardFace]Schedule {
	out := make(map[CardFace]Schedule)
	for _, a := range g {
		before := out[a.CardFace]
		if each != nil {
			each(before, a)
		}
		out[a.CardFace] = by.Next(before, a.At, a.Rating)
	}
	return out
}

// given is the answers that count, in the order they were given.
//
// An answer some line takes back is left out, and one identifier is one answer
// however many lines carry it. The files arrive in whatever order they were
// synchronised, and what a card face has been through is the order of the
// answers themselves.
func given(answers []Answer) []Answer {
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
