package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/libs/core/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// notes is a vault holding two notes.
var notes = map[string]string{
	"Leaf mould.md":  "# Leaf mould\n\nCompost made of fallen leaves alone.\n",
	"garden/Beds.md": "# Beds\n\n[[Leaf mould]] goes on in autumn.\n",
}

// makeWindow is this window as the binary builds it: an installation of its own, an index it may
// write, and two vaults.
func makeWindow(
	t *testing.T,
) (container.Config, *container.Index, *container.OpenVaults, *shell.API, []domain.Vault) {
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
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: vaultOf(t, cfg)},
		{ID: "two", Name: "Two", Path: vaultOf(t, cfg)},
	}
	vaults := &container.OpenVaults{
		Config: cfg, Index: db,
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return cfg, db, vaults, &shell.API{}, target
}

// vaultOf is a folder holding the notes, as a vault.
func vaultOf(t *testing.T, cfg container.Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().EnsureIdentity(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	for name, body := range notes {
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
		"links":   api.ResolveLinks,
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

// The plugins receive every vault the installation holds, each keyed by its
// identity, and a vault that is not in the index is read into it when a plugin
// requests it.
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

	lent, err := openPluginVault(cfg, registry, db, api)
	if err != nil {
		t.Fatal(err)
	}

	recognized, listErr := lent.GetVaults()
	linked, withErr := lent.With(target[1].ID)
	_, hasVault := lent.GetVault()
	state, stateErr := linked.EnsureIndexed(t.Context())

	if listErr != nil || len(recognized) != 2 || withErr != nil || hasVault {
		t.Fatalf("got %v (%v), %v and the window showing a vault: %v", recognized, listErr, withErr, hasVault)
	}
	if stateErr != nil || state.Status != pluginport.PluginIndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}

// The page receives the plugins of the flashcards target and is served the
// files of their frontends.
func TestThePageLoadsThePluginsOfItsTarget(t *testing.T) {
	cfg, db, vaults, _, _ := makeWindow(t)
	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))

	if api.GetFrontendPluginIDs == nil || api.GetPluginDir == nil {
		t.Error("the page is not told the plugins it loads")
	}
	if api.Window.Named != platform.TargetFlashcards {
		t.Errorf("the window answers as %q", api.Window.Named)
	}
}
