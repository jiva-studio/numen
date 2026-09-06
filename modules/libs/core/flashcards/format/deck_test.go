package format_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// note is a file as the application reads one, so a test says what is in the
// file and nothing about how it is parsed.
func note(t *testing.T, raw string) domain.Note {
	t.Helper()
	return markdown.Parse(domain.Fingerprint{Path: "Animals.md", Size: int64(len(raw))}, []byte(raw))
}

// filed is the one problem of a fault, and it fails the test where there is
// none or more than one.
func filed(t *testing.T, problems []format.Problem, fault format.Fault) format.Problem {
	t.Helper()
	var found []format.Problem
	for _, p := range problems {
		if p.Fault == fault {
			found = append(found, p)
		}
	}
	if len(found) != 1 {
		t.Fatalf("problems of %q = %d, want one\n%+v", fault, len(found), problems)
	}
	if found[0].Detail == "" {
		t.Errorf("the problem says nothing to the person: %+v", found[0])
	}
	return found[0]
}

func names(deck format.Deck) []string {
	var out []string
	for _, c := range deck.Cards {
		out = append(out, c.Heading)
	}
	return out
}

func fieldsOf(card format.Card) []string {
	var out []string
	for _, v := range card.Values {
		out = append(out, v.Field)
	}
	return out
}

func TestADeckIsReadCardByCard(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
type: deck
---

Cards I am learning.

## Llama

[[Animal]]

### Height

about 45" (shoulder)

### Life span

about 20 years

## компост

[[Термин]]

### Значение

перегной из листьев и травы
`))

	if deck.Preamble != "\nCards I am learning.\n\n" {
		t.Errorf("preamble = %q", deck.Preamble)
	}
	if got := names(deck); !slices.Equal(got, []string{"Llama", "компост"}) {
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
	if got, _ := deck.Cards[1].Value("Значение"); got != "перегной из листьев и травы" {
		t.Errorf("Значение = %q", got)
	}
	if deck.Cards[1].Stencil != "Термин" {
		t.Errorf("stencil = %q", deck.Cards[1].Stencil)
	}
}

// Two first fields that begin alike are two cards of one heading. Their marks
// tell them apart, and nothing else needed them to differ.
func TestTwoCardsOfOneHeading(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
type: deck
---

## Llama ^k7m2xq9fzp

[[Animal]]

### Height

about 45"

## Llama ^zpqrstvwxy

[[Animal]]

### Height

about 46"
`))

	if got := names(deck); !slices.Equal(got, []string{"Llama", "Llama"}) {
		t.Fatalf("cards = %v, both are read", got)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %+v, want two cards of one heading to be two cards", deck.Problems)
	}
	got, err := deck.Card("zpqrstvwxy")
	if err != nil {
		t.Fatalf("the second card is not addressed by its mark: %v", err)
	}
	if held, _ := got.Value("Height"); held != `about 46"` {
		t.Errorf("the mark reached the wrong card: %q", held)
	}
}

// A card copied by hand carries the mark it was copied from. Both are read and
// both are shown marked: nothing a person wrote goes missing from the screen,
// and which of the two is meant is a thing only they know.
func TestTwoCardsOfOneMark(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
type: deck
---

## Llama ^k7m2xq9fzp

[[Animal]]

### Height

about 45"

## Alpaca ^k7m2xq9fzp

[[Animal]]

### Height

about 35"

## Vicuña ^zpqrstvwxy

[[Animal]]

### Height

about 30"
`))

	var against []int
	for _, p := range deck.Problems {
		if p.Fault == format.FaultTwoMarks {
			against = append(against, p.Card)
		}
	}
	if !slices.Equal(against, []int{0, 1}) {
		t.Errorf("problems = %+v, want one against each of the two", deck.Problems)
	}
	if got := names(deck); !slices.Equal(got, []string{"Llama", "Alpaca", "Vicuña"}) {
		t.Errorf("cards = %v, want all of them read", got)
	}
	if got, _ := deck.Cards[1].Value("Height"); got != `about 35"` {
		t.Errorf("the second of the two was not read whole: %q", got)
	}
}

// A heading is closed by a run of hashes only where whitespace stands in front
// of them, so a name ending in one keeps it and is the whole of what the card
// is called.
func TestAHeadingEndingInAHash(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
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
	if got := fieldsOf(deck.Cards[0]); !slices.Equal(got, []string{"F#"}) {
		t.Errorf("fields = %v", got)
	}
}

