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
		t.Errorf("the window was told %v (%v)", notified, inUse)
	}
	if stateErr != nil || state.Status != port.PluginIndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}
