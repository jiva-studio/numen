// Package format reads the two notes a flashcard is made of. A card model declares
// the fields a card has and the templates it is shown by; a deck holds the cards.
// It is pure: no filesystem, no clock, no database.
//
// A card is HTML. The file it is written in is cut at three heading levels, so
// a value holds whatever HTML holds short of a heading of those three; one
// wanting a heading writes it as <h4> or below.
package format

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/cardid"
)

// The three heading levels a deck spends. A value may hold a heading below
// them and none of them, and every reader of a deck — this package and the
// index that records what a deck holds — reads a level from here.
const (
	// SectionLevel opens a section, whose cards are its own until the next one.
	SectionLevel = 1
	// CardLevel opens a card, which runs to the next card, the next section or
	// the end of the file.
	CardLevel = 2
	// FieldLevel opens one of a card's fields.
	FieldLevel = 3
)

// CardModel is a note declaring what a card has and the card templates it is
// shown through.
type CardModel struct {
	Ref       domain.FileStat
	Fields    []string
	Preamble  string
	Tail      string
	Templates []CardTemplate
	Problems  []Problem
}

// CardTemplate is one way a card is displayed, as its card model declares it: the
// content before the answer and the content after it, each with `{{Field}}` where a
// value goes. A template with only one of the two lays out nothing.
type CardTemplate struct {
	Name     string
	Preamble string
	Front    string
	Back     string
}

// Deck is a note whose body is cards.
type Deck struct {
	Ref      domain.FileStat
	Preamble string
	Tail     string
	Sections []Section
	Cards    []Card
	Problems []Problem
}

// Section is a named run of a deck. It is a name and nothing else: no fields, no
// card model, no schedule, no card id.
type Section struct {
	Name     string
	Preamble string
}

// NoSection is what a card standing before the first section carries.
const NoSection = -1

// Card is one filled-in set of a card model's fields.
type Card struct {
	Heading       string
	CardID        cardid.ID
	CardModelLink string
	Preamble      string
	Section       int
	Values        []Value
}

// Value is what a card holds under one field's heading.
type Value struct {
	Field string
	Text  string
}

// First is the field a card's heading holds the value of. A card model declaring
// none cuts nothing, and answers with no name.
func (s CardModel) First() string {
	if len(s.Fields) == 0 {
		return ""
	}
	return s.Fields[0]
}

// Value is what the card holds under one field's heading, and whether it holds
// it. Two headings of one name are a problem against the deck, and the first is
// used.
func (c Card) Value(field string) (string, bool) {
	for _, v := range c.Values {
		if v.Field == field {
			return v.Text, true
		}
	}
	return "", false
}

// Card is the card with the given card id. A deck holding none of that card id
// yields ErrNoSuchCard, and a deck holding two yields ErrTwoCards: both are
// reported, and choosing automatically between them would guess which one the
// author meant.
func (d Deck) Card(carried cardid.ID) (Card, error) {
	found := Card{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial found Card is overwritten before Card returns on success."
		Heading: "", CardID: "", CardModelLink: "", Preamble: "", Section: 0, Values: nil,
	}
	target := false
	for _, c := range d.Cards {
		if c.CardID != carried {
			continue
		}
		if target {
			return Card{}, fmt.Errorf("%w: %s", ErrTwoCards, carried)
		}
		found, target = c, true
	}
	if !target {
		return Card{}, fmt.Errorf("%w: %s", ErrNoSuchCard, carried)
	}
	return found, nil
}
