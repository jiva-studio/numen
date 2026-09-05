package domain

// Edit is a change to a note's prose as it is being made, for whoever is
// looking at that note while it happens.
//
// It is a report and never the change itself: the file is what the note says,
// and this only says what is about to happen to it. A report that never arrives
// costs a drawing, not a note.
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
	// Text is what is going in where that stretch stands.
	Text string
	// Done is the last report of this change. It arrives whether the change
	// landed or was refused, and it is what ends the drawing.
	Done bool
}
