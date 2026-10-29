package cards_test

import (
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// A path that the vault reports as a file it does not handle is an outcome of
// the file, and not an error.
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

// A deck that existed when it was looked up and is gone when it is read reports
// what it was when it was looked up.
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
	if got.FileStat.Path != "decks/Birds.md" || got.FileStat.Size == 0 {
		t.Errorf("fingerprint = %+v, want the file as it was looked at", got.FileStat)
	}
}

// A deck whose cards name no card model has no card model to return, and the
// result is no map at all.
func TestADeckNamingNoCardModelIsSplitByNone(t *testing.T) {
	vs := newVaults(t)
	u := cards.NewRead(vs.ports.Readers, nil)
	deck, err := u.Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	got, err := u.GetCardCardModels(t.Context(), vs.first, "decks/Birds.md", deck.Body)
	if err != nil {
		t.Fatalf("card models: %v", err)
	}
	if got != nil {
		t.Errorf("card models = %v, want none", got)
	}
}
