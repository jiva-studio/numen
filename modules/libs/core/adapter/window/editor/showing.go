package editor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// passes is the half of the window that belongs to one vault: what runs behind
// it, what a request reaches them through, and what ends them.
//
// It is published as one, through API.on, and every request reads it there.
type passes struct {
	opening      *container.VaultOpener
	recognising  *source.RecognitionWorker
	transcribing *source.TranscriptionWorker

	// recognises reads a scanned document, transcribes hears a recording, and
	// proofreads puts a transcript right, each for whoever asks.
	recognises  Runner
	transcribes Runner
	proofreads  Proofreader
	// cut asks for a source to be cut again from what its text now says, and
	// forgets takes a recording out of what the queue has had an answer about.
	cut     func(context.Context, domain.Vault, string) error
	forgets func(domain.Vault, string)

	// under is what every pass behind this vault runs under, and what work
	// nobody is waiting for is started under. stop ends them, and ended waits
	// for them.
	under context.Context
	stop  context.CancelFunc
	ended func()
}

// errAsking is a vault asked for while a page holds work a person is being
// asked about.
var errAsking = errors.New("a page is holding work a person has to answer for")

// Show puts a vault in the window: another in place of the one it has, or the
// first where it is standing on nothing. The index and the embedder belong to
// the installation and stay; what belongs to the vault is taken down and built
// again.
//
// Nothing is taken away until the vault asked for reads as a vault and every
// page has written what only it holds. A page holding text a person has to
// answer for calls the swap off, and the window stays on the vault it had.
//
// A vault that will not come up leaves the window on the one it was showing. A
// window neither of them comes up in stands on nothing and says so.
func (o *Installation) Show(ctx context.Context, v domain.Vault) error {
	if v.ID == o.API.Showing().ID {
		return nil
	}
	if err := readable(o.cfg, v); err != nil {
		return err
	}
	if err := o.shutting.alone(); err != nil {
		return err
	}
	defer o.shutting.free()

	// A page that says nothing is waited for HandedOverIn and no longer.
	held, spent := context.WithTimeout(ctx, HandedOverIn)
	defer spent()

	if !settling(held, o.API.Window, &o.API.Writing) {
		return errAsking
	}

	was := o.API.Showing()
	o.leave()
	o.forget()

	err := o.arrive(v, false)
	if err != nil {
		if back := o.arrive(was, false); back != nil {
			// The window is standing on nothing: it says so, and the door on
			// writes stays shut.
			o.API.show(domain.Vault{})
			o.API.Failed.Store(back.Error())
			return errors.Join(err, back)
		}
	}
	// Writes are taken again: there is a vault to write in.
	o.API.Writing.open()
	// The round the settling was is over, and what a page holds from here is
	// this vault's.
	o.API.Window.Over()
	// Everything a page is holding was read in a vault that is no longer in
	// front of it.
	o.API.Listeners.tell(change{reload: true})
	return err
}

// arrive puts a vault in the window and builds everything that belongs to it.
//
// The zero vault is a window standing on nothing: it shows no vault, and no
// pass runs behind it.
func (o *Installation) arrive(v domain.Vault, rebuild bool) error {
	o.API.show(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		o.API.Ready.Store(true)
		return nil
	}
	// Recorded before the vault is built, so the next window opens on it. A
	// list that could not be written is said and nothing more.
	if err := o.registry.Opened(v.ID); err != nil {
		fmt.Fprintf(o.out, "not recording %s as the vault opened: %v\n", v.Name, err)
	}
	on, err := o.begins(v, rebuild)
	if err != nil {
		return err
	}
	// Last, so a request that reads a run reads the one belonging to the vault
	// in front of it.
	o.API.runs(on)
	return nil
}

