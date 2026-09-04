package cards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/cards"
)

// declaring is a stencil of nothing but its fields, which is what laying a face
// out needs of one.
func declaring(fields ...string) cards.Stencil {
	return cards.Stencil{Fields: fields}
}

func TestLayFillsAFaceWithACard(t *testing.T) {
	face := cards.FaceTemplate{
		Name:  "Recognise",
		Front: "![[llama.jpg]]",
		Back:  "**{{Name}}** is {{Height}} and lives {{Life span}}.",
	}
	card := cards.Card{
		Heading: "Llama",
		Values: []cards.Value{
			{Field: "Name", Text: "Llama"},
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

// The first field is a field like any other: it is laid out from what the card
// holds under its heading, and the card's own heading lays out nothing.
func TestTheFirstFieldLaysOutWhatTheCardHolds(t *testing.T) {
	card := cards.Card{
		Heading: "Llama, cut short",
		Values: []cards.Value{
			{Field: "Name", Text: "Llama, and everything the heading had no room for"},
			{Field: "Height", Text: `45"`},
		},
	}
	face := cards.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}

	front, _ := cards.Lay(declaring("Name", "Height"), face, card)
	if front != "Llama, and everything the heading had no room for" {
		t.Errorf("front = %q, want the field and not the heading", front)
	}

	// Which field stands first changes nothing about how one is laid out.
	front, back := cards.Lay(declaring("Height", "Name"), face, card)
	if front != "Llama, and everything the heading had no room for" || back != `45"` {
		t.Errorf("front = %q, back = %q", front, back)
	}
}

// A card the deck holds no first field for lays that field out as nothing, and
// its heading is not put there in its place.
func TestACardWithNoFirstFieldLaysNothingOutForIt(t *testing.T) {
	card := cards.Card{Heading: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	front, _ := cards.Lay(declaring("Name", "Height"),
		cards.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
}

// A field the card leaves out lays out as nothing, and so does one the stencil
// never declared.
func TestAPlaceholderWithNothingBehindItLaysOutAsNothing(t *testing.T) {
	face := cards.FaceTemplate{Front: "{{Name}}", Back: "[{{Height}}][{{Weight}}]"}
	card := cards.Card{Heading: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	front, back := cards.Lay(declaring("Name", "Height"), face, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `[45"][]` {
		t.Errorf("back = %q", back)
	}
}

// The name inside the braces is a field's name written exactly: a space is part
// of it, and so is every letter of every alphabet.
func TestAPlaceholderIsANameWrittenExactly(t *testing.T) {
	card := cards.Card{
		Heading: "яблоня",
		Values: []cards.Value{
			{Field: "Life span", Text: "20"},
			{Field: "Жизнь", Text: "двадцать"},
			{Field: "Height", Text: `45"`},
		},
	}
	s := declaring("Слово", "Life span", "Жизнь", "Height")

	_, back := cards.Lay(s, cards.FaceTemplate{
		Front: "{{Слово}}", Back: "{{Life span}} {{Жизнь}} {{ Height }} {{Слово}}",
	}, card)
	if want := "20 двадцать  "; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// Nothing escapes the braces. A person writing them in prose gets a
// placeholder, and this is what that looks like.
func TestBracesAreNotEscaped(t *testing.T) {
	card := cards.Card{Heading: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	_, back := cards.Lay(declaring("Name", "Height"),
		cards.FaceTemplate{Front: "{{Name}}", Back: `\{{Height}} and {{{Height}}}`}, card)
	if want := `\45" and {45"}`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A value is markdown, and it lays out as the person wrote it.
func TestAValueOfSeveralLinesLaysOutWhole(t *testing.T) {
	card := cards.Card{Heading: "Llama", Values: []cards.Value{
		{Field: "Height", Text: "#### At the shoulder\n\nabout 45\""},
	}}

	_, back := cards.Lay(declaring("Name", "Height"),
		cards.FaceTemplate{Front: "{{Name}}", Back: "> {{Height}}"}, card)
	if want := "> #### At the shoulder\n\nabout 45\""; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A stencil declaring no field cuts nothing: every placeholder on it stands for
// a field nobody declared.
func TestAStencilOfNoFieldsLaysNothingOut(t *testing.T) {
	card := cards.Card{Heading: "Llama", Values: []cards.Value{{Field: "Height", Text: `45"`}}}

	front, back := cards.Lay(declaring(), cards.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `45"` {
		t.Errorf("back = %q, want the value the card does hold", back)
	}
}
