package port

// StartOutcome is how a start over a source somebody named ended. What the run
// itself came to is said later, over the work it is listed under.
//
// A source a person named is never refused for want of a turn: the run begins
// where nothing of its kind is going, and otherwise the source waits in line
// and is taken up as soon as the turn is free.
type StartOutcome int

const (
	// Began is a run over the named source, started now.
	Began StartOutcome = iota
	// Queued is a named source waiting behind the run already going.
	Queued
)
