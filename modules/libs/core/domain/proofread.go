package domain

// A ProofreadLine is one run of words the recogniser read in one go: the number
// it is known by, and what it says.
//
// A correction may cover a run of lines, and Last is the last of them. A line
// standing on its own has Last equal to Number, or left at zero where nothing
// put a run together.
type ProofreadLine struct {
	Number int
	Last   int
	Text   string
}

// IsJoined says whether this correction puts more than one line together.
func (l ProofreadLine) IsJoined() bool { return l.Last > l.Number }

// A ProofreadBatch is the lines one reply is accepted or refused as a whole: the
// number it is known by, and its lines in the order they are read.
type ProofreadBatch struct {
	Number int
	Lines  []ProofreadLine

	// IsJoinable is whether a reply may answer for a run of these lines as one.
	// Where it does not, a run refuses the batch.
	IsJoinable bool

	// Context is what the whole text holds, said in its own words. It stands
	// before the lines in the question and is answered for by nothing.
	Context string
}

// Last is the number of the final line a batch carries, and -1 for a batch
// carrying none.
func (b ProofreadBatch) Last() int {
	if len(b.Lines) == 0 {
		return -1
	}
	return b.Lines[len(b.Lines)-1].Number
}

// A ProofreadPrompt is the question put to a proofreader about one batch: the
// number of the batch, and the text asked.
type ProofreadPrompt struct {
	Number int
	Text   string
}
