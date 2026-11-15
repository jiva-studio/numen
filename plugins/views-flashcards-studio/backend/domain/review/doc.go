// Package review stores the answers a user gave to their cards and computes
// when each card is due again.
//
// An answer is written down once and never changed. A schedule is a pure
// function of the answers, so changing how cards are spaced replays the
// history. A projection replays the schedules forward and works out the load
// and the retention a preset gives over the days ahead, and a preset note is a
// preset as a read of its note found it.
//
// It is pure: no filesystem, no clock, no database. What a run is written to
// and where a schedule is kept is the flashcards use case.
package review
