package flashcards_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// faces is the card faces a session asks, in the order it asks them.
func faces(sat flashcards.SessionResult) []review.CardFaceID {
	out := make([]review.CardFaceID, 0, len(sat.Queue))
	for _, one := range sat.Queue {
		out = append(out, one.ID)
	}
	return out
}

// getCardFace is a card face of the one stencil these vaults are cut by.
func getCardFace(at int) review.CardFaceID {
	return review.CardFaceID{Card: mark(at), Face: "Say it"}
}

// The debt is paid oldest first, whatever order the cards stand in their deck.
//
// A day that cannot pay all of it answers the card faces waiting longest and
// leaves the rest of the pile standing.
func TestTheDebtIsPaidOldestFirst(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Two.md":        preset("new_a_day: 0\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Two", 5, 0),
	})
	// The cards stand in their deck in the reverse of the order they came round
	// in: the last of them was answered longest ago.
	for i := range 5 {
		answer(t, s.run(t, saturday.AddDate(0, 0, -10+i)), mark(4-i), 6*time.Second)
	}

	got := faces(s.openSessionByPreset(t, today, saturday, "Two.md"))
	want := []review.CardFaceID{getCardFace(4), getCardFace(3)}
	if !slices.Equal(got, want) {
		t.Errorf("a day of two reviews asked %v, want %v", got, want)
	}
	for _, one := range []int{2, 1, 0} {
		if slices.Contains(got, getCardFace(one)) {
			t.Errorf("%s was asked, and the day had room for the two older", mark(one))
		}
	}
}

// A session over several presets is one session however many times it is asked
// for: the same request over the same vault is the same cards in the same
// order, card face by card face.
func TestASessionOverSeveralPresetsIsTheSameSessionTwice(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":    term,
		"One.md":     preset("new_a_day: 2\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"Two.md":     preset("new_a_day: 3\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"Three.md":   preset("new_a_day: 1\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"decks/A.md": deckOf("One", 6, 0),
		"decks/B.md": deckOf("Two", 6, 100),
		"decks/C.md": deckOf("Three", 6, 200),
		"decks/D.md": deckOf("One", 6, 300),
	})
	// A debt under each preset, so every one of them has both sides of its day
	// to spend and none of them is handed its cards in one lump.
	for _, days := range []int{-9, -8} {
		record := s.run(t, saturday.AddDate(0, 0, days))
		for _, from := range []int{0, 100, 200, 300} {
			for i := range 2 {
				answer(t, record, mark(from+i), 6*time.Second)
			}
		}
	}

	first := faces(s.sessionAt(t, today, saturday))
	if len(first) == 0 {
		t.Fatal("the session asked nothing")
	}
	for range 5 {
		if again := faces(s.sessionAt(t, today, saturday)); !slices.Equal(first, again) {
			t.Fatalf("the session asked %v, and the same request asked %v", first, again)
		}
	}
}
