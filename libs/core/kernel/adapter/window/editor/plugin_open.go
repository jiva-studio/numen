package editor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// errNoVaultSwitch is returned when the window build binds no vault switch.
var errNoVaultSwitch = errors.New("the window cannot switch vaults")

// OpenPluginVault rejects a vault whose folder is not the vault, then switches the window through
// State.OpenVault, which flushes the pages first.
// It runs under the lock of a close and a vault switch and never ends their
// flush request; a close, a quit, a user vault switch or the end of the call
// cancels it with ErrUnavailable. An open ASKING prompt returns ErrUnsavedChanges without a flush.
func (o *Installation) OpenPluginVault(ctx context.Context, v domain.Vault) error {
	if o.State.OpenVault == nil {
		return errNoVaultSwitch
	}
	if err := o.checkIdentity.Execute(v); err != nil {
		return err
	}
	ctx, open, end, err := o.shutdown.tryBeginPluginOpen(ctx)
	if err != nil {
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, err)
	}
	defer end()
	if o.State.Window.IsQuestionOpen() {
		return plugin.ErrUnsavedChanges
	}
	err = o.State.OpenVault(ctx, v, open)
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		if !o.State.Window.IsQuestionOpen() {
			o.State.Window.CancelFlush()
		}
		return fmt.Errorf("%s cannot be opened now: %w: %w", v.Name, domain.ErrUnavailable, context.Cause(ctx))
	case errors.Is(err, window.ErrAnswerPending):
		return plugin.ErrUnsavedChanges
	}
	return err
}
