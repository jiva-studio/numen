package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
)

// joined is a vault whose deck points at a note of prose, the way a person
// cuts cards out of something they wrote.
var joined = map[string]string{
	"Term.md": deck["Term.md"],
	"Mould.md": "---\ntype: note\n---\n" +
		"\n# Leaf mould\n\nLeaves left in a heap for two winters, and nothing else.\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\nFrom [[Mould]].\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// Reading around a deck reaches only the vaults this installation holds, and a
// question about another is refused.
func TestReadingAroundIsRefusedForAVaultThisInstallationDoesNotHold(t *testing.T) {
	api, _ := newAPI(t, joined)

	_, err := api.GetDeckNeighbourhood(t.Context(), connect.NewRequest(&v1.GetDeckNeighbourhoodRequest{
		Vault: "no-vault-of-this-identity",
		Deck:  "decks/Words.md",
	}))
	if err == nil {
		t.Fatal("a vault this installation does not hold was read around")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
	if !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("refused because %v", err)
	}
}

// A note the deck points at that cannot be read comes back named the way the
// editor names it, so a person meets one vocabulary for a refused note and not
// one for each window.
func TestANoteThatCannotBeReadIsRefusedAsTheEditorRefusesIt(t *testing.T) {
	api, held := newAPI(t, joined)
	v := held[0]

	// The index still resolves the link; the file behind it is gone. That is
	// the ordinary way a person meets this — a note deleted since the scan.
	if err := os.Remove(filepath.Join(v.Path, "Mould.md")); err != nil {
		t.Fatal(err)
	}

	out, err := api.GetDeckNeighbourhood(t.Context(), connect.NewRequest(&v1.GetDeckNeighbourhoodRequest{
		Vault: string(v.ID), Deck: "decks/Words.md",
	}))
	if err != nil {
		t.Fatal(err)
	}

	var found *v1.DeckNeighbour
	for _, one := range out.Msg.GetNotes() {
		if one.GetPath() == "Mould.md" {
			found = one
		}
	}
	if found == nil {
		t.Fatalf("the note the deck points at is not among %+v", out.Msg.GetNotes())
	}
	if found.Error == nil {
		t.Fatal("a note that is not there came back without an error")
	}
	if got := found.GetError(); got != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("a note that is not there is refused as %v", got)
	}
	if found.GetBody() != "" {
		t.Errorf("a note that could not be read came with prose: %q", found.GetBody())
	}
}

// brokenLinks is an index that cannot say what a note points at.
type brokenLinks struct{ ports.LinkQueries }

func (brokenLinks) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errors.New("the index is out of reach")
}

// backlinked is a vault whose deck points at one note and is pointed at by
// another.
var backlinked = map[string]string{
	"Term.md": deck["Term.md"],
	"Mould.md": "---\ntype: note\n---\n" +
		"\n# Leaf mould\n\nLeaves left in a heap for two winters, and nothing else.\n",
	"Mention.md": "---\ntype: note\n---\n" +
		"\n# Mention\n\nSee [[Words]] for the cards.\n",
	"decks/Words.md": joined["decks/Words.md"],
}

// The index being out of reach is an internal failure and not an empty
// neighbourhood.
func TestAnIndexOutOfReachIsAnInternalFailureForReadingAround(t *testing.T) {
	api, held := newAPI(t, joined)
	api.scenarios().neighbourhood = flashcards.NewShowNeighbourhood(
		note.NewShowLinks(brokenLinks{}), nil, note.NewRead(api.ports.Readers))

	_, err := api.GetDeckNeighbourhood(t.Context(), connect.NewRequest(&v1.GetDeckNeighbourhoodRequest{
		Vault: string(held[0].ID), Deck: "decks/Words.md",
	}))
	if code := connect.CodeOf(err); code != connect.CodeInternal {
		t.Errorf("reading around answered %v", err)
	}
}

// A note the deck points at is said to be pointed at, and a note pointing at
// the deck is not. Every entry is a note of the vault.
func TestANoteIsSaidToBePointedAtOnlyWhereTheDeckPointsAtIt(t *testing.T) {
	api, held := newAPI(t, backlinked)

	out, err := api.GetDeckNeighbourhood(t.Context(), connect.NewRequest(&v1.GetDeckNeighbourhoodRequest{
		Vault: string(held[0].ID), Deck: "decks/Words.md",
	}))
	if err != nil {
		t.Fatal(err)
	}

	pointed := map[string]bool{}
	for _, one := range out.Msg.GetNotes() {
		if one.GetPath() == "" {
			t.Errorf("an entry stands for no note: %+v", one)
		}
		pointed[one.GetPath()] = one.GetIsPointedAt()
	}
	if is, found := pointed["Mould.md"]; !found || !is {
		t.Errorf("the note the deck points at came back as %v (found %v)", is, found)
	}
	if is, found := pointed["Mention.md"]; !found || is {
		t.Errorf("the note pointing at the deck came back as %v (found %v)", is, found)
	}
}
