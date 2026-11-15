package windowsession

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// errNoVaultSwitch is what a window build that binds no vault switch gives.
var errNoVaultSwitch = errors.New("the window cannot switch vaults")

// OpenPluginVault rejects a vault whose folder is not the vault, then switches the window through the
// switch it is given, which flushes the pages first.
// It runs under the lock of a close and a vault switch and never ends their
// flush request; a close, a quit, a user vault switch or the end of the call
// cancels it with ErrUnavailable. An open ASKING prompt returns ErrUnsavedChanges without a flush.
type OpenPluginVault struct {
	guard    *Guard
	identity IdentityCheck
	pages    Pages
	switcher VaultSwitch
}

func NewOpenPluginVault(guard *Guard, identity IdentityCheck, pages Pages, switcher VaultSwitch) OpenPluginVault {
	return OpenPluginVault{guard: guard, identity: identity, pages: pages, switcher: switcher}
}

// Execute opens v on behalf of a plugin; the vault switch decides whether the window can switch.
func (u OpenPluginVault) Execute(ctx context.Context, v domain.Vault) error {
	if err := u.identity.Execute(v); err != nil {
		return err
	}
	ctx, open, end, err := u.guard.tryBeginPluginOpen(ctx)
	if err != nil {
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, err)
	}
	defer end()
	if u.pages.IsQuestionOpen() {
		return plugin.ErrUnsavedChanges
	}
	err = u.switcher.Execute(ctx, v, open)
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		if !u.pages.IsQuestionOpen() {
			u.pages.CancelFlush()
		}
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, context.Cause(ctx))
	case errors.Is(err, window.ErrAnswerPending):
		return plugin.ErrUnsavedChanges
	}
	return err
}

// VaultSwitch shows a vault in the window. `open` is the plugin open whose lock the switch shares.
type VaultSwitch interface {
	Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error
}
