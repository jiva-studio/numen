package flashcards_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// busy is an index that cannot be brought level, which is what a second writer
// holding it leaves behind.
func busy(context.Context, domain.Vault, []string) error {
	return errors.New("the index is busy")
}

// A deck that could not be written is named and left as it stands. The editor
// may be holding it, and a person is told which deck their cards are missing
// from rather than left to wonder.
func TestADeckThatCouldNotBeWrittenIsNamed(t *testing.T) {
	t.Parallel()
	s := opened(t, handwritten)

	// The folder is closed to writing, so the deck cannot be replaced. What a
	// person meets is the editor holding the file; this is the same refusal
	// from the same place.
	decks := filepath.Join(s.vault.Path, "decks")
	at, err := os.Stat(decks)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(decks, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(decks, at.Mode()) })

	marked, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a deck that could not be written was trouble: %v", err)
	}
	if !slices.Contains(marked.Unwritten, "decks/Own.md") {
		t.Errorf("the deck that could not be written is %v", marked.Unwritten)
	}
}

// A deck the index could not be brought level with was written all the same.
// Its cards carry the marks that were minted, so they stand in this sitting and
// nothing has to mint them a second time.
func TestADeckWrittenWithNoLevellingIsNotNamed(t *testing.T) {
	t.Parallel()
	s := opened(t, handwritten)
	marking := s.marking
	marking.Index = busy

	marked, err := marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a levelling that failed was trouble: %v", err)
	}
	if len(marked.Unwritten) != 0 {
		t.Fatalf("the deck was written and is named as unwritten: %v", marked.Unwritten)
	}
	if held := read(t, s.vault, "decks/Own.md"); !strings.Contains(held, "## Leaf mould ^") {
		t.Errorf("the deck carries no mark: %q", held)
	}

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 {
		t.Errorf("the marked card stands %d times, want once", len(stood))
	}
}

// A vault whose decks are all written has nothing to report, which is what says
// the naming above is the refusal and not the ordinary case.
func TestAVaultWhoseDecksAreWrittenNamesNone(t *testing.T) {
	t.Parallel()
	s := opened(t, handwritten)

	marked, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(marked.Unwritten) != 0 {
		t.Errorf("named %v, want none", marked.Unwritten)
	}
}
