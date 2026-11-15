package deckedit

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// CardUseCases groups the operations that read and edit the cards of a deck.
type CardUseCases struct {
	GetCardPage     GetCardPage
	AddCard         AddCard
	EditCard        EditCard
	RemoveCardValue RemoveCardValue
	RemoveCard      RemoveCard
}

func NewCardUseCases(read ports.DeckReader, write ports.DeckWriter) CardUseCases {
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

func NewSectionUseCases(read ports.DeckReader, write ports.DeckWriter) SectionUseCases {
	return SectionUseCases{
		AddSection:    NewAddSection(read, write),
		RenameSection: NewRenameSection(read, write),
		RemoveSection: NewRemoveSection(read, write),
	}
}
