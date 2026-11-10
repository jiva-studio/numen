package editor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"

	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
)

// errNoVaultSwitch is returned when the window build binds no vault switch.
var errNoVaultSwitch = errors.New("the window cannot switch vaults")

// pluginVaultList implements PluginVaultList over the window state.
type pluginVaultList struct {
	installation *Installation
	state        *State
	scenarios    *vaults.Scenarios
}

// NewPluginVaultList returns a PluginVaultList that uses the window's vault use cases and switches
// the vault the window shows through the state.
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

// RemoveVault removes the vault from the list and from the index, and does not delete its folder.
func (l pluginVaultList) RemoveVault(ctx context.Context, id domain.VaultID) error {
	return l.scenarios.Remove.Execute(ctx, string(id), l.state.GetCurrentVault().ID, false)
}

// OpenVault rejects a vault whose folder is not the vault, then switches the
// window through State.OpenVault, which flushes the pages first.
// It runs under the lock of a close and a vault switch and never ends their
// flush request; a close, a quit, a user vault switch or the end of the call
// cancels it with ErrUnavailable. An open ASKING prompt returns ErrUnsavedChanges without a flush.
func (l pluginVaultList) OpenVault(ctx context.Context, id domain.VaultID) error {
	v, err := l.scenarios.Find.Execute(string(id))
	if err != nil {
		return err
	}
	if l.state.OpenVault == nil {
		return errNoVaultSwitch
	}
	if err := l.installation.checkIdentity.Execute(v); err != nil {
		return err
	}
	ctx, open, end, err := l.installation.shutdown.tryBeginPluginOpen(ctx)
	if err != nil {
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, err)
	}
	defer end()
	if l.state.Window.IsQuestionOpen() {
		return plugin.ErrUnsavedChanges
	}
	err = l.state.OpenVault(ctx, v, open)
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		if !l.state.Window.IsQuestionOpen() {
			l.state.Window.CancelFlush()
		}
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, context.Cause(ctx))
	case errors.Is(err, errAnswerPending):
		return plugin.ErrUnsavedChanges
	}
	return err
}

func (l pluginVaultList) IsMissing(v domain.Vault) bool {
	return l.scenarios.FolderCheck.Execute(v)
}
