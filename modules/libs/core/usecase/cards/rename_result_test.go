package cards_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
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
		filesystem.VaultReaders{}, filesystem.VaultWriters{},
		decksOf{paths: []string{"decks/Birds.md"}, cancel: cancel},
		vs.db.NoteQueries(), indexNothing, time.Now,
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
