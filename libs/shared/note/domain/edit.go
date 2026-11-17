package domain

// Edit is a change to a note's prose, reported while it is being made.
// It is a report, not the change: the file is authoritative.
// A lost report degrades the display, not the note.
type Edit struct {
	Change string
	Path   string
	From   int
	To     int
	Text   string
	IsDone bool
}

// Move is a note that has changed path, with its new path.
//
// The content does not change, so this carries no information about the note's
// content; it records only that the path changed.
type Move struct {
	From string
	To   string
}

// MoveResult reports where the note went and the effect on the links that
// pointed at it.
type MoveResult struct {
	From     string
	To       string
	IsMoved  bool
	Repaired []string
	Dangling []string
}
