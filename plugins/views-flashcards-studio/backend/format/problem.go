package format

// Problem is something in a card model or a deck that could not be acted on and
// was not guessed at. It is filed against the file somebody would open to
// settle it, and shown on the card or the template it is about.
type Problem struct {
	Fault Fault
	// Card is where the card stands, counted from the first card of the deck,
	// its position because two cards of one card id is a problem, and there the
	// card id reaches both.
	Card     int
	Template int
	// Field is a field's name, which is all a field ever is.
	Field string
	// Detail says what is wrong, in the words the person is shown.
	Detail string
}

// NoPosition is what a problem about no card and no template carries.
const NoPosition = -1

// Fault is one thing that can be wrong with a card model or a deck. The list is
// closed: a new rule is a new name here.
type Fault string

const (
	// FaultTwoFields is one name declared twice by a card model. The first stands.
	FaultTwoFields Fault = "two-fields"
	// FaultNoFields is a card model declaring no field at all. Its cards have
	// nothing to be filled with, and it cuts nothing.
	FaultNoFields Fault = "no-fields"
	// FaultTemplateSide is a template with no Front or no Back. It lays out nothing.
	FaultTemplateSide Fault = "template-side"
	// FaultPlaceholder is a template placing a field the card model does not declare.
	// The rest of the template is read as usual.
	FaultPlaceholder Fault = "placeholder"

	// FaultNoCardModel is a card whose first paragraph is not a lone wikilink.
	// Its values are read.
	FaultNoCardModel Fault = "no-card-model"
	// FaultNotACardModel is a card whose wikilink reaches a note that is not a
	// card model. Its values are read.
	FaultNotACardModel Fault = "not-a-card-model"
	// FaultDuplicateCardID is one card id carried by two cards of a deck. Both are read
	// and both are shown marked, and neither is given another.
	FaultDuplicateCardID Fault = "duplicate-card-id"
	// FaultTwoValues is one card writing a field's heading twice. The first
	// stands.
	FaultTwoValues Fault = "two-values"
	// FaultNotWritten is a deck a rename did not reach, so a card holds a
	// heading its card model no longer declares.
	FaultNotWritten Fault = "not-written"
	// FaultTooLarge is a deck over the bound, whose bytes were not read.
	FaultTooLarge Fault = "too-large"
)

// OnFile files a problem about a whole file, of no card of it and no template of
// it.
func OnFile(fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Template: NoPosition, Detail: detail}
}

// OnCard files a problem about one card, counted from the deck's first card.
func OnCard(card int, fault Fault, detail string) Problem {
	return newCardProblem(card, fault, detail)
}

// newCardProblem files a problem about one card.
func newCardProblem(card int, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: card, Template: NoPosition, Detail: detail}
}

// onTemplate files a problem about one template.
func onTemplate(template int, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Template: template, Detail: detail}
}

// onField files a problem about one field of a card model's frontmatter.
func onField(name string, fault Fault, detail string) Problem {
	return Problem{Fault: fault, Card: NoPosition, Template: NoPosition, Field: name, Detail: detail}
}
