package container

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	vaults, target := makeOpenVaults(t)
	v := target[0]

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
	if vaults.OpenedCount() != 1 {
		t.Errorf("the window opened the vault %d times", vaults.OpenedCount())
	}
}

// after the index has been waited for.
func TestClosingWindowReadsNothing(t *testing.T) {
	vaults, target := makeOpenVaults(t)

	vaults.CloseAndWait()

	if err := vaults.IndexVault(t.Context(), target[0]); !errors.Is(err, ErrClosing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
	}
}

// TestClosingWindowUpdatesNothing checks that a closing window does not begin
// writing to the index after it has been waited for. The text is on disk either
// way, and the window reports the note as written and not yet indexed.
func TestClosingWindowUpdatesNothing(t *testing.T) {
	vaults, target := makeOpenVaults(t)

	vaults.CloseAndWait()

	err := vaults.UpdateIndex(t.Context(), target[0], []string{"samples/Words.md"})
	if !errors.Is(err, ErrClosing) {
		t.Errorf("a levelling asked for while closing came back with %v", err)
	}
}

// A vault is levelled through the opening it was walked with, and neither opens
// the vault a second time.
func TestUpdateIndexUsesTheVaultsOwnOpenedVault(t *testing.T) {
	vaults, target := makeOpenVaults(t)
	v := target[0]

	if err := vaults.IndexVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if err := vaults.UpdateIndex(t.Context(), v, []string{"samples/Words.md"}); err != nil {
		t.Fatal(err)
	}
	if vaults.OpenedCount() != 1 {
		t.Errorf("levelling and walking opened the vault %d times", vaults.OpenedCount())
	}
}

// openVaultsSamples is a vault holding a note of type form and a note of type
// sample that links to it.
var openVaultsSamples = map[string]string{
	"Term.md": "---\ntype: form\n---\n" +
		"\n## Layout\n\n### Header\n\nWord\n\n### Footer\n\nMeaning\n",
	"samples/Words.md": "---\ntype: sample\n---\n" +
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

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: makeSampleVault(t, cfg)},
		{ID: "two", Name: "Two", Path: makeSampleVault(t, cfg)},
	}
	vaults := &OpenVaults{
		Config: cfg, Index: db,
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return vaults, target
}

// makeSampleVault is a folder holding openVaultsSamples, as a vault.
func makeSampleVault(t *testing.T, cfg Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().EnsureIdentity(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	for name, body := range openVaultsSamples {
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
