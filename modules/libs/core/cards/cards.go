// Package cards reads the two notes a flashcard is made of. A stencil declares
// the fields a card has and the faces it is shown by; a deck holds the cards.
// It is pure: no filesystem, no clock, no database.
//
// A card is markdown and nothing else. The heading carries the structure, so a
// value holds whatever markdown holds short of a heading of the three levels
// the format spends.
package cards

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
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

// CardStencil is a note declaring what a card has and the face templates it is
// shown through.
type CardStencil struct {
	Ref    domain.Fingerprint
	Fields []string
	// Preamble is every byte above the first face, down to the one its heading
	// opens on, and Tail is what the file ends with once the last side has been
	// read. Both are kept as they were written and neither is any face's.
	Preamble string
	Tail     string
	Faces    []CardFaceTemplate
	// Problems are what was wrong with the file and could not be repaired.
	// They are shown to the person, and never guessed at.
	Problems []Problem
}

// CardFaceTemplate is one way a card is shown, as its stencil declares it: what
// stands before the answer and what stands after it, each with `{{Field}}` where
// a value goes. A face with only one of the two lays out nothing.
type CardFaceTemplate struct {
	Name string
	// Lead is what stands between the face's heading and its first side.
	Lead  string
	Front string
	Back  string
}

// Deck is a note whose body is cards.
type Deck struct {
	Ref domain.Fingerprint
	// Preamble is every byte above the first section or card, down to the one
	// its heading opens on, and Tail is what the file ends with once the last
	// value has been read. Both are kept as they were written and neither is
	// any card's.
	Preamble string
	Tail     string
	// Sections are the deck's own, in the order they stand in the file.
	Sections []Section
	Cards    []Card
	Problems []Problem
}

// Section is a run of a deck a person has given a name. It is a name and
// nothing else: no fields, no stencil, no schedule, no mark.
type Section struct {
	Name string
	// Lead is what stands between the section's heading and its first card.
	// Nothing lays it out and nothing reads it: it is a person writing about
	// their own deck, and it is kept exactly as it stands.
	Lead string
}

// NoSection is what a card standing before the first section carries.
const NoSection = -1

// Card is one filled-in set of a stencil's fields.
type Card struct {
	// Heading is what the card's heading shows, with the mark taken off. It is
	// the first line of the card's first field, read back, and it holds nothing
	// of its own.
	Heading string
	// Mark is what the card is, for as long as it exists, and it is empty until
	// the application next writes the deck. It is the ten characters alone: the
	// caret in front of them is how a heading writes one.
	Mark cardid.CardID
	// Stencil is the target of the lone wikilink under the heading, as it is
	// written and without its brackets. A card whose first paragraph is not one
	// names no stencil.
	Stencil string
	// Lead is what stands between the wikilink and the first field.
	Lead string
	// Section is where the section this card stands under stands in the deck's
	// own, counted from the first. A card standing before the first section
	// carries NoSection.
	Section int
	Values  []Value
}

// Value is what a card holds under one field's heading.
type Value struct {
	Field string
	Text  string
}

// First is the field a card's heading holds the value of. A stencil declaring
// none cuts nothing, and answers with no name.
func (s CardStencil) First() string {
	if len(s.Fields) == 0 {
		return ""
	}
	return s.Fields[0]
}

// Value is what the card holds under one field's heading, and whether it holds
// it. Two headings of one name are a problem against the deck, and the first
// stands.
func (c Card) Value(field string) (string, bool) {
	for _, v := range c.Values {
		if v.Field == field {
			return v.Text, true
		}
	}
	return "", false
}

// Card is the card of a mark. A deck holding none of that mark is
// ErrNoSuchCard, and a deck holding two is ErrTwoCards: both are read and both
// are shown, and a machine choosing between them would be choosing which of
// the two a person meant.
func (d Deck) Card(carried cardid.CardID) (Card, error) {
	found := Card{}
	held := false
	for _, c := range d.Cards {
		if c.Mark != carried {
			continue
		}
		if held {
			return Card{}, fmt.Errorf("%w: %s", ErrTwoCards, carried)
		}
		found, held = c, true
	}
	if !held {
		return Card{}, fmt.Errorf("%w: %s", ErrNoSuchCard, carried)
	}
	return found, nil
}
