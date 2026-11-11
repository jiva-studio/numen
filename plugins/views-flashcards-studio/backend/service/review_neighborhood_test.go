package service

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
)

func getJoined() map[string]string {
	return map[string]string{
		"Term.md": getDeck()["Term.md"],
		"Mold.md": "---\ntype: note\n---\n" +
			"\n# Leaf mold\n\nLeaves left in a heap for two winters, and nothing else.\n",
		"decks/Words.md": "---\ntype: deck\n---\n" +
			"\nFrom [[Mold]].\n" +
			"\n## Leaf mold ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mold\n" +
			"\n### Meaning\n\nCompost made of fallen leaves alone\n",
	}
}

// Reading around a deck covers only the vaults this installation holds, and a
// query about another vault fails.
func TestGetDeckNeighborhood_UnknownVaultReturnsNotFound(t *testing.T) {
	handler, _ := newAPI(t, getJoined())

	_, err := handler.GetDeckNeighborhood(t.Context(), &api.GetDeckNeighborhoodRequest{
		Vault: "no-vault-of-this-identity",
		Deck:  "decks/Words.md",
	})
	if err == nil {
		t.Fatal("a vault this installation does not hold was read around")
	}
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
	if !hasCause(err, host.ErrUnknownVault) {
		t.Errorf("failed because %v", err)
	}
}

// A note the deck points at that cannot be read is reported with the same error
// the editor reports, so a rejected note has one vocabulary across the windows.
func TestGetDeckNeighborhood_UnreadableNoteMatchesEditorError(t *testing.T) {
	handler, target := newAPI(t, getJoined())
	v := target[0]

	// The index still resolves the link; the file behind it is gone. This is the
	// ordinary case of a note deleted since the last scan.
	if err := os.Remove(filepath.Join(v.Path, "Mold.md")); err != nil {
		t.Fatal(err)
	}

	out, err := handler.GetDeckNeighborhood(t.Context(), &api.GetDeckNeighborhoodRequest{
		Vault: string(v.ID), Deck: "decks/Words.md",
	})
	if err != nil {
		t.Fatal(err)
	}

	var found *api.DeckNeighbor
	for _, one := range out.Notes {
		if one.Path == "Mold.md" {
			found = one
		}
	}
	if found == nil {
		t.Fatalf("the note the deck points at is not among %+v", out.Notes)
	}
	if found.Error == api.ErrorCodeUnspecified {
		t.Fatal("a note that is not there came back without an error")
	}
	if got := found.Error; got != api.ErrorCodeNotFound {
		t.Errorf("a note that is not there fails as %v", got)
	}
	if found.Body != "" {
		t.Errorf("a note that could not be read came with prose: %q", found.Body)
	}
}

// brokenLinks is an index that cannot say what a note points at.
type brokenLinks struct{ ports.LinkQueries }

func (brokenLinks) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errors.New("the index is out of reach")
}

func getBacklinked() map[string]string {
	return map[string]string{
		"Term.md": getDeck()["Term.md"],
		"Mold.md": "---\ntype: note\n---\n" +
			"\n# Leaf mold\n\nLeaves left in a heap for two winters, and nothing else.\n",
		"Mention.md": "---\ntype: note\n---\n" +
			"\n# Mention\n\nSee [[Words]] for the cards.\n",
		"decks/Words.md": getJoined()["decks/Words.md"],
	}
}

// The index being out of reach is an internal failure and not an empty
// neighborhood.
func TestAnIndexOutOfReachIsAnInternalFailureForReadingAround(t *testing.T) {
	handler, target := newAPI(t, getJoined())
	*handler.neighborhood() = flashcards.NewShowNeighborhood(
		note.NewShowLinks(brokenLinks{}), nil, note.NewRead(hostports.Readers{Vaults: handler.vaults}))

	_, err := handler.GetDeckNeighborhood(t.Context(), &api.GetDeckNeighborhoodRequest{
		Vault: string(target[0].ID), Deck: "decks/Words.md",
	})
	if code := codeOf(err); code != server.CodeInternal {
		t.Errorf("reading around answered %v", err)
	}
}

// A note the deck points at is marked as linked from the deck, and a note
// pointing at the deck is not. Every entry is a note of the vault.
func TestGetDeckNeighborhood_MarksOnlyDeckLinkedNotes(t *testing.T) {
	handler, target := newAPI(t, getBacklinked())

	out, err := handler.GetDeckNeighborhood(t.Context(), &api.GetDeckNeighborhoodRequest{
		Vault: string(target[0].ID), Deck: "decks/Words.md",
	})
	if err != nil {
		t.Fatal(err)
	}

	pointed := map[string]bool{}
	for _, one := range out.Notes {
		if one.Path == "" {
			t.Errorf("an entry stands for no note: %+v", one)
		}
		pointed[one.Path] = one.IsPointedAt
	}
	if is, found := pointed["Mold.md"]; !found || !is {
		t.Errorf("the note the deck points at came back as %v (found %v)", is, found)
	}
	if is, found := pointed["Mention.md"]; !found || is {
		t.Errorf("the note pointing at the deck came back as %v (found %v)", is, found)
	}
}
