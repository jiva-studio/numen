package cards_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// note is a file as the application reads one, so a test says what is in the
// file and nothing about how it is parsed.
func note(t *testing.T, raw string) domain.Note {
	t.Helper()
	return markdown.Parse(domain.FileRef{Path: "Animals.md", Size: int64(len(raw))}, []byte(raw))
}

// filed is the one problem of a check, and it fails the test where there is
// none or more than one.
func filed(t *testing.T, problems []cards.Problem, check cards.Check) cards.Problem {
	t.Helper()
	var found []cards.Problem
	for _, p := range problems {
		if p.Check == check {
			found = append(found, p)
		}
	}
	if len(found) != 1 {
		t.Fatalf("problems of %q = %d, want one\n%+v", check, len(found), problems)
	}
	if found[0].Detail == "" {
		t.Errorf("the problem says nothing to the person: %+v", found[0])
	}
	return found[0]
}

func names(deck cards.Deck) []string {
	var out []string
	for _, c := range deck.Cards {
		out = append(out, c.Name)
	}
	return out
}

func fieldsOf(card cards.Card) []string {
	var out []string
	for _, v := range card.Values {
		out = append(out, v.Field)
	}
	return out
}

func TestADeckIsReadCardByCard(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

Cards I am learning.

## Llama

[[Animal]]

### Height

about 45" (shoulder)

### Life span

about 20 years

## шраддха

[[Термин]]

### Значение

вера, рождённая из слушания
`))

	if deck.Preamble != "\nCards I am learning.\n\n" {
		t.Errorf("preamble = %q", deck.Preamble)
	}
	if got := names(deck); !slices.Equal(got, []string{"Llama", "шраддха"}) {
		t.Fatalf("cards = %v", got)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %v", deck.Problems)
	}

	llama := deck.Cards[0]
	if llama.Stencil != "Animal" {
		t.Errorf("stencil = %q", llama.Stencil)
	}
	if got := fieldsOf(llama); !slices.Equal(got, []string{"Height", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
	if got, _ := llama.Value("Height"); got != `about 45" (shoulder)` {
		t.Errorf("Height = %q", got)
	}
	// A name that is not Latin is a name.
	if got, _ := deck.Cards[1].Value("Значение"); got != "вера, рождённая из слушания" {
		t.Errorf("Значение = %q", got)
	}
	if deck.Cards[1].Stencil != "Термин" {
		t.Errorf("stencil = %q", deck.Cards[1].Stencil)
	}
}

// The first of two cards of one name stands, and the second is read and shown
// as a problem: nothing in the file is dropped.
func TestTwoCardsOfOneName(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

## Llama

[[Animal]]

### Height

about 45"

## Llama

[[Animal]]

### Height

nonsense
`))

	if got := names(deck); !slices.Equal(got, []string{"Llama", "Llama"}) {
		t.Fatalf("cards = %v, both are read", got)
	}
	// The name reaches both, so the problem is filed against the second by
	// where it stands.
	if got := filed(t, deck.Problems, cards.CheckTwoCards).Card; got != 1 {
		t.Errorf("card = %d, want the second of the two", got)
	}
	if got, _ := deck.Card("Llama"); got.Values[0].Text != `about 45"` {
		t.Errorf("the first card does not stand: %q", got.Values[0].Text)
	}
}

