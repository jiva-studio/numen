package service

// Line is one numbered run of words. Last is the final line number of a run put
// together, and Number or zero marks a line standing alone.
type Line struct {
	Number int
	Last   int
	Text   string
}

// IsJoined reports whether the line is a correction that joins more than one
// line.
func (l Line) IsJoined() bool { return l.Last > l.Number }

type Batch struct {
	Number int
	Lines  []Line

	// IsJoinable is whether a reply may answer for a run of lines as one.
	IsJoinable bool

	// Context is the whole text. It is placed before the lines in the prompt
	// and receives no reply.
	Context string
}

// Last is the number of the final line the batch carries, and -1 for none.
func (b Batch) Last() int {
	if len(b.Lines) == 0 {
		return -1
	}
	return b.Lines[len(b.Lines)-1].Number
}

// A line's number is written between these marks in a prompt.
const (
	Opens  = "⟦"
	Closes = "⟧"
)
