package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	embedmodel "github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// quietPeriod is how long the vault has to have been still before the notes
// written into it are embedded. It is longer than the bound in the editor
// window's note/tab.ts, which writes an unfinished edit every five seconds
// while a person goes on typing.
const quietPeriod = indexing.QuietPeriod

// indexEvents shows what the indexing worker reports in the window.
type indexEvents struct {
	state *State
	out   io.Writer
}

func (e indexEvents) ReportTask(at run.Task) { e.state.reportTask(at) }

func (e indexEvents) FinishTask(id string) { e.state.finishTask(id) }

func (e indexEvents) ReportError(err error) { e.state.Error.Store(err.Error()) }

func (e indexEvents) ReportNotes(v domain.Vault, notes int) {
	fmt.Fprintf(e.out, "%s: %d notes\n", v.Name, notes)
	e.state.Ready.Store(true)
}

func (e indexEvents) ReportBooks(v domain.Vault, res source.ExtractResult) {
	fmt.Fprintf(e.out, "%s: %d books, %d chunks\n", v.Name, res.Extracted, res.Chunks)
}

// errNoProgress is a window with nothing that says how far the vectors have got.
var errNoProgress = errors.New("nothing says how far the index has got")

// newIndexWorker is the indexing of the vaults this window shows, over the
// embedder the installation holds. still is how long the vault has to have been
// quiet before a note that was written is embedded.
func newIndexWorker(
	made Assembly, state *State, embedder port.Embedder, still time.Duration, out io.Writer,
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:  made.ReadWholeVault,
		Embedder: embedder,
		Wait:     getModelWait(embedder),
		Owed: func(ctx context.Context, v domain.Vault) (held, embedded int64, err error) {
			if state.Indexing.Progress == nil {
				return 0, 0, errNoProgress
			}
			return state.Indexing.Progress.GetProgress(ctx, v.ID, text(&state.Indexing.Recipe))
		},
		Events: indexEvents{state: state, out: out},
		Quiet:  still,
		After:  time.After,
	})
}

// getModelWait is what blocks until the model behind an embedder has turned up,
// and nothing for an embedder whose model is always there.
func getModelWait(embedder port.Embedder) indexing.WaitForModel {
	if waiting, ok := embedder.(embedmodel.WaitingEmbedder); ok {
		return waiting.Wait
	}
	return nil
}

// startIndexing starts the watch and the first scan together, and answers with
// what waits for both to stop.
//
// The watch is acted on alongside the scan: a change reaches the window through
// it and through nothing else.
//
// A vault that cannot be watched is still a vault: the application keeps
// working, and says that changes will not appear by themselves.
func startIndexing(
	ctx context.Context,
	v domain.Vault,
	made Assembly,
	state *State,
	rebuild bool,
	worker *indexing.Worker,
	session *indexing.Session,
	out io.Writer,
) (vaults.Refresh, func()) {
	// Said before the goroutine starts, so a window that opens on a fresh vault
	// is shown the walk from its first moment.
	state.reportTask(run.Task{ID: walkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	read, run, refresh, unwatched := made.StartVault(
		ctx, v, rebuild, newChangeReporter(state, worker), newErrorReporter(state))
	if unwatched != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", v.Name, unwatched)
		state.WatchError.Store(unwatched.Error())
	}
	running.Add(1)
	go func() {
		defer running.Done()
		watchVault(ctx, state, run, unwatched)
	}()

	running.Add(1)
	go func() {
		defer running.Done()
		session.Run(ctx, v, rebuild, read)
	}()

	return refresh, running.Wait
}

// newErrorReporter tells the window what went wrong along the way, and tells it
// that nothing does once a later attempt succeeds.
func newErrorReporter(state *State) port.ErrorHandler {
	return func(err error) {
		if err == nil {
			state.Error.Store("")
			return
		}
		state.Error.Store(err.Error())
	}
}

// newChangeReporter tells the window what changed, and wakes the passes that owe
// work for it.
func newChangeReporter(state *State, worker *indexing.Worker) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client draws every file the vault holds, so an asset is named to it
		// the way a note is.
		state.ChangeEvents.publish(change{paths: slices.Concat(paths, assets), shouldReload: reload})
		if reload || len(assets) > 0 {
			// A book dropped into an open vault is read without anybody asking.
			worker.WakeSources()
		}
		if len(paths) > 0 {
			// A note written is a note cut again, and its chunks owe their
			// vectors. Which ones is in the index.
			worker.WakeNotes()
		}
	}
}

// watchVault runs the watch until it stops.
func watchVault(ctx context.Context, state *State, run func(context.Context), unwatched error) {
	run(ctx)
	// Nothing reaches the window once the watch stops, so from here on the
	// vault is one that is not being followed.
	if unwatched == nil && ctx.Err() == nil {
		state.WatchError.Store("the watch stopped")
	}
}
