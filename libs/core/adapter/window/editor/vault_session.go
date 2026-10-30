package editor

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// vaultSession is the part of the window that belongs to one vault: the passes
// running for it, the path requests reach them through, and what ends them.
//
// It is published as one unit through API.showing, and every request reads it there.
type vaultSession struct {
	// refresh brings a note of this vault up to date through the opening it is
	// being followed by.
	refresh vaults.Refresh

	// cancel ends every pass behind this vault, and wait waits for them.
	cancel context.CancelFunc
	wait   func()
}

// errAnswerPending is returned when a vault is requested while a page holds work
// awaiting the user's confirmation.
var errAnswerPending = errors.New("pending review confirmation")

// Show puts a vault in the window: another in place of the current one, or the
// first when none is shown. The index and the embedder belong to the installation
// and stay; everything belonging to the vault is torn down and rebuilt.
//
// Nothing is torn down until the requested vault validates as a vault and every
// page has written its pending state. A page holding text awaiting user
// confirmation aborts the swap, as does a vault that fails to start: the window
// stays on its current vault, and a window with no vault to fall back on reports
// the error.
func (o *Installation) Show(ctx context.Context, v domain.Vault) error {
	if v.ID == o.State.GetCurrentVault().ID {
		return nil
	}
	if err := o.checkIdentity.Execute(v); err != nil {
		return err
	}
	if err := o.shutdown.tryBegin(); err != nil {
		return err
	}
	defer o.shutdown.endKeepOpen()

	// A page that reports nothing is waited for at most FlushTimeout.
	target, elapsed := context.WithTimeout(ctx, FlushTimeout)
	defer elapsed()

	if !flushPending(target, o.State.Window, &o.State.Writing) {
		return errAnswerPending
	}

	was := o.State.GetCurrentVault()
	o.stopVault()
	o.resetStatus()

	err := o.startVault(ctx, v, false)
	if err != nil {
		if back := o.startVault(ctx, was, false); back != nil {
			// The window shows no vault and writes stay disabled.
			o.State.setVault(domain.Vault{})
			o.State.Error.Store(back.Error())
			return errors.Join(err, back)
		}
	}
	// Writes are taken again: there is a vault to write in.
	o.State.Writing.open()
	// The round the settling was is over, and what a page holds from here is
	// this vault's.
	o.State.Window.CancelFlush()
	// Every page's held content was read from the previous vault.
	o.State.ChangeEvents.publish(change{shouldReload: true})
	return err
}

// startVault puts a vault in the window and builds everything that belongs to it.
//
// The zero vault is a window standing on nothing: it shows no vault, and no
// pass runs behind it.
func (o *Installation) startVault(ctx context.Context, v domain.Vault, rebuild bool) error {
	o.State.setVault(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		o.State.Ready.Store(true)
		return nil
	}
	// Recorded before the vault is built, so the next window opens on it. A failure
	// to write the list is reported and does not stop the vault from opening.
	if err := o.vaults.RecordOpen.Execute(v.ID); err != nil {
		writeDiagnostic(o.State.reportTask, o.out, "not recording %s as the vault opened: %v\n", v.Name, err)
	}
	on := o.buildSession(ctx, v, rebuild)
	// Last, so a request that reads a run reads the one belonging to the vault
	// in front of it.
	o.State.setSession(on)
	return nil
}

// buildSession builds the half of the window that belongs to one vault: the scan
// and the watch behind it.
func (o *Installation) buildSession(ctx context.Context, v domain.Vault, rebuild bool) *vaultSession {
	watching, stop := context.WithCancel(context.WithoutCancel(ctx))
	if o.closing != nil {
		go func() {
			select {
			case <-o.closing:
				stop()
			case <-watching.Done():
			}
		}()
	}

	// What a plugin writes as a text version is cut where every other cut happens.
	// A version being made and a vault being scanned are then never two passes
	// over the index at once.
	indexed := o.indexer.NewSession()
	o.made.SetTextVersionListener(indexed)

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	refresh, ended := startIndexing(watching, v, o.made, o.State, rebuild, o.indexer, indexed, o.out)

	return &vaultSession{
		refresh: refresh,
		cancel:  stop,
		wait:    ended,
	}
}

// stopVault takes down the half of the window that belongs to the vault it is
// showing.
func (o *Installation) stopVault() {
	on := o.State.session.Swap(nil)
	if on == nil {
		return
	}
	on.cancel()
	on.wait()
}

// about indexing it, and the entries its passes left in the list of running work.
//
// An error that prevented embedding for the whole installation is restored, and
// persists while the window is open.
func (o *Installation) resetStatus() {
	o.State.Ready.Store(false)
	o.State.Error.Store("")
	o.State.WatchError.Store("")

	for _, pass := range []string{walkingNotes, indexingTexts, makingVectors, wordsAlone} {
		o.State.finishTask(pass)
	}
	if o.embedderError != nil {
		o.State.reportTask(run.Task{ID: makingVectors, Title: "Indexing", Error: o.embedderError.Error()})
	}
}
