package flashcards_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A vault the index has never read is not a vault holding no cards. The
// difference matters at the front door: one owes nothing, the other is not
// known yet, and a person is told which.
func TestAVaultTheIndexHasNotReadIsNotAVaultOfNoCards(t *testing.T) {
	s := opened(t, vault)
	unread := domain.Vault{ID: "nobody-scanned-this", Name: "Unread", Path: s.vault.Path}

	_, err := s.standings.Execute(t.Context(), unread)
	if !errors.Is(err, flashcards.ErrUnread) {
		t.Errorf("a vault the index has not read came back with %v", err)
	}
}

// One deck that cannot be read is not a reason to refuse a person the rest of
// their cards. What was wrong with it is settled in the editor.
func TestADeckThatCannotBeReadLeavesTheOthersStanding(t *testing.T) {
	s := opened(t, vault)

	whole, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(whole) == 0 {
		t.Fatal("the vault stands for nothing")
	}

	// The index still lists the deck; the file behind it is gone, which is what
	// a person deleting one outside the editor leaves behind.
	if err := os.Remove(filepath.Join(s.vault.Path, "decks", "Words.md")); err != nil {
		t.Fatal(err)
	}

	left, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("one deck gone refused the whole vault: %v", err)
	}
	if len(left) == 0 {
		t.Error("one deck gone left the vault standing for nothing")
	}
	if len(left) >= len(whole) {
		t.Errorf("the gone deck still stands: %d faces of %d", len(left), len(whole))
	}
	for _, one := range left {
		if one.Deck == "decks/Words.md" {
			t.Errorf("a card of the gone deck stands: %+v", one)
		}
	}
}
