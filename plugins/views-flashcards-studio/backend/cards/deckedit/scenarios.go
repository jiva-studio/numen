package deckedit

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
)

// Read is the deck reading scenario of the cards package.
type Read = cards.Read

// Write is the note writing scenario of the cards package.
type Write = cards.Write

// DeckContents is the parsed contents of a deck.
type DeckContents = cards.DeckContents

// MaxBytes is the largest deck that is read.
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
