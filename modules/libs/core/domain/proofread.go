package domain

// ProofreadKind is what is being proofread.
type ProofreadKind int

const (
	// ProofreadScan is text a machine read off the pages of a printed book.
	ProofreadScan ProofreadKind = iota + 1
	// ProofreadSpeech is text a machine heard in a recording.
	ProofreadSpeech
)

// A ProofreadLine is one run of words and the number it is known by.
type ProofreadLine struct {
	Number int
	Text   string
}

// A ProofreadGroup is lines answered for as a whole. A scan group is one page
// and a speech request holds the whole recording as its one group.
type ProofreadGroup struct {
	Number int
	Lines  []ProofreadLine
}

// A ProofreadRequest is what a proofreader is asked to put right.
type ProofreadRequest struct {
	Kind   ProofreadKind
	Groups []ProofreadGroup

	// Size and Overlap say how many lines of speech one model request holds and
	// how many it shares with the one before it. FromLine is the first line to
	// answer and LineLimit how many lines at most, where that is above zero.
	Size, Overlap, FromLine, LineLimit int

	// MaxEditDistance is how far a correction of one line may stand from the
	// line as read. Zero takes the proofreader's own limit.
	MaxEditDistance float64
}

// A ProofreadCorrection puts a line right. Last is the final line of a run put
// together as one, and is not above Number for a line standing on its own.
type ProofreadCorrection struct {
	Number int
	Last   int
	Text   string
}

// IsJoined says whether the correction puts more than one line together.
func (c ProofreadCorrection) IsJoined() bool { return c.Last > c.Number }

// A ProofreadAnswer is what a proofreader put right.
type ProofreadAnswer struct {
	Corrections []ProofreadCorrection

	// Uncorrected is how many groups were answered with something that is no
	// correction. Their lines stand as they were read.
	Uncorrected int

	// NextLine is the line a speech request continues from, and IsDone says
	// nothing is left after the lines answered.
	NextLine int
	IsDone   bool
}
