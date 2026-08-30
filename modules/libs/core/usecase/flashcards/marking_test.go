package flashcards_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A deck that could not be written is named and left as it stands. The editor
// may be holding it, and a person is told which deck their cards are missing
// from rather than left to wonder.
func TestADeckThatCouldNotBeWrittenIsNamed(t *testing.T) {
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

// A vault whose decks are all written has nothing to report, which is what says
// the naming above is the refusal and not the ordinary case.
func TestAVaultWhoseDecksAreWrittenNamesNone(t *testing.T) {
	s := opened(t, handwritten)

	marked, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(marked.Unwritten) != 0 {
		t.Errorf("named %v, want none", marked.Unwritten)
	}
}
