package windowsession

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FlushTimeout bounds how long a page may take to write its unsaved data when the vault closes. A page
// with a pending user prompt counts as responded; further waiting depends on the user.
const FlushTimeout = 3 * time.Second

// ShowVault puts a vault in the window, in place of the current one or as the first.
// The index and the embedder stay; everything belonging to the vault is rebuilt.
// Nothing is torn down until the vault validates and every page has written its
// pending state; a page awaiting confirmation aborts the swap.
type ShowVault struct {
	guard    *Guard
	identity IdentityCheck
	flush    *PageFlush
	changes  Changes
	session  vaultSession
}

// NewShowVault makes the use case over the guard and the session of the window, the check of the
// identity of a vault, the flush of the pages and the feed of the changes.
func NewShowVault(guard *Guard, session *Session, identity IdentityCheck, flush *PageFlush, changes Changes) ShowVault {
	return ShowVault{guard: guard, identity: identity, flush: flush, changes: changes, session: session}
}

// Execute shows v. `open` is the plugin open whose lock the switch shares, nil for a user switch.
func (u ShowVault) Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	if v.ID == u.session.getCurrentVault().ID {
		return nil
	}
	if err := u.identity.Execute(v); err != nil {
		return err
	}
	end, isInsidePluginOpen, err := u.guard.tryBeginSwitch(open)
	if err != nil {
		return err
	}
	defer end()
	u.guard.switching.Lock()
	defer u.guard.switching.Unlock()

	// A page that reports nothing is waited for at most FlushTimeout.
	target, elapsed := context.WithTimeout(ctx, FlushTimeout)
	defer elapsed()

	if !u.flush.Execute(target) {
		return window.ErrAnswerPending
	}
	// A plugin's open canceled during the flush switches nothing.
	if isInsidePluginOpen && ctx.Err() != nil {
		u.flush.ReopenWrites()
		return context.Cause(ctx)
	}

	u.session.stopVault()
	u.session.resetStatus()
	u.session.startVault(ctx, v, false)

	// Writes are taken again: there is a vault to write in.
	u.flush.ReopenWrites()
	// The round the settling was is over, and what a page holds from here is
	// this vault's.
	u.flush.EndRound()
	// Every page's held content was read from the previous vault.
	u.changes.Publish(window.Change{ShouldReload: true})
	return nil
}
