package cards

import (
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Page is a run of a deck's cards, with the deck they stand in. Total is how many cards the deck holds.
type Page struct {
	Cards       []format.Card
	Sections    []string
	Total       int
	Problems    []format.Problem
	Fingerprint domain.Fingerprint
}
