package cards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
)

// twoStencils is a deck of a card under each of two stencils, one of which
// writes its first field a second time.
const twoStencils = "---\ntype: deck\n---\n\n" +
	"## Llama\n\n[[Animal]]\n\n### Name\n\nLlama, a second time\n\n### Height\n\nabout 45\"\n\n" +
	"## компост\n\n[[Term]]\n\n### Значение\n\nперегной\n"

func cutting() map[string]cards.Stencil {
	return map[string]cards.Stencil{
		"Animal": {Fields: []string{"Name", "Height"}},
		"Term":   {Fields: []string{"Слово", "Значение"}},
	}
}

// The heading is where the first field is written, so a card writing it as a
// heading of its own carries it twice. That is a problem against the deck: the
// deck is the file somebody would open to settle it.
func TestACardWritingItsFirstFieldTwice(t *testing.T) {
	deck := cards.ReadDeck(note(t, twoStencils))

	problems := cards.Cut(deck, cutting())
	if len(problems) != 1 {
		t.Fatalf("problems = %+v, want the one card that writes it twice", problems)
	}
	if problems[0].Check != cards.CheckFirstFieldTwice {
		t.Errorf("check = %q", problems[0].Check)
	}
	if problems[0].Card != 0 || problems[0].Field != "Name" {
		t.Errorf("problem = %+v, want it against the first card and Name", problems[0])
	}

	// The heading stands, and the third-level one is still in the file.
	card := deck.Cards[0]
	if got, _ := cutting()["Animal"].Value(card, "Name"); got != "Llama" {
		t.Errorf("Name = %q, want the heading", got)
	}
	if got, ok := card.Value("Name"); !ok || got != "Llama, a second time" {
		t.Errorf("the second one was dropped from the file: %q", got)
	}
}

// Which field the heading holds is whichever stands first now, so a person who
// reordered `fields` by hand loses nothing and nothing falls over.
func TestTheHeadingIsReadAgainstWhicheverFieldStandsFirst(t *testing.T) {
	deck := cards.ReadDeck(note(t, twoStencils))
	reordered := map[string]cards.Stencil{"Animal": {Fields: []string{"Height", "Name"}}}

	if got := cards.Cut(deck, reordered); len(got) != 1 || got[0].Field != "Height" {
		t.Errorf("problems = %+v, want the one now standing first said twice", got)
	}
	if got, _ := reordered["Animal"].Value(deck.Cards[0], "Height"); got != "Llama" {
		t.Errorf("Height = %q, want the heading, which is the field standing first now", got)
	}
	if got, _ := reordered["Animal"].Value(deck.Cards[0], "Name"); got != "Llama, a second time" {
		t.Errorf("Name = %q, want what the card writes under that heading", got)
	}
}

// A card naming a stencil nothing was read for is read against none, and a
// stencil declaring no field names no card at all.
func TestACardReadAgainstNoStencil(t *testing.T) {
	deck := cards.ReadDeck(note(t, twoStencils))

	for name, stencils := range map[string]map[string]cards.Stencil{
		"nothing at all":  nil,
		"another name":    {"Camelid": {Fields: []string{"Name"}}},
		"no field of its": {"Animal": {}},
	} {
		t.Run(name, func(t *testing.T) {
			if problems := cards.Cut(deck, stencils); len(problems) != 0 {
				t.Errorf("problems = %+v, want none", problems)
			}
		})
	}
}
