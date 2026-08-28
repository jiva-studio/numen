// Package cards reads the two notes a flashcard is made of. A stencil declares
// the fields a card has and the faces it is shown by; a deck holds the cards.
// It is pure: no filesystem, no clock, no database.
//
// A card is markdown and nothing else. The heading carries the structure, so a
// value holds whatever markdown holds short of a heading of the two levels the
// format spends.
package cards

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"

// Stencil is a note declaring what a card has and how it is shown.
type Stencil struct {
	Ref    domain.FileRef
	Fields []string
	// Preamble is whatever the person wrote above the first face, and Tail is
	// what the file ends with once the last side has been read. Both are kept
	// as they were written and neither is any face's.
	Preamble string
	Tail     string
	Faces    []Face
	// Problems are what was wrong with the file and could not be repaired.
	// They are shown to the person, and never guessed at.
	Problems []Problem
}

// Face is one way a card is shown: what stands before the answer and what
// stands after it, each with `{{Field}}` where a value goes. A face with only
// one of the two lays out nothing.
type Face struct {
	Name string
	// Lead is what stands between the face's heading and its first side.
	Lead  string
	Front string
	Back  string
}

// Deck is a note whose body is cards.
type Deck struct {
	Ref domain.FileRef
	// Preamble is whatever the person wrote above the first card, and Tail is
	// what the file ends with once the last value has been read. Both are kept
	// as they were written and neither is any card's.
	Preamble string
	Tail     string
	Cards    []Card
	Problems []Problem
}

// Card is one filled-in set of a stencil's fields.
type Card struct {
	// Name is the heading. It is the value of the stencil's first field, and it
	// is the whole of the card's identity.
	Name string
	// Stencil is the target of the lone wikilink under the heading, as it is
	// written and without its brackets. A card whose first paragraph is not one
	// names no stencil.
	Stencil string
	// Lead is what stands between the wikilink and the first field.
	Lead   string
	Values []Value
}

// Value is what a card holds under one field's heading.
type Value struct {
	Field string
	Text  string
}

// First is the field a card's heading holds the value of. A stencil declaring
// none cuts nothing, and answers with no name.
func (s Stencil) First() string {
	if len(s.Fields) == 0 {
		return ""
	}
	return s.Fields[0]
}

// Value is what a card holds for one of this stencil's fields, and whether it
// holds it. The first field's value is the heading, so a card writing that
// field as a heading of its own is read the heading.
func (s Stencil) Value(c Card, field string) (string, bool) {
	if field != "" && field == s.First() {
		return c.Name, true
	}
	return c.Value(field)
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

// Card is the card of a name, and whether the deck holds it. Two cards of one
// name are a problem against the deck, and the first stands.
func (d Deck) Card(name string) (Card, bool) {
	for _, c := range d.Cards {
		if c.Name == name {
			return c, true
		}
	}
	return Card{}, false
}
