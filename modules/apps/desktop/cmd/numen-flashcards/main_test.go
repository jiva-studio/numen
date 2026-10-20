package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

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
	cfg, db, vaults, _, held := makeWindow(t)
	notes := makeNotes(cfg, db, vaults)
	api := makeWindowAPI(cfg, notes)

	bindVaultIndexer(t.Context(), api, vaults)

	if vaults.RecordVault == nil {
		t.Fatal("a vault brought level is recorded nowhere")
	}
	vaults.RecordVault(held[0])
}

// The plugins are lent every vault the installation holds, each worked by its identity, and a vault
// that is not in the index is read into it when a plugin asks.
func TestThePluginsAreLentTheVaultsOfTheInstallation(t *testing.T) {
	cfg, db, vaults, _, held := makeWindow(t)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range held {
		if err := registry.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))
	api.SetVaultIndexer(t.Context(), nil)
	var told []domain.VaultID
	api.Opened = func(_ context.Context, v domain.Vault) { told = append(told, v.ID) }

	lent, err := openPluginVault(cfg, registry, db, api)
	if err != nil {
		t.Fatal(err)
	}

	known, listErr := lent.GetVaults()
	bound, withErr := lent.With(held[1].ID)
	_, shown := lent.GetVault()
	inUse := bound.SetInUse(t.Context())
	state, stateErr := bound.EnsureIndexed(t.Context())

	if listErr != nil || len(known) != 2 || withErr != nil || shown {
		t.Fatalf("got %v (%v), %v and the window showing a vault: %v", known, listErr, withErr, shown)
	}
	if inUse != nil || len(told) != 1 || told[0] != held[1].ID {
		t.Errorf("the window was told %v (%v)", told, inUse)
	}
	if stateErr != nil || state.Status != port.PluginIndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}
