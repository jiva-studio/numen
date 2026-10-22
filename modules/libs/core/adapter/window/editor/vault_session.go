package editor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// vaultSession is the half of the window that belongs to one vault: what runs behind
// it, what a request reaches them through, and what ends them.
//
// It is published as one, through API.showing, and every request reads it there.
type vaultSession struct {
	// refresh brings a note of this vault up to date through the opening it is
	// being followed by.
	refresh             vaults.Refresh
	recognitionWorker   *recognition.Worker
	transcriptionWorker *transcription.Worker

	// ctx is what every pass behind this vault runs under, and what work
	// nobody is waiting for is started under. cancel ends them, and wait waits
	// for them.
	ctx    context.Context
	cancel context.CancelFunc
	wait   func()
}

// errAnswerPending is a vault asked for while a page holds work a person is being
// asked about.
var errAnswerPending = errors.New("pending review confirmation")

// Show puts a vault in the window: another in place of the one it has, or the
// first where it is standing on nothing. The index and the embedder belong to
// the installation and stay; what belongs to the vault is taken down and built
// again.
//
// Nothing is taken away until the vault asked for reads as a vault and every
// page has written what only it holds. A page holding text a person has to
// answer for calls the swap off, and so does a vault that will not come up: the
// window stays on the one it had, and a window neither comes up in relies on
// nothing and says so.
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

	// A page that says nothing is waited for HandedOverIn and no longer.
	target, elapsed := context.WithTimeout(ctx, FlushTimeout)
	defer elapsed()

	if !flushPending(target, o.State.Window, &o.State.Writing) {
		return errAnswerPending
	}

	was := o.State.GetCurrentVault()
	o.stopVault()
	o.resetStatus()

	//nolint:contextcheck // the passes behind a vault run under o.under, for as long as the window stands
	err := o.startVault(v, false)
	if err != nil {
		if back := o.startVault(was, false); back != nil { //nolint:contextcheck // and so do the ones behind the vault it goes back to
			// The window is standing on nothing: it says so, and the door on
			// writes stays shut.
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
	// Everything a page is holding was read in a vault that is no longer in
	// front of it.
	o.State.ChangeEvents.publish(change{shouldReload: true})
	return err
}

// startVault puts a vault in the window and builds everything that belongs to it.
//
// The zero vault is a window standing on nothing: it shows no vault, and no
// pass runs behind it.
func (o *Installation) startVault(v domain.Vault, rebuild bool) error {
	o.State.setVault(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		o.State.Ready.Store(true)
		return nil
	}
	// Recorded before the vault is built, so the next window opens on it. A
	// list that could not be written is said and nothing more.
	if err := o.vaults.RecordOpen.Execute(v.ID); err != nil {
		fmt.Fprintf(o.out, "not recording %s as the vault opened: %v\n", v.Name, err)
	}
	on, err := o.buildSession(v, rebuild)
	if err != nil {
		return err
	}
	// Last, so a request that reads a run reads the one belonging to the vault
	// in front of it.
	o.State.setSession(on)
	return nil
}

// buildSession builds the half of the window that belongs to one vault: the scan
// and the watch behind it, the extraction of the documents it holds, and the
// batches left with a proofreader.
func (o *Installation) buildSession(v domain.Vault, rebuild bool) (*vaultSession, error) {
	recognized, err := o.vaults.List.Execute()
	if err != nil {
		return nil, err
	}

	watching, stop := context.WithCancel(o.ctx)
	recognising := o.made.OpenRecognitionWorker(watching, o.tasks, o.models)
	sources := o.made.GetSourceQueries()

	// What a recognition writes down is cut where every other cut happens. A
	// document being read and a vault being scanned are then never two passes
	// over the index at once.
	indexed := o.indexer.NewSession()
	recognising.Cut = func(_ context.Context, of domain.Vault, path string) error {
		indexed.RequestCut(of, path)
		return nil
	}

	// batches is taken up when the application opens.
	recognising.TakeUp(watching, sources, recognized...)

	// A recording says nothing until a model has listened to it, so the ones
	// this vault holds no transcript for are work whether or not anybody asks.
	// What it writes is cut where every other cut happens.
	transcribing := o.made.OpenTranscriptionWorker(watching, o.tasks, o.models)
	transcribing.Cut = recognising.Cut
	if o.made.IsTranscribingUnasked() {
		transcribing.Queue(watching, sources, heardEvery, v)
	}

	// here whether or not this installation listens to recordings on its own.
	// Every vault this installation holds is asked after.
	transcribing.TakeUp(watching, sources, recognized...)

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	refresh, ended := startIndexing(watching, v, o.made, o.State, rebuild, o.indexer, indexed, o.out)

	return &vaultSession{
		refresh:             refresh,
		recognitionWorker:   recognising,
		transcriptionWorker: transcribing,
		ctx:                 watching,
		cancel:              stop,
		wait:                ended,
	}, nil
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
	// A recognition and a transcription both write to the index, so they end before
	// anything reads what they wrote.
	on.recognitionWorker.Wait()
	on.transcriptionWorker.Wait()
}

// about indexing it, and the entries its passes left in the list of what is
// being done.
//
// What stopped this installation from embedding at all is put back. It stands
// for as long as the window is open.
func (o *Installation) resetStatus() {
	o.State.Ready.Store(false)
	o.State.Error.Store("")
	o.State.WatchError.Store("")

	for _, pass := range []string{walkingNotes, extractingBooks, makingVectors, wordsAlone} {
		o.State.finishTask(pass)
	}
	if o.embedderError != nil {
		o.State.reportTask(run.Task{ID: makingVectors, Title: "Indexing", Error: o.embedderError.Error()})
	}
}

// heardEvery is how often the vault is asked which recordings owe their text. A
// recording is dropped into a folder by hand, and one arriving a minute after a
// round is heard by the next.
const heardEvery = time.Minute
