package format

// Problem is something in a stencil or a deck that could not be acted on and
// was not guessed at. It is filed against the file somebody would open to
// settle it, and shown on the card or the face it is about.
type Problem struct {
	Fault Fault
	// Card is where the card stands, counted from the first card of the deck,
	// and Face is where the face stands in its stencil. A card is addressed by
	// its position because two cards of one mark is a problem, and there the
	// mark reaches both.
	Card int
	Face int
	// Field is a field's name, which is all a field ever is.
	Field string
	// Detail says what is wrong, in the words the person is shown.
	Detail string
}

// NoPosition is what a problem about no card and no face carries.
const NoPosition = -1

// Fault is one thing that can be wrong with a stencil or a deck. The list is
// closed: a new rule is a new name here.
type Fault string

const (
	// FaultTwoFields is one name declared twice by a stencil. The first stands.
	FaultTwoFields Fault = "two-fields"
	// FaultNoFields is a stencil declaring no field at all. Its cards have
	// nothing to be filled with, and it cuts nothing.
	FaultNoFields Fault = "no-fields"
	// FaultFaceSide is a face with no Front or no Back. It lays out nothing.
	FaultFaceSide Fault = "face-side"
	// FaultPlaceholder is a face placing a field the stencil does not declare.
	// The rest of the face is read as usual.
	FaultPlaceholder Fault = "placeholder"

	// FaultNoStencil is a card whose first paragraph is not a lone wikilink.
	// Its values are read.
	FaultNoStencil Fault = "no-stencil"
	// FaultNotAStencil is a card whose wikilink reaches a note that is not a
	// stencil. Its values are read.
	FaultNotAStencil Fault = "not-a-stencil"
	// FaultTwoMarks is one mark carried by two cards of a deck. Both are read
	// and both are shown marked, and neither is given another.
	FaultTwoMarks Fault = "two-marks"
	// FaultTwoValues is one card writing a field's heading twice. The first
	// stands.
	FaultTwoValues Fault = "two-values"
	// FaultNotWritten is a deck a rename did not reach, so a card holds a
	// heading its stencil no longer declares.
	FaultNotWritten Fault = "not-written"
	// FaultTooLarge is a deck over the bound, whose bytes were not read.
	FaultTooLarge Fault = "too-large"
)

// OnFile files a problem about a whole file, of no card of it and no face of
// it.
func OnFile(fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Face: NoPosition, Detail: detail}
}

// OnCard files a problem about one card, counted from the deck's first card.
func OnCard(card int, fault Fault, detail string) Problem {
	return against(card, fault, detail)
}

// against files a problem about one card.
func against(card int, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: card, Face: NoPosition, Detail: detail}
}

// onFace files a problem about one face.
func onFace(face int, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Face: face, Detail: detail}
}

// onField files a problem about one field of a stencil's frontmatter.
func onField(name string, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Face: NoPosition, Field: name, Detail: detail}
}
