package domain

// Edit is a change to a note's prose, reported while it is being made, to
// anyone viewing that note.
//
// It is a report, not the change itself: the file is authoritative, and this
// announces what is about to happen to it. A lost report degrades the display,
// not the note.
type Edit struct {
	// Change names one change. Every report of the same change carries the same
	// name, and no two changes carry one.
	Change string
	// Path is the note, relative to the vault folder.
	Path string
	// From and To are the stretch being replaced, counted the way a client counts
	// text: in UTF-16 code units over the prose a read hands out.
	From int
	To   int
	// Text is the replacement for that range.
	Text string
	// IsDone is the last report of this change. It arrives whether the change
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
	From string
	To   string
	// IsLanded is whether the file is at To.
	IsLanded bool
	// Repaired is the notes whose link stopped resolving and was written
	// again, by name.
	Repaired []string
	// Dangling is the notes whose link stopped resolving and could not be
	// written again. Each of them still points at the name the file left.
	Dangling []string
}
