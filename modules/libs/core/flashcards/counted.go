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

// Spent is what a day came to under one preset: how many of its cards were
// answered, and how long those answers took.
//
// The counts are of cards and not of answers, which is the unit the budget is
// kept in: a card that comes round again in the same day is one card, and every
// answer given to it is time spent.
//
// New and Reviews divide the cards the way a budget does: a card is new on the
// day its face is first answered at all, and a review on every day after.
type Spent struct {
	Answered int
	New      int
	Reviews  int
	Took     time.Duration
}

// Sat is what the day named came to under each preset, by the path the card
// faces are grouped under.
//
// A card face nothing groups is left out, and so is an answer taken back. One
// answer counts LongestAnswer at most: a card left on the screen while a person
// answered the door stands there for an hour, and the hour is not review.
//
// Counts says how each preset counts, by the same path, and a path it does not
// name counts in cards: a card face counts once for the day however many times
// it is answered in it. Every one of those answers counts its time either way.
func Sat(
	d Day, day string, answers []Answer,
	under map[CardFace]string, counts map[string]Counts,
) map[string]Spent {
	return Give(answers).Sat(d, day, under, counts)
}

// Sat is the same over a history already in order.
func (g Given) Sat(
	d Day, day string, under map[CardFace]string, counts map[string]Counts,
) map[string]Spent {
	// Which card faces have been answered before the answer in hand, over the
	// whole history and not this day alone.
	before := make(map[CardFace]bool, len(g))
	// Which card faces the day has already counted, so a card that comes round
	// again in it is still the one card.
	counted := make(map[CardFace]bool)
	out := make(map[string]Spent)
	for _, a := range g {
		first := !before[a.CardFace]
		before[a.CardFace] = true
		if d.Names(a.At) != day {
			continue
		}
		path, held := under[a.CardFace]
		if !held {
			continue
		}
		one := out[path]
		if counts[path] == CountsShows || !counted[a.CardFace] {
			counted[a.CardFace] = true
			one.Answered++
			if first {
				one.New++
			} else {
				one.Reviews++
			}
		}
		if took := min(a.Took, LongestAnswer); took > 0 {
			one.Took += took
		}
		out[path] = one
	}
	return out
}

// Faced is the card faces answered in the day named. An answer taken back is
// not one, and a card face answered again in the day is the one face.
func Faced(d Day, day string, answers []Answer) map[CardFace]bool {
	return Give(answers).Faced(d, day)
}

// Faced is the same over a history already in order.
func (g Given) Faced(d Day, day string) map[CardFace]bool {
	out := make(map[CardFace]bool)
	for _, a := range g {
		if d.Names(a.At) == day {
			out[a.CardFace] = true
		}
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
	at := now.In(d.zone())
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
