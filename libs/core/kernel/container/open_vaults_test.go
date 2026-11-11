package container

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	vaults, target := makeOpenVaults(t)
	v := target[0]

	var walks sync.WaitGroup
	for range 8 {
		walks.Go(func() {
			if err := vaults.IndexVault(t.Context(), v); err != nil {
				t.Error(err)
			}
		})
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

	if err := vaults.IndexVault(t.Context(), target[0]); !errors.Is(err, vault.ErrClosing) {
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
	if !errors.Is(err, vault.ErrClosing) {
		t.Errorf("an index update asked for while closing came back with %v", err)
	}
}

// A vault index is updated through the opening it was walked with, and neither opens
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
		t.Errorf("updating and walking opened the vault %d times", vaults.OpenedCount())
	}
}

func getOpenVaultsSamples() map[string]string {
	return map[string]string{
		"Term.md": "---\ntype: form\n---\n" +
			"\n## Layout\n\n### Header\n\nWord\n\n### Footer\n\nMeaning\n",
		"samples/Words.md": "---\ntype: sample\n---\n" +
			"\n## Leaf mold ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mold\n" +
			"\n### Meaning\n\nCompost made of fallen leaves alone\n",
	}
}

func makeOpenVaults(t *testing.T) (*vault.OpenVaults[*Index], []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := NewConfig(Config{
		Paths: Paths{IndexPath: filepath.Join(state, "index.db"), RegistryPath: filepath.Join(state, "vaults.json")},
	})
	db, err := openIndex(vaultOf(cfg), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: makeSampleVault(t, cfg)},
		{ID: "two", Name: "Two", Path: makeSampleVault(t, cfg)},
	}
	vaults := &vault.OpenVaults[*Index]{
		Opener: func(i *Index) *vault.VaultOpener { return vaultOpener(vaultOf(cfg), i) }, Index: db,
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return vaults, target
}

func makeSampleVault(t *testing.T, cfg Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := vaultIdentity(vaultOf(cfg)).EnsureIdentity(root, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	for name, body := range getOpenVaultsSamples() {
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
