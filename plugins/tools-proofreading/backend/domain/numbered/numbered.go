// Package numbered holds the numbered lines of a text, the batches they are sent in, and the digest of a recording.
package numbered

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

// Batch is a group of lines sent to the model in one request.
type Batch struct {
	Number int
	Lines  []Line

	IsJoinable bool

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

// GetLastNumber is the number of the final line the line covers.
func (l Line) GetLastNumber() int {
	if l.IsJoined() {
		return l.Last
	}
	return l.Number
}
