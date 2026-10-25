package deckedit

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

func TestGetCardRangeClampsToTheDeck(t *testing.T) {
	deck := make([]format.Card, 3)
	cases := []struct {
		name        string
		from, limit int
		want        int
	}{
		{"limit past the end gives the whole deck", 0, 50, 3},
		{"limit cuts the run", 0, 2, 2},
		{"negative start begins at the first card", -4, 50, 3},
		{"start past the end is empty", 9, 50, 0},
		{"limit past the end stops at the end", 1, 50, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := len(getCardRange(deck, c.from, c.limit)); got != c.want {
				t.Fatalf("got %d cards, want %d", got, c.want)
			}
		})
	}
}
