package format_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// A review card is shown on two surfaces and each lays it out for itself: this
// package for the window a card is reviewed in, and modules/libs/ui/core for the
// file, and neither owns it.
const corpus = "../../frontend/testdata/templates.json"

// table is the corpus as written: its own description, and the templates.
type table struct {
	Invariant string   `json:"invariant"`
	Count     int      `json:"count"`
	Changed   int      `json:"changed"`
	Templates []laying `json:"templates"`
}

// laying is one template of the table: what it is called, the construct it pins,
// and what a surface must lay it out as.
type laying struct {
	Name      string   `json:"name"`
	Construct string   `json:"construct"`
	Template  string   `json:"template"`
	Fields    []string `json:"fields"`
	Values    []filled `json:"values"`
	Laid      string   `json:"laid"`
}

type filled struct {
	Field string `json:"field"`
	Text  string `json:"text"`
}

// A review card means the same thing on every surface it is shown on. Laying it
// out is the step each surface takes for itself; drawing what comes out is one
// implementation both of them reach, so what is compared here is the text the
// slots have been filled in, and text that agrees is drawn alike.
func TestAReviewCardMeansTheSameOnEverySurfaceItIsShownOn(t *testing.T) {
	read := corpusOf(t)

	laid, changed := 0, 0
	for _, one := range read.Templates {
		card := format.Card{Heading: "Llama"}
		for _, value := range one.Values {
			card.Values = append(card.Values, format.Value{Field: value.Field, Text: value.Text})
		}

		template := format.CardTemplate{Name: one.Name, Front: one.Template, Back: one.Template}
		front, back := format.Lay(template, card)
		laid++
		if one.Laid != one.Template {
			changed++
		}

		if front != one.Laid {
			t.Errorf("%s — %s\nthe template  %q\nlays out  %q\nthe table %q",
				one.Name, one.Construct, one.Template, front, one.Laid)
		}
		if back != front {
			t.Errorf("%s — %s: the two sides of one template lay out as %q and %q",
				one.Name, one.Construct, front, back)
		}
	}

	if laid != read.Count {
		t.Fatalf("the table declares %d templates and %d were laid out", read.Count, laid)
	}
	if changed != read.Changed {
		t.Fatalf("the table declares %d templates the slots change and %d of them differ from the template",
			read.Changed, changed)
	}
}

// corpusOf reads the table and rejects one that would pass without testing
// anything: a file that is empty, that declares no templates, that does not state
// its purpose, or that names one template twice.
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
	if len(read.Templates) == 0 || read.Count == 0 || read.Changed == 0 {
		t.Fatalf("the table holds %d templates and declares %d, %d of which the slots change",
			len(read.Templates), read.Count, read.Changed)
	}
	observed := map[string]bool{}
	for _, one := range read.Templates {
		if one.Name == "" || one.Construct == "" {
			t.Fatalf("a template of the table is called %q and pins %q", one.Name, one.Construct)
		}
		if observed[one.Name] {
			t.Fatalf("two templates of the table are called %q", one.Name)
		}
		observed[one.Name] = true
	}
	return read
}

