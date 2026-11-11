package windowsession

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// errNoVaultSwitch is returned when the window build binds no vault switch.
var errNoVaultSwitch = errors.New("the window cannot switch vaults")

// OpenPluginVault rejects a vault whose folder is not the vault, then switches the window through its
// switch, which flushes the pages first.
// It runs under the lock of a close and a vault switch and never ends their
// flush request; a close, a quit, a user vault switch or the end of the call
// cancels it with ErrUnavailable. An open ASKING prompt returns ErrUnsavedChanges without a flush.
type OpenPluginVault struct {
	life *lifecycle

	// Switch shows the vault in the window. A window with none cannot open a vault for a plugin.
	Switch VaultSwitch
}

// VaultSwitch shows a vault in the window. `open` is the plugin open whose lock the switch shares.
type VaultSwitch interface {
	Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error
}

// Execute opens v on behalf of a plugin.
func (u OpenPluginVault) Execute(ctx context.Context, v domain.Vault) error {
	if u.Switch == nil {
		return errNoVaultSwitch
	}
	deps := u.life.deps
	if err := deps.Vault.Identity.Execute(v); err != nil {
		return err
	}
	ctx, open, end, err := u.life.rt.guard.tryBeginPluginOpen(ctx)
	if err != nil {
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, err)
	}
	defer end()
	if deps.Window.Pages.IsQuestionOpen() {
		return plugin.ErrUnsavedChanges
	}
	err = u.Switch.Execute(ctx, v, open)
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		if !deps.Window.Pages.IsQuestionOpen() {
			deps.Window.Pages.CancelFlush()
		}
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, context.Cause(ctx))
	case errors.Is(err, window.ErrAnswerPending):
		return plugin.ErrUnsavedChanges
	}
	return err
}
