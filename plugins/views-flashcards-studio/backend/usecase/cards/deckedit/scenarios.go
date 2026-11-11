package deckedit

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

// Read is the deck reading scenario of the cards package.
type Read = cards.Read

// Write is the note writing scenario of the cards package.
type Write = cards.Write

// DeckContents is the parsed contents of a deck.
type DeckContents = cards.DeckContents

// MaxBytes is the largest deck that is read.
const MaxBytes = cards.MaxBytes

// CardScenarios groups the operations that read and edit the cards of a deck.
type CardScenarios struct {
	GetCardPage     GetCardPage
	AddCard         AddCard
	EditCard        EditCard
	RemoveCardValue RemoveCardValue
	RemoveCard      RemoveCard
}

// NewCardScenarios creates the scenarios that edit the cards of a deck.
func NewCardScenarios(read Read, write Write) CardScenarios {
	return CardScenarios{
		GetCardPage:     NewGetCardPage(read),
		AddCard:         NewAddCard(read, write),
		EditCard:        NewEditCard(read, write),
		RemoveCardValue: NewRemoveCardValue(read, write),
		RemoveCard:      NewRemoveCard(read, write),
	}
}

// SectionScenarios groups the operations that edit the sections of a deck.
type SectionScenarios struct {
	AddSection    AddSection
	RenameSection RenameSection
	RemoveSection RemoveSection
}

// NewSectionScenarios creates the scenarios that edit the sections of a deck.
func NewSectionScenarios(read Read, write Write) SectionScenarios {
	return SectionScenarios{
		AddSection:    NewAddSection(read, write),
		RenameSection: NewRenameSection(read, write),
		RemoveSection: NewRemoveSection(read, write),
	}
}
