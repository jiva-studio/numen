package format_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
)

// A card face is shown on two surfaces and each lays it out for itself: this
// package for the window a card is reviewed in, and modules/libs/ui for the
// preview beside the stencil being written. The table both are held to is one
// file, and neither owns it.
const corpus = "../../../protocol/testdata/faces.json"

// table is the corpus as it is written down: what it says of itself, and the
// faces.
type table struct {
	Invariant string   `json:"invariant"`
	Count     int      `json:"count"`
	Changed   int      `json:"changed"`
	Faces     []laying `json:"faces"`
}

// laying is one face of the table: what it is called, the construct it pins,
// and what a surface must lay it out as.
type laying struct {
	Name      string   `json:"name"`
	Construct string   `json:"construct"`
	Face      string   `json:"face"`
	Fields    []string `json:"fields"`
	Values    []filled `json:"values"`
	Laid      string   `json:"laid"`
}

type filled struct {
	Field string `json:"field"`
	Text  string `json:"text"`
}

// A card face means the same thing on every surface it is shown on. Laying it
// out is the step each surface takes for itself; drawing what comes out is one
// implementation both of them reach, so what is compared here is the text the
// slots have been filled in, and text that agrees is drawn alike.
func TestACardFaceMeansTheSameOnEverySurfaceItIsShownOn(t *testing.T) {
	read := corpusOf(t)

	laid, changed := 0, 0
	for _, one := range read.Faces {
		card := format.Card{Heading: "Llama"}
		for _, value := range one.Values {
			card.Values = append(card.Values, format.Value{Field: value.Field, Text: value.Text})
		}

		face := format.FaceTemplate{Name: one.Name, Front: one.Face, Back: one.Face}
		front, back := format.Lay(declaring(one.Fields...), face, card)
		laid++
		if one.Laid != one.Face {
			changed++
		}

		if front != one.Laid {
			t.Errorf("%s — %s\nthe face  %q\nlays out  %q\nthe table %q",
				one.Name, one.Construct, one.Face, front, one.Laid)
		}
		if back != front {
			t.Errorf("%s — %s: the two sides of one face lay out as %q and %q",
				one.Name, one.Construct, front, back)
		}
	}

	if laid != read.Count {
		t.Fatalf("the table declares %d faces and %d were laid out", read.Count, laid)
	}
	if changed != read.Changed {
		t.Fatalf("the table declares %d faces the slots change and %d of them differ from the face",
			read.Changed, changed)
	}
}

// corpusOf reads the table and refuses one that would pass without asking
// anything: a file that is empty, that declares no faces, that says nothing of
// what it is for, or that names one face twice.
func corpusOf(t *testing.T) table {
	t.Helper()

	raw, err := os.ReadFile(corpus)
	if err != nil {
		t.Fatal(err)
	}
	var read table
	if err := json.Unmarshal(raw, &read); err != nil {
		t.Fatal(err)
	}
	if read.Invariant == "" {
		t.Fatal("the table names no invariant")
	}
	if len(read.Faces) == 0 || read.Count == 0 || read.Changed == 0 {
		t.Fatalf("the table holds %d faces and declares %d, %d of which the slots change",
			len(read.Faces), read.Count, read.Changed)
	}
	seen := map[string]bool{}
	for _, one := range read.Faces {
		if one.Name == "" || one.Construct == "" {
			t.Fatalf("a face of the table is called %q and pins %q", one.Name, one.Construct)
		}
		if seen[one.Name] {
			t.Fatalf("two faces of the table are called %q", one.Name)
		}
		seen[one.Name] = true
	}
	return read
}

// declaring is a stencil of nothing but its fields, which is what laying a face
// out needs of one.
func declaring(fields ...string) format.Stencil {
	return format.Stencil{Fields: fields}
}

func TestLayFillsAFaceWithACard(t *testing.T) {
	face := format.FaceTemplate{
		Name:  "Recognise",
		Front: "![[llama.jpg]]",
		Back:  "**{{Name}}** is {{Height}} and lives {{Life span}}.",
	}
	card := format.Card{
		Heading: "Llama",
		Values: []format.Value{
			{Field: "Name", Text: "Llama"},
			{Field: "Height", Text: `about 45"`},
			{Field: "Life span", Text: "about 20 years"},
		},
	}

	front, back := format.Lay(declaring("Name", "Height", "Life span"), face, card)
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
	card := format.Card{
		Heading: "Llama, cut short",
		Values: []format.Value{
			{Field: "Name", Text: "Llama, and everything the heading had no room for"},
			{Field: "Height", Text: `45"`},
		},
	}
	face := format.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}

	front, _ := format.Lay(declaring("Name", "Height"), face, card)
	if front != "Llama, and everything the heading had no room for" {
		t.Errorf("front = %q, want the field and not the heading", front)
	}

	// Which field stands first changes nothing about how one is laid out.
	front, back := format.Lay(declaring("Height", "Name"), face, card)
	if front != "Llama, and everything the heading had no room for" || back != `45"` {
		t.Errorf("front = %q, back = %q", front, back)
	}
}

// A card the deck holds no first field for lays that field out as nothing, and
// its heading is not put there in its place.
func TestACardWithNoFirstFieldLaysNothingOutForIt(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, _ := format.Lay(declaring("Name", "Height"),
		format.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
}

// A field the card leaves out lays out as nothing, and so does one the stencil
// never declared.
func TestAPlaceholderWithNothingBehindItLaysOutAsNothing(t *testing.T) {
	face := format.FaceTemplate{Front: "{{Name}}", Back: "[{{Height}}][{{Weight}}]"}
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, back := format.Lay(declaring("Name", "Height"), face, card)
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
	card := format.Card{
		Heading: "яблоня",
		Values: []format.Value{
			{Field: "Life span", Text: "20"},
			{Field: "Жизнь", Text: "двадцать"},
			{Field: "Height", Text: `45"`},
		},
	}
	s := declaring("Слово", "Life span", "Жизнь", "Height")

	_, back := format.Lay(s, format.FaceTemplate{
		Front: "{{Слово}}", Back: "{{Life span}} {{Жизнь}} {{ Height }} {{Слово}}",
	}, card)
	if want := "20 двадцать  "; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// Nothing escapes the braces. A person writing them in prose gets a
// placeholder, and this is what that looks like.
func TestBracesAreNotEscaped(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	_, back := format.Lay(declaring("Name", "Height"),
		format.FaceTemplate{Front: "{{Name}}", Back: `\{{Height}} and {{{Height}}}`}, card)
	if want := `\45" and {45"}`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A value is markdown, and it lays out as the person wrote it.
func TestAValueOfSeveralLinesLaysOutWhole(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{
		{Field: "Height", Text: "#### At the shoulder\n\nabout 45\""},
	}}

	_, back := format.Lay(declaring("Name", "Height"),
		format.FaceTemplate{Front: "{{Name}}", Back: "> {{Height}}"}, card)
	if want := "> #### At the shoulder\n\nabout 45\""; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A stencil declaring no field cuts nothing: every placeholder on it stands for
// a field nobody declared.
func TestAStencilOfNoFieldsLaysNothingOut(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, back := format.Lay(declaring(), format.FaceTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `45"` {
		t.Errorf("back = %q, want the value the card does hold", back)
	}
}
