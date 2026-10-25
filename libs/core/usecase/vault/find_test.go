package vault_test

import (
	"errors"
	"testing"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
)

func TestAVaultIsFoundByNameFolderOrIdentity(t *testing.T) {
	t.Parallel()
	_, second, registry := twoVaults(t)
	find := vaults.NewFind(registry)

	for _, ask := range []string{second.Name, second.Path, string(second.ID)} {
		got, err := find.Execute(ask)
		if err != nil || got.ID != second.ID {
			t.Errorf("%q resolved to %+v, %v", ask, got, err)
		}
	}
	if _, err := find.Execute("nothing"); !errors.Is(err, vaults.ErrUnknown) {
		t.Errorf("a vault nobody holds resolved: %v", err)
	}
}

func TestNamingNoVaultIsRefused(t *testing.T) {
	t.Parallel()
	_, _, registry := twoVaults(t)

	if _, err := vaults.NewFind(registry).Execute(""); !errors.Is(err, vaults.ErrNoName) {
		t.Errorf("no name resolved: %v", err)
	}
}