// A heading is closed by a run of hashes only where whitespace stands in front
// of them, so a name ending in one keeps it and is the whole of what the card
// is called.
func TestAHeadingEndingInAHash(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

## C#

[[Language]]

### F#

1999

## C ###

[[Language]]

### Height

1972
`))

	if got := names(deck); !slices.Equal(got, []string{"C#", "C"}) {
		t.Fatalf("cards = %v", got)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %v, want two cards of two names", deck.Problems)
	}
	card, held := deck.Card("C#")
	if !held {
		t.Fatal("the card is not addressed by the heading as it is written")
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"F#"}) {
		t.Errorf("fields = %v", got)
	}
}

func TestTwoFieldsOfOneName(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

## Llama

[[Animal]]

### Height

about 45"

### Height

about 46"
`))

	card := deck.Cards[0]
	if got := filed(t, deck.Problems, cards.CheckTwoValues); got.Card != 0 || got.Field != "Height" {
		t.Errorf("problem = %+v, want it against the first card and its Height", got)
	}
	if got, ok := card.Value("Height"); !ok || got != `about 45"` {
		t.Errorf("Height = %q, want the first", got)
	}
	if len(card.Values) != 2 {
		t.Errorf("values = %v, both are kept", card.Values)
	}
}

// A card whose first paragraph is not a lone wikilink names no stencil. Its
// values are read either way.
func TestACardNamesNoStencil(t *testing.T) {
	for name, head := range map[string]string{
		"prose":                      "A llama is an animal.",
		"a link in prose":            "[[Animal]] is what it is.",
		"two links":                  "[[Animal]] [[Mammal]]",
		"a link under prose":         "About this one.\n\n[[Animal]]",
		"a link with prose under it": "[[Animal]]\nand a second line",
		"nothing at all":             "",
	} {
		t.Run(name, func(t *testing.T) {
			deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\n## Llama\n\n"+head+"\n\n### Height\n\nabout 45\"\n"))

			card := deck.Cards[0]
			if card.Stencil != "" {
				t.Errorf("stencil = %q, want none", card.Stencil)
			}
			if got := filed(t, deck.Problems, cards.CheckNoStencil).Card; got != 0 {
				t.Errorf("card = %d", got)
			}
			if got, ok := card.Value("Height"); !ok || got != `about 45"` {
				t.Errorf("the value was not read: %q", got)
			}
			if head != "" && !strings.Contains(card.Lead, strings.SplitN(head, "\n", 2)[0]) {
				t.Errorf("lead = %q, want what stands under the heading", card.Lead)
			}
		})
	}
}

func TestALoneWikilinkIsTheStencilAndTheRestIsLead(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

## Llama

[[Animal]]

Asked of me by Anna.

### Height

about 45"
`))

	card := deck.Cards[0]
	if card.Stencil != "Animal" {
		t.Errorf("stencil = %q", card.Stencil)
	}
	if card.Lead != "Asked of me by Anna." {
		t.Errorf("lead = %q", card.Lead)
	}
}

func TestACardWithNoName(t *testing.T) {
	deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\n## \n\n[[Animal]]\n\n### Height\n\nabout 45\"\n"))

	if len(deck.Cards) != 1 {
		t.Fatalf("cards = %v, an unnamed heading still opens a card", deck.Cards)
	}
	if deck.Cards[0].Name != "" {
		t.Errorf("name = %q", deck.Cards[0].Name)
	}
	// A card with no name is addressed by where it stands, because its name
	// addresses nothing.
	if got := filed(t, deck.Problems, cards.CheckNoName).Card; got != 0 {
		t.Errorf("card = %d", got)
	}
	if got, _ := deck.Cards[0].Value("Height"); got != `about 45"` {
		t.Errorf("the value was not read: %q", got)
	}
}

// The format spends two heading levels and no more, so a heading below them is
// part of the value it stands in.
func TestAValueHoldsADeeperHeading(t *testing.T) {
	deck := cards.ReadDeck(note(t, `---
type: deck
---

## Llama

[[Animal]]

### Height

#### At the shoulder

about 45"

##### Standing

about 5' 6"

### Life span

about 20 years
`))

	card := deck.Cards[0]
	if got := fieldsOf(card); !slices.Equal(got, []string{"Height", "Life span"}) {
		t.Fatalf("fields = %v", got)
	}
	height, _ := card.Value("Height")
	for _, want := range []string{"#### At the shoulder", "##### Standing", `about 5' 6"`} {
		if !strings.Contains(height, want) {
			t.Errorf("Height = %q, want it to hold %q", height, want)
		}
	}
}

