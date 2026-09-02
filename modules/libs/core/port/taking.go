package port

// Taking is what became of a run over a source somebody named.
//
// A source a person named is never refused for want of a turn: the run begins
// where nothing of its kind is going, and otherwise the source waits in line
// and is taken up as soon as the turn is free.
type Taking int

const (
	// Began is a run over the named source, started now.
	Began Taking = iota
	// Queued is a named source waiting behind the run already going.
	Queued
)
