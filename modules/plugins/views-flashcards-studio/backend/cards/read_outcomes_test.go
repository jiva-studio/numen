package cards_test

import (
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/ports"
)

// A path the vault says holds a file it leaves alone is an outcome of the file,
// and no error.
func TestAPathTheVaultPreservesAloneIsNotADeckWhenItIsLookedAt(t *testing.T) {
	vs := newVaults(t)
	remaining := faulty{inner: vs.ports.Readers,
		stat: map[string]error{"decks/Birds.md": ports.ErrNotANote},
		read: map[string]error{"decks/Birds.md": errOutOfReach},
	}

	got, err := cards.NewRead(remaining, nil).Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.NotANote || got.Raw != "" || len(got.Body.Cards) != 0 {
		t.Errorf("outcome = %q, raw = %q, cards = %+v, want it not read as a note",
			got.Outcome, got.Raw, got.Body.Cards)
	}
}

// A deck that was there when it was looked at and gone when it was read says
// what it was when it was looked at.
func TestADeckMissingBetweenBeingLookedAtAndReadReturnsWhatItWas(t *testing.T) {
	vs := newVaults(t)
	removed := faulty{inner: vs.ports.Readers, read: map[string]error{"decks/Birds.md": fs.ErrNotExist}}

	got, err := cards.NewRead(removed, nil).Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Missing {
		t.Errorf("outcome = %q, want missing", got.Outcome)
	}
	if got.Fingerprint.Path != "decks/Birds.md" || got.Fingerprint.Size == 0 {
		t.Errorf("fingerprint = %+v, want the file as it was looked at", got.Fingerprint)
	}
}

// A deck whose cards name no stencil has no stencil to hand back, and that is
// no map at all.
func TestADeckNamingNoStencilIsSplitByNone(t *testing.T) {
	vs := newVaults(t)
	u := cards.NewRead(vs.ports.Readers, nil)
	deck, err := u.Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	got, err := u.GetCardStencils(t.Context(), vs.first, "decks/Birds.md", deck.Body)
	if err != nil {
		t.Fatalf("stencils: %v", err)
	}
	if got != nil {
		t.Errorf("stencils = %v, want none", got)
	}
}
