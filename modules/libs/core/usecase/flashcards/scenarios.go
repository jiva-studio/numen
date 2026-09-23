package flashcards

import "github.com/jiva-studio/numen/modules/libs/core/flashcards/review"

// Scenarios is everything that runs a vault's cards: what stands in it, what it
// owes, what to ask next, and what an answer is written to.
type Scenarios struct {
	CardFaces ListCardFaces
	Marks     MarkCards
	Schedules Schedules
	CardsDue  CountCardsDue
	Session   Session
	Log       Log
	// Counted is how much of a vault was answered on each day it was reviewed.
	Counted CountReviews
	// Presets is which preset each deck is scheduled by, and how one is read,
	// written and made.
	Presets Presets
	// Curves is what the one control of a preset comes to over the whole range
	// of its goal.
	Curves ProjectCurve
	// Day is where one day of review gives way to the next.
	Day review.Day
}
