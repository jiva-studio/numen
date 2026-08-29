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
		out[a.CardFace] = by.Next(out[a.CardFace], a.At, a.Rating)
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
