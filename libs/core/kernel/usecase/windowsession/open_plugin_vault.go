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

// OpenPluginVault rejects a vault whose folder is not the vault, then switches the window through the
// switch it is given, which flushes the pages first.
// It runs under the lock of a close and a vault switch and never ends their
// flush request; a close, a quit, a user vault switch or the end of the call
// cancels it with ErrUnavailable. An open ASKING prompt returns ErrUnsavedChanges without a flush.
type OpenPluginVault struct {
	guard    *Guard
	identity IdentityCheck
	pages    Pages
}

// NewOpenPluginVault makes the use case over the guard of the window, the check of the identity of a
// vault and the pages of the window.
func NewOpenPluginVault(guard *Guard, identity IdentityCheck, pages Pages) OpenPluginVault {
	return OpenPluginVault{guard: guard, identity: identity, pages: pages}
}

// Execute opens v on behalf of a plugin. show switches the window and is nil for a window that cannot.
func (u OpenPluginVault) Execute(
	ctx context.Context, v domain.Vault, show func(context.Context, domain.Vault, *PluginOpen) error,
) error {
	if show == nil {
		return errNoVaultSwitch
	}
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
	err = show(ctx, v, open)
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
