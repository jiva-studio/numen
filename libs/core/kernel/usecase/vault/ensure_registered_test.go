package vault_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func newEnsure(t *testing.T, registry *appstate.VaultRegistry) vaults.EnsureRegistered {
	t.Helper()
	identity := vaultreader.Identity{}
	add := vaults.NewAdd(identity, registry, registry, registry, testsupport.NewClock())
	return vaults.NewEnsureRegistered(identity, registry, add, registry)
}

func TestEnsureRegistered_ReturnsAFolderOnTheListAsItIs(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	root := folder(t, "orchard")
	listed, err := add.Execute(root, "")
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}

	got, err := newEnsure(t, registry).Execute(vaults.EnsureRequest{Root: link, Name: "other", IsRecordedOpened: true})

	if err != nil || got.ID != listed.ID {
		t.Fatalf("got %+v (%v), want the vault %+v", got, err, listed)
	}
	if all, _ := registry.List(); len(all) != 1 {
		t.Errorf("the list holds %d vaults, want 1", len(all))
	}
	if _, isRecorded, _ := registry.GetLastVault(); isRecorded {
		t.Error("a vault on the list was recorded as opened")
	}
}

func TestEnsureRegistered_AddsAFolderMissingFromTheList(t *testing.T) {
	t.Parallel()
	registry := registryAt(t)
	root := folder(t, "orchard")

	got, err := newEnsure(t, registry).Execute(vaults.EnsureRequest{Root: root, Name: "numen"})

	if err != nil || got.Name != "numen" {
		t.Fatalf("got %+v (%v), want a vault called numen", got, err)
	}
	if all, _ := registry.List(); len(all) != 1 {
		t.Errorf("the list holds %d vaults, want 1", len(all))
	}
	if _, isRecorded, _ := registry.GetLastVault(); isRecorded {
		t.Error("a vault was recorded as opened without being asked")
	}
}

func TestEnsureRegistered_RecordsAnAddedVaultWhenAsked(t *testing.T) {
	t.Parallel()
	registry := registryAt(t)

	got, err := newEnsure(t, registry).Execute(vaults.EnsureRequest{Root: folder(t, "pond"), IsRecordedOpened: true})
	if err != nil {
		t.Fatal(err)
	}

	if last, isRecorded, _ := registry.GetLastVault(); !isRecorded || last.ID != got.ID {
		t.Errorf("the vault opened last is %+v, want %+v", last, got)
	}
}

func TestEnsureRegistered_AddsNothingAfterAFailedLookup(t *testing.T) {
	t.Parallel()
	registry := registryAt(t)
	if err := os.MkdirAll(filepath.Dir(registry.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry.Path(), []byte("not a list"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := newEnsure(t, registry).Execute(vaults.EnsureRequest{Root: folder(t, "pond")}); err == nil {
		t.Error("the lookup failed and a vault was returned")
	}
	if raw, _ := os.ReadFile(registry.Path()); string(raw) != "not a list" {
		t.Errorf("the registry file is now %q", raw)
	}
}
