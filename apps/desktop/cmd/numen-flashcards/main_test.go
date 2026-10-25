package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	window "github.com/jiva-studio/numen/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// deck is a vault holding one stencil and one deck.
var deck = map[string]string{
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// makeWindow is this window as the binary builds it: an installation of its
// own, an index it may write, and two vaults to sit down to.
func makeWindow(
	t *testing.T,
) (container.Config, *container.Index, *container.OpenVaults, *window.API, []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := container.Config{
		IndexPath:    filepath.Join(state, "index.db"),
		RegistryPath: filepath.Join(state, "vaults.json"),
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: vaultOf(t, cfg)},
		{ID: "two", Name: "Two", Path: vaultOf(t, cfg)},
	}
	vaults := &container.OpenVaults{
		Config: cfg, Index: db, Lifetime: t.Context(),
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return cfg, db, vaults, &window.API{}, target
}

// vaultOf is a folder holding one stencil and one deck, as a vault.
func vaultOf(t *testing.T, cfg container.Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().EnsureIdentity(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	for name, body := range deck {
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

// The notes the page and the tools are served are built, not empty.
func TestTheNotesAreBuilt(t *testing.T) {
	cfg, db, vaults, _, _ := makeWindow(t)

	notes := makeNotes(cfg, db, vaults)

	if reflect.ValueOf(notes).IsZero() {
		t.Error("no notes were built")
	}
}

// The page is given the notes it answers about, the window it draws and the plugins' services it
// forwards.
func TestThePageIsGivenWhatItWorksThrough(t *testing.T) {
	cfg, db, vaults, _, _ := makeWindow(t)
	notes := makeNotes(cfg, db, vaults)

	api := makeWindowAPI(cfg, notes)

	if api == nil {
		t.Fatal("the page has no API")
	}
	if api.PluginServices == nil {
		t.Error("the page forwards no services")
	}
	for name, wired := range map[string]any{
		"indexed": api.IsVaultIndexed,
		"window":  api.Window,
	} {
		if reflect.ValueOf(wired).IsZero() {
			t.Errorf("the page has no %s", name)
		}
	}
}

// A vault the index was brought level with is counted again, and the walk of a
// vault is the one the window's own opening makes.
func TestTheVaultIndexerIsBoundToTheWindow(t *testing.T) {
	cfg, db, vaults, _, target := makeWindow(t)
	notes := makeNotes(cfg, db, vaults)
	api := makeWindowAPI(cfg, notes)

	bindVaultIndexer(t.Context(), api, vaults)

	if vaults.RecordVault == nil {
		t.Fatal("a vault brought level is recorded nowhere")
	}
	vaults.RecordVault(target[0])
}

// The plugins are lent every vault the installation holds, each worked by its identity, and a vault
// that is not in the index is read into it when a plugin asks.
func TestThePluginsAreLentTheVaultsOfTheInstallation(t *testing.T) {
	cfg, db, vaults, _, target := makeWindow(t)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range target {
		if err := registry.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))
	api.SetVaultIndexer(t.Context(), nil)
	var notified []domain.VaultID
	api.Opened = func(_ context.Context, v domain.Vault) { notified = append(notified, v.ID) }

	lent, err := openPluginVault(cfg, registry, db, api)
	if err != nil {
		t.Fatal(err)
	}

	recognized, listErr := lent.GetVaults()
	linked, withErr := lent.With(target[1].ID)
	_, hasVault := lent.GetVault()
	inUse := linked.SetInUse(t.Context())
	state, stateErr := linked.EnsureIndexed(t.Context())

	if listErr != nil || len(recognized) != 2 || withErr != nil || hasVault {
		t.Fatalf("got %v (%v), %v and the window showing a vault: %v", recognized, listErr, withErr, hasVault)
	}
	if inUse != nil || len(notified) != 1 || notified[0] != target[1].ID {
		t.Errorf("got window update %v (%v)", notified, inUse)
	}
	if stateErr != nil || state.Status != pluginport.PluginIndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}
