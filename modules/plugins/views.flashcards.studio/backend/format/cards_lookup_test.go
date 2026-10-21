package format_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

func TestACardStandingBeforeEverySectionIsNotAtAnyOfThem(t *testing.T) {
	if format.NoSection >= 0 {
		t.Errorf("NoSection is %d, which is a position among the sections", format.NoSection)
	}
}

func TestACardHoldsNoValueUnderAFieldItDoesNotHave(t *testing.T) {
	card := format.Card{Values: []format.Value{{Field: "Front", Text: "ant"}}}

	if text, ok := card.Value("Back"); ok || text != "" {
		t.Errorf("Value(Back) = %q, %v; want no value", text, ok)
	}
	if text, ok := card.Value("Front"); !ok || text != "ant" {
		t.Errorf("Value(Front) = %q, %v; want ant", text, ok)
	}
}

func TestADeckHoldingNoCardOfAMarkSaysSo(t *testing.T) {
	deck := format.Deck{Cards: []format.Card{{Mark: "aaaaaaaaaa", Heading: "ant"}}}

	got, err := deck.Card("bbbbbbbbbb")

	if !errors.Is(err, format.ErrNoSuchCard) {
		t.Fatalf("Card(bbbbbbbbbb) error = %v, want ErrNoSuchCard", err)
	}
	if got.Heading != "" || got.Mark != "" {
		t.Errorf("Card(bbbbbbbbbb) = %+v, want no card", got)
	}
}

func TestADeckHoldingTwoCardsOfAMarkSaysSo(t *testing.T) {
	deck := format.Deck{Cards: []format.Card{
		{Mark: cardid.ID("aaaaaaaaaa"), Heading: "ant"},
		{Mark: cardid.ID("aaaaaaaaaa"), Heading: "bee"},
	}}

	if _, err := deck.Card("aaaaaaaaaa"); !errors.Is(err, format.ErrTwoCards) {
		t.Fatalf("Card(aaaaaaaaaa) error = %v, want ErrTwoCards", err)
	}
}
