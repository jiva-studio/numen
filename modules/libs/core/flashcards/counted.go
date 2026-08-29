package flashcards

import "time"

// Named is a day as it is written down and read back: the year, the month and
// the day it began on, in the zone the days are counted in.
const Named = "2006-01-02"

// Names the day an answer given at this instant falls in.
//
// It is the day a person would say they answered on, which is the day the
// review day began — an answer given at one in the morning belongs to the day
// before, and is named for it.
func (d Day) Names(at time.Time) string {
	// The day runs to its next boundary, so it began at the one before that,
	// and the date it began on is what a person calls it.
	return d.Ends(at).AddDate(0, 0, -1).Format(Named)
}

// Tally is one day's answers: how many were given, and how each of the four was
// said.
//
// The four are kept apart because a day of fifty cards a person could not
// recall is a different day from fifty they could, and the two are the same
// number.
type Tally struct {
	Answered int
	Again    int
	Hard     int
	Good     int
	Easy     int
}

// Counted is what was answered on each day, by the name of the day.
//
// It is a sum and not an order: an answer arriving from another machine after
// later ones have been counted adds to the day it belongs to and disturbs
// nothing, which is what lets these be worked out one file at a time and kept.
func Counted(d Day, answers []Answer) map[string]Tally {
	taken := make(map[string]bool)
	for _, a := range answers {
		if a.TakesBack() {
			taken[a.Undoes] = true
		}
	}

	seen := make(map[string]bool, len(answers))
	out := make(map[string]Tally)
	for _, a := range answers {
		if a.TakesBack() || taken[a.ID] || seen[a.ID] {
			continue
		}
		seen[a.ID] = true

		day := d.Names(a.At)
		one := out[day]
		one.Answered++
		switch a.Rating {
		case Again:
			one.Again++
		case Hard:
			one.Hard++
		case Good:
			one.Good++
		case Easy:
			one.Easy++
		}
		out[day] = one
	}
	return out
}

// Streak is how many days up to now were reviewed without a gap.
//
// It counts back from the day holding now, and a day that nobody has answered
// on yet does not end a streak: a person who has not sat down this morning has
// not broken anything, and their streak is what they had last night. A day they
// did answer on counts from itself.
func Streak(d Day, days map[string]Tally, now time.Time) int {
	in := d.In
	if in == nil {
		in = time.Local
	}

	at := now.In(in)
	if days[d.Names(at)].Answered == 0 {
		// Today is not answered yet, so the count is of the days behind it.
		at = at.AddDate(0, 0, -1)
	}

	out := 0
	for days[d.Names(at)].Answered > 0 {
		out++
		at = at.AddDate(0, 0, -1)
	}
	return out
}
