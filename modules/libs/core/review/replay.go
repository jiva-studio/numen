package review

import "slices"

// Replay works out where a history leaves every seat it names.
//
// The answers arrive in whatever order the files were read, and the files
// arrive in whatever order they were synchronised, so they are put in the order
// they were given first. A schedule depends on that order: an answer counted
// after a later one leaves a seat somewhere neither of them would have.
//
// An answer some line takes back is left out. Both lines stay in the file —
// nothing here is ever rewritten — and what a person took back is not counted.
func Replay(by Scheduler, answers []Answer) map[Seat]Schedule {
	taken := make(map[string]bool)
	for _, a := range answers {
		if a.TakesBack() {
			taken[a.Undoes] = true
		}
	}

	given := make([]Answer, 0, len(answers))
	for _, a := range answers {
		if !a.TakesBack() && !taken[a.ID] {
			given = append(given, a)
		}
	}
	slices.SortStableFunc(given, byWhen)

	out := make(map[Seat]Schedule)
	for _, a := range given {
		out[a.Seat] = by.Next(out[a.Seat], a.At, a.Rating)
	}
	return out
}

// byWhen puts answers in the order they were given. Two answers of one instant
// are put in the order of their identifiers, which is the order they were
// minted in: a ULID sorts as text the way it sorts in time.
func byWhen(a, b Answer) int {
	if !a.At.Equal(b.At) {
		return a.At.Compare(b.At)
	}
	return slices.Compare([]byte(a.ID), []byte(b.ID))
}
