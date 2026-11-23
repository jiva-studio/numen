package vault_test

import (
	"testing"
)

const managingPlugin = "md.numen.tools.sync"

func TestAManagedVaultKeepsItsManagerOnTheList(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)

	added, err := add.ExecuteManaged(folder(t, "Food"), "Food", managingPlugin)
	if err != nil {
		t.Fatal(err)
	}

	if added.ManagedBy != managingPlugin {
		t.Errorf("added.ManagedBy = %q, want %q", added.ManagedBy, managingPlugin)
	}
	listed, err := registry.List()
	if err != nil || len(listed) != 1 || listed[0].ManagedBy != managingPlugin {
		t.Errorf("list = %+v, error = %v, want the one vault managed by %s", listed, err, managingPlugin)
	}
}

func TestAnUnmanagedVaultHasNoManager(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)

	if _, err := add.Execute(folder(t, "notes"), ""); err != nil {
		t.Fatal(err)
	}

	listed, err := registry.List()
	if err != nil || len(listed) != 1 || listed[0].ManagedBy != "" {
		t.Errorf("list = %+v, error = %v, want one vault with no manager", listed, err)
	}
}

func TestAManagedVaultStillRefusesAFolderInsideAnotherVault(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	outer := folder(t, "outer")
	if _, err := add.Execute(outer, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := add.ExecuteManaged(outer+"/inner", "inner", managingPlugin); err == nil {
		t.Error("a managed vault was added inside another vault")
	}
}
