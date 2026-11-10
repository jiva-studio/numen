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
	life *lifecycle
}

// Execute shows v. `open` is the plugin open whose lock the switch shares, nil for a user switch.
func (u ShowVault) Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	rt, deps := u.life.rt, u.life.deps
	if v.ID == rt.GetCurrentVault().ID {
		return nil
	}
	if err := deps.Identity.Execute(v); err != nil {
		return err
	}
	end, isInsidePluginOpen, err := rt.guard.tryBeginSwitch(open)
	if err != nil {
		return err
	}
	defer end()
	rt.lock.Lock()
	defer rt.lock.Unlock()

	// A page that reports nothing is waited for at most FlushTimeout.
	target, elapsed := context.WithTimeout(ctx, FlushTimeout)
	defer elapsed()

	if !flushPending(target, deps.Pages, deps.Writes) {
		return window.ErrAnswerPending
	}
	// A plugin's open canceled during the flush switches nothing.
	if isInsidePluginOpen && ctx.Err() != nil {
		deps.Writes.Open()
		return context.Cause(ctx)
	}

	u.life.stopVault()
	u.life.resetStatus()
	u.life.startVault(ctx, v, false)

	// Writes are taken again: there is a vault to write in.
	deps.Writes.Open()
	// The round the settling was is over, and what a page holds from here is
	// this vault's.
	deps.Pages.CancelFlush()
	// Every page's held content was read from the previous vault.
	deps.Changes.Publish(window.Change{ShouldReload: true})
	return nil
}

func flushPending(ctx context.Context, pages Pages, writes Writes) bool {
	if !pages.RequestFlush(ctx) {
		return false
	}
	<-writes.Seal()
	return true
}
