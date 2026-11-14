package deckedit

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// CardScenarios groups the operations that read and edit the cards of a deck.
type CardScenarios struct {
	GetCardPage     GetCardPage
	AddCard         AddCard
	EditCard        EditCard
	RemoveCardValue RemoveCardValue
	RemoveCard      RemoveCard
}

func NewCardScenarios(read ports.DeckReader, write ports.DeckWriter) CardScenarios {
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

func NewSectionScenarios(read ports.DeckReader, write ports.DeckWriter) SectionScenarios {
	return SectionScenarios{
		AddSection:    NewAddSection(read, write),
		RenameSection: NewRenameSection(read, write),
		RemoveSection: NewRemoveSection(read, write),
	}
}
