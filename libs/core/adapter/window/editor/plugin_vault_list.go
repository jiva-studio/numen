package editor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// errNoVaultSwap is a window whose build binds no change of vault.
var errNoVaultSwap = errors.New("this window cannot show another vault")

// pluginVaultList is the installation's list of vaults and the vault the window of the state shows,
// as the plugins that call the host change them.
type pluginVaultList struct {
	state     *State
	scenarios *vaults.Scenarios
	checkRoot *vaults.CheckRoot
}

// NewPluginVaultList is the list of vaults of the installation as plugins change it: through the
// use cases on vaults the window answers with, a folder refused by checkRoot before it is added,
// and the vault the window shows changed through the state.
func NewPluginVaultList(o *Installation, checkRoot *vaults.CheckRoot) pluginport.PluginVaultList {
	return pluginVaultList{state: o.State, scenarios: o.Vaults(), checkRoot: checkRoot}
}

func (l pluginVaultList) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
	if err := l.checkRoot.Execute(root); err != nil {
		return domain.Vault{}, fmt.Errorf("%w: %w", vaults.ErrUnreadable, err)
	}
	return l.scenarios.Add.Execute(root, name)
}

func (l pluginVaultList) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	v, err := l.scenarios.Find.Execute(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	return l.scenarios.Rename.Execute(ctx, v, name)
}

// RemoveVault takes the vault off the list and out of the index, and leaves its folder where it is.
func (l pluginVaultList) RemoveVault(ctx context.Context, id domain.VaultID) error {
	return l.scenarios.Remove.Execute(ctx, string(id), l.state.GetCurrentVault().ID, false)
}

func (l pluginVaultList) OpenVault(ctx context.Context, id domain.VaultID) error {
	v, err := l.scenarios.Find.Execute(string(id))
	if err != nil {
		return err
	}
	if l.state.OpenVault == nil {
		return errNoVaultSwap
	}
	err = l.state.OpenVault(ctx, v)
	if errors.Is(err, errAnswerPending) {
		return fmt.Errorf("open vault %s: %w", id, pluginport.ErrVaultAsking)
	}
	return err
}

func (l pluginVaultList) IsMissing(v domain.Vault) bool {
	return l.scenarios.FolderCheck.Execute(v)
}
