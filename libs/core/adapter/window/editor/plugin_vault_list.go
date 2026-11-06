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
	installation *Installation
	state        *State
	scenarios    *vaults.Scenarios
}

// NewPluginVaultList is the list of vaults of the installation as plugins change it: through the
// use cases on vaults the window answers with, and the vault the window shows changed through the
// state.
func NewPluginVaultList(o *Installation) pluginport.PluginVaultList {
	return pluginVaultList{installation: o, state: o.State, scenarios: o.Vaults()}
}

func (l pluginVaultList) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
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

// OpenVault rejects a vault whose folder is not the vault, as a change of vault does, and switches
// the window through State.OpenVault, which flushes the pages first. It runs under the exclusion a
// close and a change of vault take and never ends their flush request. A close, a quit or a change
// of vault of the user cancels it, as does the end of the call, and it returns ErrUnavailable. A
// page that answers ASKING keeps the flush request open for the user's answer, and OpenVault
// returns ErrUnsavedChanges; while that question is unanswered, it returns ErrUnsavedChanges
// without a flush.
func (l pluginVaultList) OpenVault(ctx context.Context, id domain.VaultID) error {
	v, err := l.scenarios.Find.Execute(string(id))
	if err != nil {
		return err
	}
	if l.state.OpenVault == nil {
		return errNoVaultSwap
	}
	if err := l.installation.checkIdentity.Execute(v); err != nil {
		return err
	}
	ctx, end, err := l.installation.shutdown.tryBeginPluginOpen(ctx)
	if err != nil {
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, pluginport.ErrUnavailable, err)
	}
	defer end()
	if l.state.Window.IsQuestionOpen() {
		return pluginport.ErrUnsavedChanges
	}
	err = l.state.OpenVault(ctx, v)
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		if !l.state.Window.IsQuestionOpen() {
			l.state.Window.CancelFlush()
		}
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, pluginport.ErrUnavailable, context.Cause(ctx))
	case errors.Is(err, errAnswerPending):
		return pluginport.ErrUnsavedChanges
	}
	return err
}

func (l pluginVaultList) IsMissing(v domain.Vault) bool {
	return l.scenarios.FolderCheck.Execute(v)
}
