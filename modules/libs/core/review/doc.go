// Package review keeps the answers a person gave their cards and works out
// when each comes round again.
//
// An answer is written down once and never changed. A schedule is a pure
// function of the answers, so changing how cards are spaced replays the history
// rather than converting a state that has no honest conversion.
//
// It is pure: no filesystem, no clock, no database. What a run is written to
// and where a schedule is kept is the review use case.
package review
