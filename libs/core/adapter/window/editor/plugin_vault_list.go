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
	checkRoot    *vaults.CheckRoot
}

// NewPluginVaultList is the list of vaults of the installation as plugins change it: through the
// use cases on vaults the window answers with, a folder refused by checkRoot before it is added,
// and the vault the window shows changed through the state.
func NewPluginVaultList(o *Installation, checkRoot *vaults.CheckRoot) pluginport.PluginVaultList {
	return pluginVaultList{installation: o, state: o.State, scenarios: o.Vaults(), checkRoot: checkRoot}
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

// OpenVault refuses a vault whose folder is not the vault, as a change of vault does, and asks the
// pages to save their unsaved changes. It refuses when a page holds changes the person has to decide
// on, and while the window is closing or another change of vault is deciding. The change of vault it
// returns asks the pages again, so what the person typed in between is saved too.
func (l pluginVaultList) OpenVault(ctx context.Context, id domain.VaultID) (func(context.Context) error, error) {
	v, err := l.scenarios.Find.Execute(string(id))
	if err != nil {
		return nil, err
	}
	if l.state.OpenVault == nil {
		return nil, errNoVaultSwap
	}
	if err := l.installation.checkIdentity.Execute(v); err != nil {
		return nil, err
	}
	if err := l.decide(ctx, id); err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		err := l.state.OpenVault(ctx, v)
		if errors.Is(err, errAnswerPending) {
			return fmt.Errorf("open vault %s: %w", id, pluginport.ErrVaultAsking)
		}
		return err
	}, nil
}

// errFlushRunning is a decision refused because the window has a request to the pages running, such
// as a close waiting for the person to answer a page's question.
var errFlushRunning = errors.New("the window is waiting for its pages")

// decide asks the pages to save their unsaved changes under the exclusion a close and a change of
// vault take, so it never cuts their request short, and refuses while a request of theirs runs. A
// close, a quit or a change of vault of the person cancels it, as does the end of the call, and it
// answers ErrUnavailable. A page holding a question keeps the request open, as the change of vault
// does, so the person answers it.
func (l pluginVaultList) decide(ctx context.Context, id domain.VaultID) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	end, err := l.installation.shutdown.tryBeginDecision(cancel)
	if err != nil {
		return fmt.Errorf("open vault %s: %w: %w", id, pluginport.ErrUnavailable, err)
	}
	defer end()
	if l.state.Window.IsFlushRunning() {
		return fmt.Errorf("open vault %s: %w: %w", id, pluginport.ErrUnavailable, errFlushRunning)
	}
	limited, stop := context.WithTimeout(ctx, FlushTimeout)
	defer stop()
	isFlushed := l.state.Window.RequestFlush(limited)
	switch {
	case errors.Is(context.Cause(ctx), errPersonFirst):
		return fmt.Errorf("open vault %s: %w: %w", id, pluginport.ErrUnavailable, errPersonFirst)
	case ctx.Err() != nil:
		// The request belongs to a call that is gone, and would refuse the next decision.
		l.state.Window.CancelFlush()
		return fmt.Errorf("open vault %s: %w: %w", id, pluginport.ErrUnavailable, ctx.Err())
	case !isFlushed:
		return fmt.Errorf("open vault %s: %w", id, pluginport.ErrVaultAsking)
	}
	l.state.Window.CancelFlush()
	return nil
}

func (l pluginVaultList) IsMissing(v domain.Vault) bool {
	return l.scenarios.FolderCheck.Execute(v)
}
