package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func TestAVaultIsResolvedByNameFolderOrIdentity(t *testing.T) {
	t.Parallel()
	first, second, registry := twoVaults(t)
	resolve := vaults.NewResolveVault(vaults.NewFind(registry))

	for _, ask := range []string{second.Name, second.Path, string(second.ID)} {
		got, err := resolve.Execute(ask, first.ID)
		if err != nil || got.ID != second.ID {
			t.Errorf("%q resolved to %+v, %v", ask, got, err)
		}
	}
	if _, err := resolve.Execute("nothing", ""); !errors.Is(err, vaults.ErrUnknown) {
		t.Errorf("a vault nobody holds resolved: %v", err)
	}
}

func TestNamingNoVaultIsRefused(t *testing.T) {
	t.Parallel()
	_, _, registry := twoVaults(t)

	if _, err := vaults.NewResolveVault(vaults.NewFind(registry)).Execute("", ""); !errors.Is(err, vaults.ErrNoName) {
		t.Errorf("no name resolved: %v", err)
	}
}

func TestTheVaultBeingShownIsRefusedAndStillReturned(t *testing.T) {
	t.Parallel()
	first, _, registry := twoVaults(t)
	resolve := vaults.NewResolveVault(vaults.NewFind(registry))

	got, err := resolve.Execute(first.Name, first.ID)
	if !errors.Is(err, vaults.ErrShowing) || got.ID != first.ID {
		t.Errorf("the vault being shown resolved to %+v, %v", got, err)
	}
	if _, err := resolve.Execute(first.Name, ""); err != nil {
		t.Errorf("asking with no vault shown refused: %v", err)
	}
}

type dialog struct {
	chose string
	did   bool
	err   error
}

func (d dialog) ChooseFolder(context.Context, string, string) (string, bool, error) {
	return d.chose, d.did, d.err
}

var _ port.FolderDialog = dialog{}

func TestAFolderIsPickedOnlyWhenNoneIsNamed(t *testing.T) {
	t.Parallel()
	pick := vaults.NewPickFolder(dialog{chose: "/picked", did: true})

	if got, err := pick.Execute(t.Context(), "/named"); got.Path != "/named" || !got.IsChosen || err != nil {
		t.Errorf("a named folder came back as %+v, %v", got, err)
	}
	if got, err := pick.Execute(t.Context(), ""); got.Path != "/picked" || !got.IsChosen || err != nil {
		t.Errorf("a picked folder came back as %+v, %v", got, err)
	}
	if got, err := vaults.NewPickFolder(dialog{}).Execute(t.Context(), ""); got.IsChosen || err != nil {
		t.Errorf("a closed dialog came back as %+v, %v", got, err)
	}
	if _, err := vaults.NewPickFolder(nil).Execute(t.Context(), ""); err == nil {
		t.Error("a machine with no dialog picked a folder")
	}
}
