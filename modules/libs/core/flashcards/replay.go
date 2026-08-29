package flashcards

import (
	"slices"
	"strings"
)

// Replay works out where a history leaves every card face it names.
//
// The answers arrive in whatever order the files were read, and the files
// arrive in whatever order they were synchronised, so they are put in the order
// they were given first. A schedule depends on that order: an answer counted
// after a later one leaves a card face somewhere neither of them would have.
//
// An answer some line takes back is left out. Both lines stay in the file —
// nothing here is ever rewritten — and what a person took back is not counted.
//
// One identifier is one answer, however many lines carry it. A synchroniser
// that met a conflict leaves a second copy of a run beside the first, and a
// person restoring a backup puts one there by hand; counting those lines twice
// would double what a card has been through and send it away for longer than it
// was earned.
func Replay(by Scheduler, answers []Answer) map[CardFace]Schedule {
	return replayed(by, answers, nil)
}

// Retention is how much of what a person had learned came back to them, over
// one day: the answers given to cards they had learned, and how many of those
// came back at all.
//
// A card still being learned is not in it. What is asked of one is whether it
// comes back after ten minutes, which says nothing about how well anything is
// remembered.
type Retention struct {
	// Asked is the answers given to cards already learned, and Recalled the
	// ones among them that were not Again.
	Asked    int
	Recalled int
}

// Retained is what came back on each day, by the name of the day.
//
// It is worked out with the replay and not beside it, because whether a card
// was one the person had learned is a thing only the answers before it can say.
func Retained(by Scheduler, d Day, answers []Answer) map[string]Retention {
	out := make(map[string]Retention)
	replayed(by, answers, func(before Schedule, a Answer) {
		if !by.Learned(before) {
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
	taken := make(map[string]bool)
	for _, a := range answers {
		if a.TakesBack() {
			taken[a.Undoes] = true
		}
	}

	seen := make(map[string]bool, len(answers))
	given := make([]Answer, 0, len(answers))
	for _, a := range answers {
		if a.TakesBack() || taken[a.ID] || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		given = append(given, a)
	}
	slices.SortStableFunc(given, byWhen)

	out := make(map[CardFace]Schedule)
	for _, a := range given {
		before := out[a.CardFace]
		if each != nil {
			each(before, a)
		}
		out[a.CardFace] = by.Next(before, a.At, a.Rating)
	}
	return out
}

// byWhen puts answers in the order they were given.
//
// Two answers of one millisecond are put in the order of their identifiers.
// Which of them was given first is not known: an identifier carries the
// millisecond and then randomness, so inside one millisecond it orders by
// chance. What this gives is one order, the same at every launch, which is what
// a schedule worked out again has to have. A person does not answer two cards
// inside a millisecond, so the two are only ever a machine's.
func byWhen(a, b Answer) int {
	if !a.At.Equal(b.At) {
		return a.At.Compare(b.At)
	}
	return strings.Compare(a.ID, b.ID)
}