func TestLayFillsATemplateWithACard(t *testing.T) {
	template := format.CardTemplate{
		Name:  "Recognise",
		Front: `<img src="llama.jpg" alt="a llama">`,
		Back:  "<b>{{Name}}</b> is {{Height}} and lives {{Life span}}.",
	}
	card := format.Card{
		Heading: "Llama",
		Values: []format.Value{
			{Field: "Name", Text: "Llama"},
			{Field: "Height", Text: `about 45"`},
			{Field: "Life span", Text: "about 20 years"},
		},
	}

	front, back := format.Lay(template, card)
	if front != `<img src="llama.jpg" alt="a llama">` {
		t.Errorf("front = %q, want the markup around a placeholder untouched", front)
	}
	if want := `<b>Llama</b> is about 45" and lives about 20 years.`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// The first field is a field like any other: it is laid out from what the card
// holds under its heading, and the card's own heading lays out nothing.
func TestTheFirstFieldLaysOutWhatTheCardContains(t *testing.T) {
	card := format.Card{
		Heading: "Llama, cut short",
		Values: []format.Value{
			{Field: "Name", Text: "Llama, and everything the heading had no room for"},
			{Field: "Height", Text: `45"`},
		},
	}
	template := format.CardTemplate{Front: "{{Name}}", Back: "{{Height}}"}

	front, _ := format.Lay(template, card)
	if front != "Llama, and everything the heading had no room for" {
		t.Errorf("front = %q, want the field and not the heading", front)
	}

	// 	// The position of a field does not change how a field is laid out.
	front, back := format.Lay(template, card)
	if front != "Llama, and everything the heading had no room for" || back != `45"` {
		t.Errorf("front = %q, back = %q", front, back)
	}
}

// A card the deck holds no first field for lays that field out as nothing, and
// its heading is not put there in its place.
func TestACardWithNoFirstFieldLaysNothingOutForIt(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, _ := format.Lay(
		format.CardTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
}

// A field the card leaves out lays out as nothing, and so does one the card model
// never declared.
func TestAPlaceholderWithNothingPriorItLaysOutAsNothing(t *testing.T) {
	template := format.CardTemplate{Front: "{{Name}}", Back: "[{{Height}}][{{Weight}}]"}
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, back := format.Lay(template, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `[45"][]` {
		t.Errorf("back = %q", back)
	}
}

// The name inside the braces is a field's name written exactly: a space is part
// of it, and so is every letter of every alphabet.
func TestAPlaceholderIsANameSavedExactly(t *testing.T) {
	card := format.Card{
		Heading: "яблоня",
		Values: []format.Value{
			{Field: "Life span", Text: "20"},
			{Field: "Жизнь", Text: "двадцать"},
			{Field: "Height", Text: `45"`},
		},
	}

	_, back := format.Lay(format.CardTemplate{
		Front: "{{Слово}}", Back: "{{Life span}} {{Жизнь}} {{ Height }} {{Слово}}",
	}, card)
	if want := "20 двадцать  "; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// Braces are not escaped. Braces written in prose produce a placeholder, and this
// test shows the result.
func TestBracesAreNotEscaped(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	_, back := format.Lay(
		format.CardTemplate{Front: "{{Name}}", Back: `\{{Height}} and {{{Height}}}`}, card)
	if want := `\45" and {45"}`; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A value is HTML, and it is laid out as written.
func TestAValueOfSeveralLinesLaysOutWhole(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{
		{Field: "Height", Text: "<h4>At the shoulder</h4>\n\nabout 45\""},
	}}

	_, back := format.Lay(
		format.CardTemplate{Front: "{{Name}}", Back: "<blockquote>{{Height}}</blockquote>"}, card)
	if want := "<blockquote><h4>At the shoulder</h4>\n\nabout 45\"</blockquote>"; back != want {
		t.Errorf("back = %q, want %q", back, want)
	}
}

// A card model that declares no field generates nothing: every placeholder in it
// refers to an undeclared field.
func TestACardModelOfNoFieldsLaysNothingOut(t *testing.T) {
	card := format.Card{Heading: "Llama", Values: []format.Value{{Field: "Height", Text: `45"`}}}

	front, back := format.Lay(format.CardTemplate{Front: "{{Name}}", Back: "{{Height}}"}, card)
	if front != "" {
		t.Errorf("front = %q, want nothing", front)
	}
	if back != `45"` {
		t.Errorf("back = %q, want the value the card does hold", back)
	}
}

// A template holding a front and no back lays out nothing at all.
func TestATemplateWithNoBackLaysNothingOut(t *testing.T) {
	card := format.Card{Values: []format.Value{{Field: "Name", Text: "Llama"}}}
	front, back := format.Lay(format.CardTemplate{Name: "Recognise", Front: "{{Name}}"}, card)
	if front != "" || back != "" {
		t.Errorf("laid out %q and %q, want nothing", front, back)
	}
}
