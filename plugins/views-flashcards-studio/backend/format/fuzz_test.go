package format_test

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// deckSeeds are the shapes a deck's body arrives in: cards under a section and
// cards before one, a card already marked, a card naming a card model, a card whose
// heading has fallen out of step with its first field, a heading of the deeper
// level a value may hold, a card id-shaped run that is not a card id, other scripts,
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
// finished: every card carries a card id that could have been created, no card is
// gained or lost, and every card id this gave names the card it was given to.
//
// Passing what came out through again changes nothing and gives no card id, so the
// pass is safe to make on every write.
func FuzzWhole(f *testing.F) {
	for _, seed := range deckSeeds {
		f.Add(seed, "Word")
	}
	f.Add("## a card\n\n[[Стенсил]]\n\n### Слово\n\nкару\n", "Слово")

	f.Fuzz(func(t *testing.T, body, field string) {
		cardModels := map[string]format.CardModel{
			"Word":    {Fields: []string{field}},
			"Стенсил": {Fields: []string{field}},
			"empty":   {},
		}

		out, provided := format.Whole(body, cardModels, createIDs())
		was := format.ReadDeck(domain.Note{Body: body})
		now := format.ReadDeck(domain.Note{Body: out})
		if len(now.Cards) != len(was.Cards) {
			t.Fatalf("%d cards went in and %d came out\n was %q\n now %q",
				len(was.Cards), len(now.Cards), body, out)
		}
		for i, card := range now.Cards {
			if !cardid.IsValid(card.CardID) {
				t.Fatalf("the card standing at %d carries %q, which is no card id\n now %q",
					i, card.CardID, out)
			}
		}
		for _, one := range provided {
			if one.Card < 0 || one.Card >= len(now.Cards) {
				t.Fatalf("a card id was given to the card at %d, of %d cards",
					one.Card, len(now.Cards))
			}
			if got := now.Cards[one.Card].CardID; got != one.CardID {
				t.Fatalf("%q was given to the card at %d, which carries %q",
					one.CardID, one.Card, got)
			}
			if was.Cards[one.Card].CardID != "" {
				t.Fatalf("the card at %d already carried %q and was given %q",
					one.Card, was.Cards[one.Card].CardID, one.CardID)
			}
		}

		again, second := format.Whole(out, cardModels, createIDs())
		if again != out {
			t.Fatalf("a second pass rewrote the deck\n was %q\n now %q", out, again)
		}
		if len(second) != 0 {
			t.Fatalf("a second pass gave %d card ids over %q", len(second), out)
		}
	})
}

// createIDs creates card ids a run at a time, so the output for one body is the
// same on every machine and in every run.
func createIDs() func() cardid.ID {
	pos := 0
	return func() cardid.ID {
		pos++
		return cardid.ID(fmt.Sprintf("%010d", pos))
	}
}
