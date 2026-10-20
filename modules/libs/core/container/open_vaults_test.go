package container

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	vaults, held := makeOpenVaults(t)
	v := held[0]

	var walks sync.WaitGroup
	for range 8 {
		walks.Add(1)
		go func() {
			defer walks.Done()
			if err := vaults.IndexVault(t.Context(), v); err != nil {
				t.Error(err)
			}
		}()
	}
	walks.Wait()

	// The opening is one, so the watch behind it is one.
	if len(vaults.opened) != 1 {
		t.Errorf("the window opened the vault %d times", len(vaults.opened))
	}
}

// A walk asked for once the window has begun closing is refused, so none begins
// after the index has been waited for.
func TestClosingWindowReadsNothing(t *testing.T) {
	vaults, held := makeOpenVaults(t)

	vaults.CloseAndWait()

	if err := vaults.IndexVault(t.Context(), held[0]); !errors.Is(err, ErrClosing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
	}
}

// A levelling asked for once the window has begun closing is refused, so none
// begins writing to the index after it has been waited for. The prose is on
// disk either way, and the window says the note is written and unlevelled.
func TestClosingWindowUpdatesNothing(t *testing.T) {
	vaults, held := makeOpenVaults(t)

	vaults.CloseAndWait()

	err := vaults.UpdateIndex(t.Context(), held[0], []string{"decks/Words.md"})
	if !errors.Is(err, ErrClosing) {
		t.Errorf("a levelling asked for while closing came back with %v", err)
	}
}

// A vault is levelled through the opening it was walked with, and neither opens
// the vault a second time.
func TestUpdateIndexUsesTheVaultsOwnOpenedVault(t *testing.T) {
	vaults, held := makeOpenVaults(t)
	v := held[0]

	if err := vaults.IndexVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if err := vaults.UpdateIndex(t.Context(), v, []string{"decks/Words.md"}); err != nil {
		t.Fatal(err)
	}
	if len(vaults.opened) != 1 {
		t.Errorf("levelling and walking opened the vault %d times", len(vaults.opened))
	}
}

// openVaultsDeck is a vault holding one stencil and one deck.
var openVaultsDeck = map[string]string{
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// makeOpenVaults is an installation of its own, an index it may write, and two
// vaults to open.
func makeOpenVaults(t *testing.T) (*OpenVaults, []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := Config{
		IndexPath:    filepath.Join(state, "index.db"),
		RegistryPath: filepath.Join(state, "vaults.json"),
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	held := []domain.Vault{
		{ID: "one", Name: "One", Path: makeDeckVault(t, cfg)},
		{ID: "two", Name: "Two", Path: makeDeckVault(t, cfg)},
	}
	vaults := &OpenVaults{
		Config: cfg, Index: db, Lifetime: t.Context(),
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return vaults, held
}

// makeDeckVault is a folder holding one stencil and one deck, as a vault.
func makeDeckVault(t *testing.T, cfg Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().EnsureIdentity(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	for name, body := range openVaultsDeck {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
