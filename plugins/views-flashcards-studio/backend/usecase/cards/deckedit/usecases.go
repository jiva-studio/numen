package deckedit

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

// Read is the deck reading scenario of the cards package.
type Read = cards.Read

// Write is the note writing scenario of the cards package.
type Write = cards.Write

type DeckContents = cards.DeckContents

// MaxBytes is the largest deck that is read.
const MaxBytes = cards.MaxBytes

// CardUseCases groups the operations that read and edit the cards of a deck.
type CardUseCases struct {
	GetCardPage     GetCardPage
	AddCard         AddCard
	EditCard        EditCard
	RemoveCardValue RemoveCardValue
	RemoveCard      RemoveCard
}

func NewCardUseCases(read Read, write Write) CardUseCases {
	return CardUseCases{
		GetCardPage:     NewGetCardPage(read),
		AddCard:         NewAddCard(read, write),
		EditCard:        NewEditCard(read, write),
		RemoveCardValue: NewRemoveCardValue(read, write),
		RemoveCard:      NewRemoveCard(read, write),
	}
}

// SectionUseCases groups the operations that edit the sections of a deck.
type SectionUseCases struct {
	AddSection    AddSection
	RenameSection RenameSection
	RemoveSection RemoveSection
}

func NewSectionUseCases(read Read, write Write) SectionUseCases {
	return SectionUseCases{
		AddSection:    NewAddSection(read, write),
		RenameSection: NewRenameSection(read, write),
		RemoveSection: NewRemoveSection(read, write),
	}
}