// A heading inside a code fence is an example of a heading.
func TestAFencedHeadingOpensNothing(t *testing.T) {
	deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n\n### Height\n\n```markdown\n## Alpaca\n\n### Weight\n```\n"))

	if got := names(deck); !slices.Equal(got, []string{"Llama"}) {
		t.Errorf("cards = %v", got)
	}
	if got := fieldsOf(deck.Cards[0]); !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
}

func TestAnEmptyDeck(t *testing.T) {
	n := note(t, "---\ntype: deck\n---\n\nNothing here yet.\n")
	if n.Type != domain.TypeDeck {
		t.Fatalf("type = %q", n.Type)
	}

	deck := cards.ReadDeck(n)
	if len(deck.Cards) != 0 {
		t.Errorf("cards = %v", deck.Cards)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %v", deck.Problems)
	}
	if deck.Preamble != "\nNothing here yet.\n" {
		t.Errorf("preamble = %q", deck.Preamble)
	}
}

// A value stops where its own text stops, so what a file ends with once the
// last value has been read is the tail and is nobody's value. The preamble is
// the bytes above the first card, down to the one its heading opens on.
func TestThePreambleAndTheTail(t *testing.T) {
	deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\nCards I am learning.\n\n"+
		"## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n\n\n"))

	if deck.Preamble != "\nCards I am learning.\n\n" {
		t.Errorf("preamble = %q", deck.Preamble)
	}
	if got, _ := deck.Cards[0].Value("Height"); got != `about 45"` {
		t.Errorf("Height = %q, want the blank lines under it left out of it", got)
	}
	if deck.Tail != "\n\n\n" {
		t.Errorf("tail = %q, want what the file ends with", deck.Tail)
	}
}

// A deck of no cards is all preamble, and there is nothing for a tail to
// follow.
func TestADeckOfNoCardsIsAllPreamble(t *testing.T) {
	deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\n# Animals\n\n### Not a card\n\nprose.\n"))

	if len(deck.Cards) != 0 {
		t.Fatalf("cards = %v", deck.Cards)
	}
	if deck.Preamble != "\n# Animals\n\n### Not a card\n\nprose.\n" {
		t.Errorf("preamble = %q", deck.Preamble)
	}
	if deck.Tail != "" {
		t.Errorf("tail = %q", deck.Tail)
	}
}

// The tail follows the last thing that was read, whatever that was.
func TestTheTailOfACardThatHasNothingUnderIt(t *testing.T) {
	for name, want := range map[string]string{
		"## Llama\n\n[[Animal]]\n\n":               "\n\n",
		"## Llama\n\n[[Animal]]\n\nA lead.\n\n":    "\n\n",
		"## Llama\n\n":                             "\n\n",
		"## Llama\n\n[[Animal]]\n\n### Height\n\n": "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			deck := cards.ReadDeck(note(t, "---\ntype: deck\n---\n\n"+name))
			if deck.Tail != want {
				t.Errorf("tail = %q, want %q", deck.Tail, want)
			}
		})
	}
}

func TestADeckOfCRLFReadsAsOneKindOfBreak(t *testing.T) {
	raw := "---\r\ntype: deck\r\n---\r\n\r\n## Llama\r\n\r\n[[Animal]]\r\n\r\n### Height\r\n\r\nabout 45\"\r\nat the shoulder\r\n"
	deck := cards.ReadDeck(note(t, raw))

	if deck.Cards[0].Stencil != "Animal" {
		t.Errorf("stencil = %q", deck.Cards[0].Stencil)
	}
	if got, _ := deck.Cards[0].Value("Height"); got != "about 45\"\nat the shoulder" {
		t.Errorf("Height = %q, want no carriage returns", got)
	}
}
