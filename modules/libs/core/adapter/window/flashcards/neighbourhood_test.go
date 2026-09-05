package flashcards

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
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
// question about another is refused rather than answered off some other vault.
func TestReadingAroundIsRefusedForAVaultThisInstallationDoesNotHold(t *testing.T) {
	api, _ := windowed(t, joined)

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
	if !errors.Is(err, ErrNoVault) {
		t.Errorf("refused because %v", err)
	}
}

// A note the deck points at that cannot be read comes back named the way the
// editor names it, so a person meets one vocabulary for a refused note and not
// one for each window.
func TestANoteThatCannotBeReadIsRefusedAsTheEditorRefusesIt(t *testing.T) {
	api, held := windowed(t, joined)
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
	if found.Refusal == nil {
		t.Fatal("a note that is not there came back without a refusal")
	}
	if got := found.GetRefusal(); got != v1.Refusal_REFUSAL_MISSING {
		t.Errorf("a note that is not there is refused as %v", got)
	}
	if found.GetBody() != "" {
		t.Errorf("a note that could not be read came with prose: %q", found.GetBody())
	}
}