func TestTwoFieldsOfOneName(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
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
	if got := filed(t, deck.Problems, format.FaultTwoValues); got.Card != 0 || got.Field != "Height" {
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
			deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n## Llama\n\n"+head+"\n\n### Height\n\nabout 45\"\n"))

			card := deck.Cards[0]
			if card.Stencil != "" {
				t.Errorf("stencil = %q, want none", card.Stencil)
			}
			if got := filed(t, deck.Problems, format.FaultNoStencil).Card; got != 0 {
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
	deck := format.ReadDeck(note(t, `---
type: deck
---

## Llama

[[Animal]]

Written on the seed packet.

### Height

about 45"
`))

	card := deck.Cards[0]
	if card.Stencil != "Animal" {
		t.Errorf("stencil = %q", card.Stencil)
	}
	if card.Lead != "Written on the seed packet." {
		t.Errorf("lead = %q", card.Lead)
	}
}

// A first field that is empty gives a heading of nothing, and the card is a
// card like any other.
func TestACardWithNothingInItsHeading(t *testing.T) {
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n## \n\n[[Animal]]\n\n### Height\n\nabout 45\"\n"))

	if len(deck.Cards) != 1 {
		t.Fatalf("cards = %v, a heading of nothing still opens a card", deck.Cards)
	}
	if deck.Cards[0].Heading != "" {
		t.Errorf("heading = %q", deck.Cards[0].Heading)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %+v, want a heading of nothing to be no fault", deck.Problems)
	}
	if got, _ := deck.Cards[0].Value("Height"); got != `about 45"` {
		t.Errorf("the value was not read: %q", got)
	}
}

// A first-level heading opens a section. The cards under it are its own until
// the next one, and the section is a name and the person's own text.
func TestAFirstLevelHeadingOpensASection(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
type: deck
---

Cards I am learning.

## Loose ^k7m2xq9fzp

[[Animal]]

### Height

about 45"

# Roots

The ones I began with.

## Compost ^zpqrstvwxy

[[Term]]

### Значение

перегной

# Roots

## Bracken ^m9n8b7v6c5

[[Term]]

### Значение

папоротник

# Empty
`))

	if deck.Preamble != "\nCards I am learning.\n\n" {
		t.Errorf("preamble = %q, want what stands above the first of them", deck.Preamble)
	}
	// Two sections may carry one name, and an empty section is kept.
	var held []string
	for _, s := range deck.Sections {
		held = append(held, s.Name)
	}
	if !slices.Equal(held, []string{"Roots", "Roots", "Empty"}) {
		t.Fatalf("sections = %v", held)
	}
	// Text between a section's heading and its first card is the section's.
	if deck.Sections[0].Lead != "The ones I began with." {
		t.Errorf("lead = %q", deck.Sections[0].Lead)
	}
	if deck.Sections[1].Lead != "" || deck.Sections[2].Lead != "" {
		t.Errorf("sections = %+v, want no text under either", deck.Sections[1:])
	}

	// A card may stand before the first section, and stands under none.
	var under []int
	for _, c := range deck.Cards {
		under = append(under, c.Section)
	}
	if !slices.Equal(under, []int{format.NoSection, 0, 1}) {
		t.Errorf("sections = %v, want the first card under none", under)
	}
	if len(deck.Problems) != 0 {
		t.Errorf("problems = %+v", deck.Problems)
	}
}

// A deck of sections and no cards is sections all the same, and what a person
// wrote under each of them is kept.
func TestADeckOfSectionsAndNoCards(t *testing.T) {
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n# Animals\n\n### Not a card\n\nprose.\n"))

	if len(deck.Cards) != 0 {
		t.Fatalf("cards = %v", deck.Cards)
	}
	if deck.Preamble != "\n" {
		t.Errorf("preamble = %q, want what stands above the first section", deck.Preamble)
	}
	if len(deck.Sections) != 1 || deck.Sections[0].Name != "Animals" {
		t.Fatalf("sections = %+v", deck.Sections)
	}
	// A heading nothing reads is text under the section it falls in.
	if deck.Sections[0].Lead != "### Not a card\n\nprose." {
		t.Errorf("lead = %q", deck.Sections[0].Lead)
	}
}

// The format spends two heading levels and no more, so a heading below them is
// part of the value it stands in.
func TestAValueHoldsADeeperHeading(t *testing.T) {
	deck := format.ReadDeck(note(t, `---
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
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n\n### Height\n\n```markdown\n## Alpaca\n\n### Weight\n```\n"))

	if got := names(deck); !slices.Equal(got, []string{"Llama"}) {
		t.Errorf("cards = %v", got)
	}
	if got := fieldsOf(deck.Cards[0]); !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
}

// A block opened with tildes is closed by tildes, so the backticks inside it
// are part of the example and a heading standing among them opens nothing.
func TestATildeFencedExampleHoldsBackticks(t *testing.T) {
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n"+
		"## Llama\n\n[[Animal]]\n\n### Height\n\n"+
		"~~~markdown\n```\n## Alpaca\n```\n### Weight\n~~~\n\n"+
		"## Vicuña\n\n[[Animal]]\n\n### Height\n\nabout 36\"\n"))

	if got := names(deck); !slices.Equal(got, []string{"Llama", "Vicuña"}) {
		t.Errorf("cards = %v", got)
	}
	if got := fieldsOf(deck.Cards[0]); !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
	if !strings.Contains(deck.Cards[0].Values[0].Text, "## Alpaca") {
		t.Errorf("the example was cut out of the value: %q", deck.Cards[0].Values[0].Text)
	}
}

func TestAnEmptyDeck(t *testing.T) {
	n := note(t, "---\ntype: deck\n---\n\nNothing here yet.\n")
	if n.Type != domain.TypeDeck {
		t.Fatalf("type = %q", n.Type)
	}

	deck := format.ReadDeck(n)
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
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\nCards I am learning.\n\n"+
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
	deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\nAnimals\n\n### Not a card\n\nprose.\n"))

	if len(deck.Cards) != 0 {
		t.Fatalf("cards = %v", deck.Cards)
	}
	if deck.Preamble != "\nAnimals\n\n### Not a card\n\nprose.\n" {
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
			deck := format.ReadDeck(note(t, "---\ntype: deck\n---\n\n"+name))
			if deck.Tail != want {
				t.Errorf("tail = %q, want %q", deck.Tail, want)
			}
		})
	}
}

func TestADeckOfCRLFReadsAsOneKindOfBreak(t *testing.T) {
	raw := "---\r\ntype: deck\r\n---\r\n\r\n## Llama\r\n\r\n[[Animal]]\r\n\r\n### Height\r\n\r\nabout 45\"\r\nat the shoulder\r\n"
	deck := format.ReadDeck(note(t, raw))

	if deck.Cards[0].Stencil != "Animal" {
		t.Errorf("stencil = %q", deck.Cards[0].Stencil)
	}
	if got, _ := deck.Cards[0].Value("Height"); got != "about 45\"\nat the shoulder" {
		t.Errorf("Height = %q, want no carriage returns", got)
	}
}
