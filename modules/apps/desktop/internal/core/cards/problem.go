package cards

// Problem is something in a stencil or a deck that could not be acted on and
// was not guessed at. It is filed against the file somebody would open to
// settle it, and shown on the card or the face it is about.
type Problem struct {
	Check Check
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

// Check is one thing that can be wrong with a stencil or a deck. The list is
// closed: a new rule is a new name here.
type Check string

const (
	// CheckTwoFields is one name declared twice by a stencil. The first stands.
	CheckTwoFields Check = "two-fields"
	// CheckNoFields is a stencil declaring no field at all. Its cards have no
	// name to stand in a heading, and it cuts nothing.
	CheckNoFields Check = "no-fields"
	// CheckFaceSide is a face with no Front or no Back. It lays out nothing.
	CheckFaceSide Check = "face-side"
	// CheckPlaceholder is a face placing a field the stencil does not declare.
	// The rest of the face is read as usual.
	CheckPlaceholder Check = "placeholder"

	// CheckNoStencil is a card whose first paragraph is not a lone wikilink.
	// Its values are read.
	CheckNoStencil Check = "no-stencil"
	// CheckNotAStencil is a card whose wikilink reaches a note that is not a
	// stencil. Its values are read.
	CheckNotAStencil Check = "not-a-stencil"
	// CheckTwoMarks is one mark carried by two cards of a deck. Both are read
	// and both are shown marked, and neither is given another.
	CheckTwoMarks Check = "two-marks"
	// CheckTwoValues is one card writing a field's heading twice. The first
	// stands.
	CheckTwoValues Check = "two-values"
	// CheckNotWritten is a deck a rename did not reach, so a card holds a
	// heading its stencil no longer declares.
	CheckNotWritten Check = "not-written"
	// CheckTooLarge is a deck over the bound, whose bytes were not read.
	CheckTooLarge Check = "too-large"
)

// OnFile files a problem about a whole file, of no card of it and no face of
// it.
func OnFile(check Check, detail string) Problem {
	return Problem{Check: check, Card: NoPosition, Face: NoPosition, Detail: detail}
}

// OnCard files a problem about one card, counted from the deck's first card.
func OnCard(card int, check Check, detail string) Problem {
	return against(card, check, detail)
}

// against files a problem about one card.
func against(card int, check Check, detail string) Problem {
	return Problem{Check: check, Card: card, Face: NoPosition, Detail: detail}
}

// onFace files a problem about one face.
func onFace(face int, check Check, detail string) Problem {
	return Problem{Check: check, Card: NoPosition, Face: face, Detail: detail}
}

// onField files a problem about one field of a stencil's frontmatter.
func onField(name string, check Check, detail string) Problem {
	return Problem{Check: check, Card: NoPosition, Face: NoPosition, Field: name, Detail: detail}
}
