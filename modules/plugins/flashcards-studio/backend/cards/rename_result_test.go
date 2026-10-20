package cards_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/cards"
)

// A rename that stops because the decks cannot be named has still written the
// stencil, and says what the stencil now stands at.
func TestARenameWhoseDecksCannotBeNamedSaysWhatTheStencilStandsAt(t *testing.T) {
	vs := newVaults(t)
	u := newRenameField(t, vs).WithNotes(decksOf{err: errOutOfReach})

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("rename = %v, want the index's error", err)
	}
	if got.Stencil.Path != "Animal.md" || got.Stencil.Size == 0 {
		t.Errorf("stencil = %+v, want what the stencil stands at", got.Stencil)
	}
}

// A rename whose caller has gone has still written the stencil, and says what
// the stencil now stands at.
func TestARenameWhoseContextEndsSaysWhatTheStencilStandsAt(t *testing.T) {
	vs := newVaults(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	u := cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		decksOf{paths: []string{"decks/Birds.md"}, cancel: cancel},
		vs.queries, indexNothing, time.Now, cards.UnboundedDecks,
	)

	got, err := u.Execute(ctx, vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("rename = %v, want the caller's end", err)
	}
	if got.Stencil.Path != "Animal.md" || got.Stencil.Size == 0 {
		t.Errorf("stencil = %+v, want what the stencil stands at", got.Stencil)
	}
}
