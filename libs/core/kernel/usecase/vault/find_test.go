package vault_test

import (
	"errors"
	"testing"

	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

func TestAVaultIsFoundByNameFolderOrIdentity(t *testing.T) {
	t.Parallel()
	_, second, registry := twoVaults(t)
	find := vaultregistry.NewFind(registry)

	for _, ask := range []string{second.Name, second.Path, string(second.ID)} {
		got, err := find.Execute(ask)
		if err != nil || got.ID != second.ID {
			t.Errorf("%q resolved to %+v, %v", ask, got, err)
		}
	}
	if _, err := find.Execute("nothing"); !errors.Is(err, vaultregistry.ErrUnknown) {
		t.Errorf("a vault nobody holds resolved: %v", err)
	}
}

func TestNamingNoVaultIsRefused(t *testing.T) {
	t.Parallel()
	_, _, registry := twoVaults(t)

	if _, err := vaultregistry.NewFind(registry).Execute(""); !errors.Is(err, vaultregistry.ErrNoName) {
		t.Errorf("no name resolved: %v", err)
	}
}
