package format

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// The types a note carries when the cards are made of it: the `type` key of its frontmatter.
const (
	// TypeDeck is a note whose body is cards.
	TypeDeck domain.NoteType = "deck"
	// TypeStencil is a note declaring the fields a card has and the faces it is shown by.
	TypeStencil domain.NoteType = "stencil"
	// TypePreset is a note saying how the decks pointing at it are scheduled.
	TypePreset domain.NoteType = "preset"
)
