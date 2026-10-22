package flashcards_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// busy is an index that cannot be brought level, which is what a second writer
func busy(context.Context, domain.Vault, []string) error {
	return errors.New("the index is busy")
}

// A deck that could not be written is named and left currently. The editor
// from and not left to wonder.
func TestADeckThatCouldNotBeSavedIsNamed(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)

	// The deck cannot be replaced. What a person meets is the editor holding
	// the file; this is the same refusal from the same place.
	s.env.Host.RefuseWriting("decks/Own.md", fs.ErrPermission)

	marked, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a deck that could not be written was trouble: %v", err)
	}
	if !slices.Contains(marked.Unwritten, "decks/Own.md") {
		t.Errorf("the deck that could not be written is %v", marked.Unwritten)
	}
}

// A deck the index could not be brought level with was written all the same.
// Its cards carry the marks that were minted, so they stand in this session and
// nothing has to mint them a second time.
func TestADeckSavedWithNoLevellingIsNotNamed(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)
	marking := s.marking.WithIndex(busy)

	marked, err := marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a levelling that failed was trouble: %v", err)
	}
	if len(marked.Unwritten) != 0 {
		t.Fatalf("the deck was written and is named as unwritten: %v", marked.Unwritten)
	}
	if target := read(t, s.vault, "decks/Own.md"); !strings.Contains(target, "## Leaf mould ^") {
		t.Errorf("the deck carries no mark: %q", target)
	}

	status, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 {
		t.Errorf("the marked card stands %d times, want once", len(status))
	}
}

// A vault whose decks are all written has nothing to report, which is what says
// the naming above is the refusal and not the ordinary case.
func TestAVaultWhoseDecksAreSavedNamesNone(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)

	marked, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(marked.Unwritten) != 0 {
		t.Errorf("named %v, want none", marked.Unwritten)
	}
}
