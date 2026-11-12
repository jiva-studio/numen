package cards_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

// A rename that stops because the decks cannot be named has still written the
func TestRenameField_ReturnsCardModelStateOnDeckListError(t *testing.T) {
	vs := newVaults(t)
	u := newRenameFieldOver(t, vs, vs.ports.Writers, decksOf{err: errOutOfReach}, cards.UnboundedDecks)

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("rename = %v, want the index's error", err)
	}
	if got.CardModel.Path != "Animal.md" || got.CardModel.Size == 0 {
		t.Errorf("card model = %+v, want current card model state", got.CardModel)
	}
}

// A rename whose context is canceled has still written the card model, and
// returns its current state.
func TestARenameWhoseContextEndsReturnsWhatTheCardModelExistsAt(t *testing.T) {
	vs := newVaults(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	u := renameOf(
		vs.ports.Readers, vs.ports.Writers,
		decksOf{paths: []string{"decks/Birds.md"}, cancel: cancel},
		vs.queries, indexNothing)

	got, err := u.Execute(ctx, vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("rename = %v, want the caller's end", err)
	}
	if got.CardModel.Path != "Animal.md" || got.CardModel.Size == 0 {
		t.Errorf("card model = %+v, want current card model state", got.CardModel)
	}
}