// begins builds the half of the window that belongs to one vault: the scan and
// the watch behind it, the reading of the documents it holds, and the batches
// left with a proofreader.
func (o *Installation) begins(v domain.Vault, rebuild bool) (*passes, error) {
	known, err := vaults.NewList(o.registry).Execute()
	if err != nil {
		return nil, err
	}

	watching, stop := context.WithCancel(o.under)
	recognising := o.cfg.Recognising(watching, o.Index.Sources(), o.tasks)

	// What a recognition writes down is cut where every other cut happens. A
	// document being read and a vault being scanned are then never two passes
	// over the index at once.
	owed := &pending{}
	recognising.Cut = func(_ context.Context, of domain.Vault, path string) error {
		owed.put(of, path)
		raise(o.wake.read)
		return nil
	}

	// A batch left with a proofreader outlives the run that left it, so one
	// left before the application closed is collected when it opens. Every
	// vault this installation holds is asked after.
	recognising.Collecting(watching, o.Index.SourcesKnown(), collectedEvery, known...)

	// A proofreading stands at the page it reached, so one that ended among the
	// batches is taken up when the application opens.
	recognising.TakingUp(watching, o.Index.SourcesKnown(), known...)

	// A recording says nothing until a model has listened to it, so the ones
	// this vault holds no transcript for are work whether or not anybody asks.
	// What it writes is cut where every other cut happens.
	transcribing := o.cfg.Transcribing(watching, o.Index.Sources(), o.tasks)
	transcribing.Cut = recognising.Cut
	if o.cfg.Transcribes {
		transcribing.Queue(watching, o.Index.SourcesKnown(), heardEvery, v)
	}

	// A transcript's proofreading stands at the line it reached, and is taken up
	// here whether or not this installation listens to recordings on its own.
	// Every vault this installation holds is asked after.
	transcribing.TakingUp(watching, o.Index.SourcesKnown(), known...)

	// Reading every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	cfg := o.cfg
	cfg.RebuildIndex = rebuild

	// Opening a vault is the same act in both windows, so it is one thing in the
	// container. What this window says about it while it runs is below.
	opening := cfg.VaultOpener(o.Index)
	opening.Rebuild = rebuild

	ended := begin(watching, v, cfg, o.Index, o.API, opening,
		cfg.VaultReaders(), o.Embedder, o.wake, owed, o.out)

	return &passes{
		opening:      opening,
		recognising:  recognising,
		transcribing: transcribing,
		// The window asks for a scan to be read through the same job an agent
		// asks through.
		recognises:  recognising,
		transcribes: transcribing,
		// A transcript is put right by the same proofreading that runs on its
		// own, so a person asking for one is shown the run everything else is
		// shown in.
		proofreads: transcribing,
		// A transcript the window put right is cut where every other cut
		// happens.
		cut: recognising.Cut,
		// A recording whose answer was dropped is one the queue has had no
		// answer about.
		forgets: transcribing.Forget,
		under:   watching,
		stop:    stop,
		ended:   ended,
	}, nil
}

// leave takes down the half of the window that belongs to the vault it is
// showing.
func (o *Installation) leave() {
	on := o.API.showing.Swap(nil)
	if on == nil {
		return
	}
	on.stop()
	on.ended()
	// A reading and a transcription both write to the index, so they end before
	// anything reads what they wrote.
	on.recognising.Wait()
	on.transcribing.Wait()
	// The documents held open go with the vault, and each gives back the worker
	// it was holding.
	if o.API.Viewer != nil {
		o.API.Viewer.empty()
	}
}

// forget is the vault that went leaving nothing of itself behind: what was said
// about reading it, and the entries its passes left in the list of what is
// being done.
//
// What stopped this installation from embedding at all is put back. It stands
// for as long as the window is open.
func (o *Installation) forget() {
	o.API.Ready.Store(false)
	o.API.Failed.Store("")
	o.API.Unwatched.Store("")

	for _, pass := range []string{walkingNotes, readingBooks, makingVectors, wordsAlone} {
		o.API.finished(pass)
	}
	if o.why != nil {
		o.API.say(task.Task{ID: makingVectors, Doing: "Indexing", Failed: o.why.Error()})
	}
}

// chosen is the vault this window opens: the one a person named, else the one
// shown last, else the first this installation holds. An installation holding
// none answers with no vault at all.
//
// A vault named and not on the list is refused, and the window does not open.
func chosen(registry port.VaultRegistry, asked string) (domain.Vault, error) {
	if asked != "" {
		return vaults.NewFind(registry).Execute(asked)
	}
	last, found, err := registry.Last()
	if err != nil {
		return domain.Vault{}, err
	}
	if found {
		return last, nil
	}
	held, err := vaults.NewList(registry).Execute()
	if err != nil {
		return domain.Vault{}, err
	}
	if len(held) > 0 {
		return held[0], nil
	}
	return domain.Vault{}, nil
}

// readable is the vault being one this window can show: the folder reads as a
// vault, and it carries the identity the list has for it.
func readable(cfg container.Config, v domain.Vault) error {
	identity := cfg.VaultIdentity()
	if err := identity.Readable(v.Path); err != nil {
		return fmt.Errorf("%w: %w", vaults.ErrUnreadable, err)
	}
	carried, found, err := identity.Of(v.Path)
	if err != nil {
		return err
	}
	if !found || carried != v.ID {
		return fmt.Errorf("%w: %s is no longer the vault %s", vaults.ErrUnreadable, v.Path, v.Name)
	}
	return nil
}

// collectedEvery is how often the batches left with a proofreader are asked
// after. A batch is answered in hours.
const collectedEvery = 5 * time.Minute

// heardEvery is how often the vault is asked which recordings owe their text. A
// recording is dropped into a folder by hand, and one arriving a minute after a
// round is heard by the next.
const heardEvery = time.Minute
