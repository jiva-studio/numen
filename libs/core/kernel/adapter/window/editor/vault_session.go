package editor

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// vaultSession is the part of the window that belongs to one vault: the passes
// running for it, the path requests reach them through, and what ends them.
//
// It is published as one unit through API.showing, and every request reads it there.
type vaultSession struct {
	refresh vaults.Refresh

	cancel context.CancelFunc
	wait   func()
}

// errAnswerPending is returned when a vault is requested while a page holds work
// awaiting the user's confirmation.
var errAnswerPending = errors.New("pending review confirmation")

// Show puts a vault in the window, in place of the current one or as the first.
// The index and the embedder stay; everything belonging to the vault is rebuilt.
// Nothing is torn down until the vault validates and every page has written its
// pending state; a page awaiting confirmation or a vault that fails to start
// aborts the swap. `open` is the plugin open whose lock the switch shares, nil for a user switch.
func (o *Installation) Show(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	if v.ID == o.State.GetCurrentVault().ID {
		return nil
	}
	if err := o.checkIdentity.Execute(v); err != nil {
		return err
	}
	end, isInsidePluginOpen, err := o.shutdown.tryBeginSwitch(open)
	if err != nil {
		return err
	}
	defer end()
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()

	// A page that reports nothing is waited for at most FlushTimeout.
	target, elapsed := context.WithTimeout(ctx, FlushTimeout)
	defer elapsed()

	if !flushPending(target, o.State.Window, &o.State.Writing) {
		return errAnswerPending
	}
	// A plugin's open canceled during the flush switches nothing.
	if isInsidePluginOpen && ctx.Err() != nil {
		o.State.Writing.open()
		return context.Cause(ctx)
	}

	was := o.State.GetCurrentVault()
	o.stopVault()
	o.resetStatus()

	err = o.startVault(ctx, v, false)
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

func (o *Installation) rebuildSession() {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()
	current := o.State.GetCurrentVault()
	if current.ID == "" || o.shutdown.isClosed() {
		return
	}
	o.stopVault()
	o.resetStatus()
	if err := o.startVault(context.Background(), current, false); err != nil {
		o.State.Error.Store(err.Error())
	}
}

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
	o.sources.Texts.SetTextVersionListener(newVersionChangeReporter(o.State, indexed))

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	refresh, ended := startIndexing(watching, o.sources, o.State, indexJob{vault: v, rebuild: rebuild, worker: o.indexer, session: indexed, out: o.out})

	return &vaultSession{
		refresh: refresh,
		cancel:  stop,
		wait:    ended,
	}
}

func (o *Installation) stopVault() {
	on := o.State.session.Swap(nil)
	if on == nil {
		return
	}
	on.cancel()
	on.wait()
}

func (o *Installation) resetStatus() {
	o.State.Ready.Store(false)
	o.State.Error.Store("")
	o.State.WatchError.Store("")

	for _, pass := range []string{walkingNotes, indexingTexts, makingVectors, wordsAlone} {
		o.State.finishTask(pass)
	}
	if o.embedderError != nil {
		o.State.reportTask(vaultreading.Task{ID: makingVectors, Title: "Indexing", Error: o.embedderError.Error()})
	}
}

// versionChangeReporter asks the index session to cut a file whose text versions changed, and
// reports the file to the window.
type versionChangeReporter struct {
	state *State
	index *indexing.Session
}

func newVersionChangeReporter(state *State, index *indexing.Session) versionChangeReporter {
	return versionChangeReporter{state: state, index: index}
}

func (r versionChangeReporter) ReportVersionsChanged(v domain.Vault, path string) {
	r.index.ReportVersionsChanged(v, path)
	r.state.ChangeEvents.publish(change{versionPaths: []string{path}})
}
