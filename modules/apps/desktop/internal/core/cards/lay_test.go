package cards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
)

// declaring is a stencil of nothing but its fields, which is what laying a face
// out needs of one.
func declaring(fields ...string) cards.Stencil {
	return cards.Stencil{Fields: fields}
}

func TestLayFillsAFaceWithACard(t *testing.T) {
	face := cards.Face{
		Name:  "Recognise",
		Front: "![[llama.jpg]]",
		Back:  "**{{Name}}** is {{Height}} and lives {{Life span}}.",
	}
	card := cards.Card{
		Name: "Llama",
		Values: []cards.Value{
			{Field: "Height", Text: `about 45"`},
			{Field: "Life span", Text: "about 20 years"},
		},
	}

	front, back := cards.Lay(declaring("Name", "Height", "Life span"), face, card)
	if front != "![[llama.jpg]]" {
		t.Errorf("front = %q, want the markdown around a placeholder untouched", front)
	}
	if want := `**Llama** is about 45" and lives about 20 years.`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// The first field is placed by its own name, and what it lays out is the card's
// heading.
func TestTheFirstFieldLaysOutTheHeading(t *testing.T) {
	card := cards.Card{Name: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}
	face := cards.Face{Front: "{{Name}}", Back: "{{Height}}"}

	front, _ := cards.Lay(declaring("Name", "Height"), face, card)
	if front != "Llama" {
		t.Errorf("front = %q, want the heading", front)
	}

	// Which field that is, is the one standing first, so a stencil ordered
	// another way lays the heading out under another name.
	front, back := cards.Lay(declaring("Height", "Name"), face, card)
	if front != "" {
		t.Errorf("front = %q, want nothing: the card has no Name of its own", front)
	}
	if back != "Llama" {
		t.Errorf("back = %q, want the heading under the field that stands first now", back)
	}
}

// The heading is what the first field holds, so a card writing that field as a
// heading of its own is laid out the heading.
func TestTheHeadingBeatsAFieldOfTheSameName(t *testing.T) {
	card := cards.Card{Name: "Llama", Values: []cards.Value{
		{Field: "Name", Text: "written a second time"},
	}}

	front, _ := cards.Lay(declaring("Name"), cards.Face{Front: "{{Name}}", Back: "-"}, card)
	if front != "Llama" {
		t.Errorf("front = %q, want the heading", front)
	}
}

// A field the card leaves out lays out as nothing, and so does one the stencil
// never declared.
func TestAPlaceholderWithNothingBehindItLaysOutAsNothing(t *testing.T) {
	face := cards.Face{Front: "{{Name}}", Back: "[{{Height}}][{{Weight}}]"}
	card := cards.Card{Name: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	front, back := cards.Lay(declaring("Name", "Height"), face, card)
	if front != "Llama" {
		t.Errorf("front = %q", front)
	}
	if back != `[45"][]` {
		t.Errorf("back = %q", back)
	}
}

// The name inside the braces is a field's name written exactly: a space is part
// of it, and so is every letter of every alphabet.
func TestAPlaceholderIsANameWrittenExactly(t *testing.T) {
	card := cards.Card{
		Name: "яблоня",
		Values: []cards.Value{
			{Field: "Life span", Text: "20"},
			{Field: "Жизнь", Text: "двадцать"},
			{Field: "Height", Text: `45"`},
		},
	}
	s := declaring("Слово", "Life span", "Жизнь", "Height")

	_, back := cards.Lay(s, cards.Face{
		Front: "{{Слово}}", Back: "{{Life span}} {{Жизнь}} {{ Height }} {{Слово}}",
	}, card)
	if want := "20 двадцать  яблоня"; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// Nothing escapes the braces. A person writing them in prose gets a
// placeholder, and this is what that looks like.
func TestBracesAreNotEscaped(t *testing.T) {
	card := cards.Card{Name: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	_, back := cards.Lay(declaring("Name", "Height"),
		cards.Face{Front: "{{Name}}", Back: `\{{Height}} and {{{Height}}}`}, card)
	if want := `\45" and {45"}`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A value is markdown, and it lays out as the person wrote it.
func TestAValueOfSeveralLinesLaysOutWhole(t *testing.T) {
	card := cards.Card{Name: "Llama", Values: []cards.Value{
		{Field: "Height", Text: "#### At the shoulder\n\nabout 45\""},
	}}

	_, back := cards.Lay(declaring("Name", "Height"),
		cards.Face{Front: "{{Name}}", Back: "> {{Height}}"}, card)
	if want := "> #### At the shoulder\n\nabout 45\""; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A stencil declaring no field cuts nothing: there is no name a heading is the
// value of, and every placeholder stands for a field nobody declared.
func TestAStencilOfNoFieldsLaysNothingOut(t *testing.T) {
	card := cards.Card{Name: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	front, back := cards.Lay(declaring(), cards.Face{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `45"` {
		t.Errorf("back = %q, want the value the card does hold", back)
	}
}
