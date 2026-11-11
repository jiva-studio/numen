package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getNotes() map[string]string {
	return map[string]string{
		"Leaf mould.md":  "# Leaf mold\n\nCompost made of fallen leaves alone.\n",
		"garden/Beds.md": "# Beds\n\n[[Leaf mold]] goes on in autumn.\n",
	}
}

func makeWindow(
	t *testing.T,
) (container.Config, *container.Index, *vault.OpenVaults[*container.Index], *shell.API, []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(state, "index.db"), RegistryPath: filepath.Join(state, "vaults.json")},
	})
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	target := []domain.Vault{addVaultOf(t, cfg, "One"), addVaultOf(t, cfg, "Two")}
	vaults := &vault.OpenVaults[*container.Index]{
		Opener: func(i *container.Index) *vault.VaultOpener { return container.VaultOpener(cfg, i) }, Index: db,
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return cfg, db, vaults, &shell.API{}, target
}

func addVaultOf(t *testing.T, cfg container.Config, name string) domain.Vault {
	t.Helper()

	root := t.TempDir()
	for name, body := range getNotes() {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	added, err := cfg.AddVault(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return added
}

// The page is given whether a vault is indexed, the window it draws and the plugins' services it
// forwards.
func TestThePageIsGivenWhatItWorksThrough(t *testing.T) {
	cfg, db, _, _, _ := makeWindow(t)

	api := makeWindowAPI(cfg)
	container.SetShellNotes(api, db)

	if api == nil {
		t.Fatal("the page has no API")
	}
	if api.PluginServices == nil {
		t.Error("the page forwards no services")
	}
	for name, wired := range map[string]any{
		"indexed": api.Ensure,
		"window":  api.Window,
	} {
		if reflect.ValueOf(wired).IsZero() {
			t.Errorf("the page has no %s", name)
		}
	}
}

// The page resolves the links of a note against the vaults the application lists.
func TestTheFlashcardsPageIsGivenTheLinksOfItsVaults(t *testing.T) {
	state := t.TempDir()
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(state, "index.db"), RegistryPath: filepath.Join(state, "vaults.json")},
	})
	addVaultOf(t, cfg, "One")
	api := makeWindowAPI(cfg)

	started, err := container.OpenFlashcards(cfg, t.Context(), api, io.Discard, "flashcards", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		started.CloseVaults()
		if err := started.CloseIndex(); err != nil {
			t.Error(err)
		}
	})

	if api.ResolveLinks == nil {
		t.Error("the page has no links")
	}
}

// A vault the index was updated for is counted again, and the walk of a
// vault is the one the window's own opening makes.
func TestTheVaultIndexerIsBoundToTheWindow(t *testing.T) {
	cfg, db, vaults, _, target := makeWindow(t)
	api := makeWindowAPI(cfg)
	container.SetShellNotes(api, db)

	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		return errors.New("boom")
	})
	for {
		state, err := api.EnsureIndexed(t.Context(), target[0])
		if err != nil {
			t.Fatal(err)
		}
		if state.Reason == "boom" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	api.SetVaultIndexer(t.Context(), vaults.IndexVault)

	moved, err := api.WatchMoves(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	api.ReportVaultRead(target[0])
	select {
	case <-moved:
	case <-time.After(time.Second):
		t.Error("a vault read did not report a change")
	}

	for {
		state, err := api.EnsureIndexed(t.Context(), target[0])
		if err != nil {
			t.Fatal(err)
		}
		if state.Status == plugin.IndexFailed {
			t.Fatalf("expected SetVaultIndexer to replace failing indexer, got %+v", state)
		}
		if state.Status == plugin.Indexed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// The plugins receive every vault the installation holds, each keyed by its
// identity, and a vault that is not in the index is read into it when a plugin
// requests it.
func TestThePluginsAreLentTheVaultsOfTheInstallation(t *testing.T) {
	cfg, db, _, _, target := makeWindow(t)
	api := makeWindowAPI(cfg)
	api.SetVaultIndexer(t.Context(), nil)

	host, err := container.NewPluginHost("md.numen.test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.PluginHost = host
	if err := container.InstallPluginVault(cfg, db, getNoCurrentVault, getPluginVaultHooks(api)); err != nil {
		t.Fatal(err)
	}
	lent := cfg.PluginHost.Endpoint.GetPluginVault()

	recognized, listErr := lent.Switch.GetVaults()
	linked, withErr := lent.Switch.With(target[1].ID)
	_, hasVault := lent.Switch.GetVault()
	state, stateErr := linked.Switch.EnsureIndexed(t.Context())

	if listErr != nil || len(recognized) != 2 || withErr != nil || hasVault {
		t.Fatalf("got %v (%v), %v and the window showing a vault: %v", recognized, listErr, withErr, hasVault)
	}
	if stateErr != nil || state.Status != plugin.IndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}

// The page receives the plugins of the flashcards target and is served the
// files of their frontends.
func TestThePageLoadsThePluginsOfItsTarget(t *testing.T) {
	cfg, _, _, _, _ := makeWindow(t)
	api := makeWindowAPI(cfg)

	if api.GetFrontendPluginIDs == nil || api.GetPluginDir == nil {
		t.Error("the page is not told the plugins it loads")
	}
	if api.Window.Name != platform.TargetFlashcards {
		t.Errorf("the window answers as %q", api.Window.Name)
	}
}

func getNoCurrentVault() (domain.Vault, bool) { return domain.Vault{}, false }

func getPluginVaultHooks(api *shell.API) container.PluginVaultHooks {
	return container.PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
}
