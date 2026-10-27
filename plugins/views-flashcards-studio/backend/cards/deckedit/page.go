package deckedit

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// Page is a run of a deck's cards, with the deck they stand in. Total is how many cards the deck holds.
type Page struct {
	Cards    []format.Card
	Sections []string
	Total    int
	Problems []format.Problem
	FileStat domain.FileStat
}
