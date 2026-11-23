package container

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	vaults, target, openings := makeOpenVaults(t)
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
	if got := openings.Load(); got != 1 {
		t.Errorf("the window opened the vault %d times", got)
	}
}

// A closing window walks no vault after the index has been waited for.
func TestClosingWindowReadsNothing(t *testing.T) {
	vaults, target, _ := makeOpenVaults(t)

	vaults.CloseAndWait()

	if err := vaults.IndexVault(t.Context(), target[0]); !errors.Is(err, indexing.ErrClosing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
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

func makeOpenVaults(t *testing.T) (*indexing.OpenVaults[*Index], []domain.Vault, *atomic.Int32) {
	t.Helper()

	state := t.TempDir()
	cfg := NewTestInstallation(Config{
		Paths: Paths{
			IndexPath:    filepath.Join(state, "index.db"),
			RegistryPath: filepath.Join(state, "vaults.json"),
		},
	}, nil)
	db, err := openIndex(cfg.buildVaultWiring(), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: makeSampleVault(t, cfg)},
		{ID: "two", Name: "Two", Path: makeSampleVault(t, cfg)},
	}
	var openings atomic.Int32
	vaults := &indexing.OpenVaults[*Index]{
		CreateOpener: func(i *Index) *indexing.VaultOpener {
			openings.Add(1)
			return createVaultOpener(cfg.buildVaultWiring(), i)
		},
		Index:       db,
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return vaults, target, &openings
}

func makeSampleVault(t *testing.T, cfg TestInstallation) string {
	t.Helper()

	root := t.TempDir()
	if _, err := createVaultIdentity(cfg.buildVaultWiring().Options).EnsureIdentity(root, testsupport.GetEpoch()); err != nil {
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
