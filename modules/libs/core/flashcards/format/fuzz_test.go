package format_test

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
)

// deckSeeds are the shapes a deck's body arrives in: cards under a section and
// cards before one, a card already marked, a card naming a stencil, a card whose
// heading has fallen out of step with its first field, a heading of the deeper
// level a value may hold, a mark-shaped run that is not a mark, other scripts,
// carriage returns, and bytes that are no deck at all.
var deckSeeds = []string{
	"# Roots\n\n## a card\n\n[[Word]]\n\n### Word\n\nkaru\n\n### Meaning\n\nto do\n",
	"## a card before any section\n\n### Word\n\nkaru\n",
	"## a card ^0123456789\n\n### Word\n\nkaru\n",
	"## an old heading ^0123456789\n\n[[Word]]\n\n### Word\n\nthe new one\n",
	"## a card\n\n### Word\n\n#### a heading inside a value\n\nkaru\n",
	"## a card ^iloubadchar\n\n### Word\n\nkaru\n",
	"## карточка\n\n### Слово\n\nкару\n",
	"## a card\r\n\r\n### Word\r\n\r\nkaru\r\n",
	"##\n",
	"## \n## \n## \n",
	"prose above every card\n",
	"\x00\xff\xfe",
	"",
}

// A deck passes here once, on its way to the vault, and what comes out is
// finished: every card carries a mark that could have been minted, no card is
// gained or lost, and every mark this minted names the card it was minted for.
//
// Passing what came out through again changes nothing and mints nothing, so the
// pass is safe to make on every write.
func FuzzWhole(f *testing.F) {
	for _, seed := range deckSeeds {
		f.Add(seed, "Word")
	}
	f.Add("## a card\n\n[[Стенсил]]\n\n### Слово\n\nкару\n", "Слово")

	f.Fuzz(func(t *testing.T, body, field string) {
		stencils := map[string]format.Stencil{
			"Word":    {Fields: []string{field}},
			"Стенсил": {Fields: []string{field}},
			"empty":   {},
		}

		out, minted, err := format.Whole(body, stencils, counting())
		if err != nil {
			t.Fatalf("%q could not be made whole: %v", body, err)
		}
		was := format.ReadDeck(domain.Note{Body: body})
		now := format.ReadDeck(domain.Note{Body: out})
		if len(now.Cards) != len(was.Cards) {
			t.Fatalf("%d cards went in and %d came out\n was %q\n now %q",
				len(was.Cards), len(now.Cards), body, out)
		}
		for i, card := range now.Cards {
			if !cardid.Valid(card.Mark) {
				t.Fatalf("the card standing at %d carries %q, which is no mark\n now %q",
					i, card.Mark, out)
			}
		}
		for _, one := range minted {
			if one.Card < 0 || one.Card >= len(now.Cards) {
				t.Fatalf("a mark was minted for the card at %d, of %d cards",
					one.Card, len(now.Cards))
			}
			if got := now.Cards[one.Card].Mark; got != one.Mark {
				t.Fatalf("%q was minted for the card at %d, which carries %q",
					one.Mark, one.Card, got)
			}
			if was.Cards[one.Card].Mark != "" {
				t.Fatalf("the card at %d already carried %q and was minted %q",
					one.Card, was.Cards[one.Card].Mark, one.Mark)
			}
		}

		again, second, err := format.Whole(out, stencils, counting())
		if err != nil {
			t.Fatalf("a deck already made whole could not be: %v", err)
		}
		if again != out {
			t.Fatalf("a second pass rewrote the deck\n was %q\n now %q", out, again)
		}
		if len(second) != 0 {
			t.Fatalf("a second pass minted %d marks over %q", len(second), out)
		}
	})
}

// counting mints marks a run at a time, so that what comes out of one body is
// the same on every machine and in every run.
func counting() func() (domain.CardID, error) {
	at := 0
	return func() (domain.CardID, error) {
		at++
		return domain.CardID(fmt.Sprintf("%010d", at)), nil
	}
}
