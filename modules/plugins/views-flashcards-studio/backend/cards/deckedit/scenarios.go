package deckedit

import (
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/cards"
)

// Aliases from the parent cards package.
type Read = cards.Read
type Write = cards.Write
type DeckContents = cards.DeckContents

const MaxBytes = cards.MaxBytes

// Scenarios groups the operations that edit the cards and sections of a deck.
type Scenarios struct {
	GetCardPage     GetCardPage
	AddCard         AddCard
	EditCard        EditCard
	RemoveCardValue RemoveCardValue
	RemoveCard      RemoveCard
	AddSection      AddSection
	RenameSection   RenameSection
	RemoveSection   RemoveSection
}

// NewScenarios creates the deck editing scenarios.
func NewScenarios(read Read, write Write) Scenarios {
	return Scenarios{
		GetCardPage:     NewGetCardPage(read),
		AddCard:         NewAddCard(read, write),
		EditCard:        NewEditCard(read, write),
		RemoveCardValue: NewRemoveCardValue(read, write),
		RemoveCard:      NewRemoveCard(read, write),
		AddSection:      NewAddSection(read, write),
		RenameSection:   NewRenameSection(read, write),
		RemoveSection:   NewRemoveSection(read, write),
	}
}
