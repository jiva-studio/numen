package editor

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/protocol"

	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"

	windowport "github.com/jiva-studio/numen/libs/core/usecase/vault/windowcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

// quietPeriod is how long the vault must be unchanged before the notes written
// into it are embedded. It is longer than the bound in the editor window's
// note/tab.ts, which writes an unfinished edit every five seconds while the user
// is typing.
const quietPeriod = indexing.QuietPeriod

// indexEvents shows what the indexing worker reports in the window.
type indexEvents struct {
	state *State
	out   io.Writer
}

func (e indexEvents) ReportTask(pos run.Task) { e.state.reportTask(pos) }

func (e indexEvents) FinishTask(id string) { e.state.finishTask(id) }

func (e indexEvents) ReportError(err error) { e.state.Error.Store(err.Error()) }

func (e indexEvents) ReportNotes(v domain.Vault, notes int) {
	writeDiagnostic(e.state.reportTask, e.out, "%s: %d notes\n", v.Name, notes)
	e.state.Ready.Store(true)
}

func (e indexEvents) ReportTexts(v domain.Vault, res filetext.IndexResult) {
	writeDiagnostic(e.state.reportTask, e.out, "%s: %d file texts, %d chunks\n", v.Name, res.Extracted, res.Chunks)
}

// newFloorErrorReporter reports a store of floors that cannot be read as a task.
func newFloorErrorReporter(state *State) func(error) {
	return func(err error) {
		state.reportTask(run.Task{ID: readingTheFloor, Title: "Read search floor", Error: err.Error()})
	}
}

// errNoProgress is the error of a window that cannot report vector progress.
var errNoProgress = errors.New("index progress is unknown")

// newIndexWorker is the indexing of the vaults this window shows, over the
// embedder the installation holds. still is how long the vault has to have been
// quiet before a note that was written is embedded.
func newIndexWorker(
	made Assembly, state *State, embedder embeddingport.Embedder, idleDuration time.Duration, out io.Writer,
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:  made.ReadWholeVault,
		Embedder: embedder,
		VectorCounts: func(ctx context.Context, v domain.Vault) (activeItem, embedded int64, err error) {
			if state.Indexing.Progress == nil {
				return 0, 0, errNoProgress
			}
			if embedder != nil {
				name, recipe := vectors.DescribeModel(embedder)
				state.Indexing.Model.Store(name)
				state.Indexing.Recipe.Store(recipe)
			}
			return indexing.GetProgress(ctx, state.Indexing.Progress, v.ID, text(&state.Indexing.Recipe))
		},
		VectorsReady: func(_ context.Context, v domain.Vault) {
			made.EmitEvent(protocol.GetVectorsReadyEvent(string(v.ID)))
		},
		Events: indexEvents{state: state, out: out},
		Quiet:  idleDuration,
		After:  time.After,
	})
}

// indexJob is the vault to index and the parts that index it.
type indexJob struct {
	vault   domain.Vault
	rebuild bool
	worker  *indexing.Worker
	session *indexing.Session
	out     io.Writer
}

// startIndexing starts the watch and the first scan together, and returns a
// function that waits for both to stop.
//
// The watch runs alongside the scan: changes reach the window only through it.
//
// A vault that cannot be watched stays usable: the application keeps working
// and reports that changes will not appear automatically.
func startIndexing(ctx context.Context, made Assembly, state *State, job indexJob) (vaults.Refresh, func()) {
	v, rebuild, worker, session, out := job.vault, job.rebuild, job.worker, job.session, job.out
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	state.reportTask(run.Task{ID: walkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	read, run, refresh, unwatched := made.StartVault(
		ctx, v, rebuild, newChangeReporter(state, worker), newErrorReporter(state))
	if unwatched != nil {
		writeDiagnostic(state.reportTask, out, "not watching %s: %v\n", v.Name, unwatched)
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

// newErrorReporter reports errors to the window, and clears the reported error
// when a later attempt succeeds.
func newErrorReporter(state *State) windowport.ErrorHandler {
	return func(err error) {
		if err == nil {
			state.Error.Store("")
			return
		}
		state.Error.Store(err.Error())
	}
}

// newChangeReporter reports changes to the window and wakes the passes that
// process them.
func newChangeReporter(state *State, worker *indexing.Worker) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client shows every file in the vault, so an asset is reported as a note is.
		changePaths := slices.Concat(paths, assets)
		state.ChangeEvents.publish(change{paths: changePaths, shouldReload: reload})
		worker.ClearSkips(changePaths)
		if reload || len(assets) > 0 {
			// A book added to an open vault is read without a request.
			worker.WakeSources()
		}
		if len(paths) > 0 {
			// A written note is chunked again, and its chunks need new vectors. The index
			// determines which.
			worker.WakeNotes()
		}
	}
}

// watchVault runs the watch until it stops.
func watchVault(ctx context.Context, state *State, run func(context.Context), unwatched error) {
	run(ctx)
	// No changes reach the window once the watch stops, so the vault is not
	// followed from this point.
	if unwatched == nil && ctx.Err() == nil {
		state.WatchError.Store("the watch stopped")
	}
}

// modelWatcher is an embedder that reports a change of its model.
type modelWatcher interface {
	OnModelChange(fn func())
}

// followModel records the model of the embedder again and wakes the passes over the file
// texts each time the embedder reports that its model changed. A write of any other
// setting does neither.
//
// recut, when given, runs each time the chunk size the model reports differs from the last one seen, so the vault
// session is built again and the notes are cut at the new size.
func followModel(embedder embeddingport.Embedder, state *State, worker *indexing.Worker, recut func()) {
	watcher, ok := embedder.(modelWatcher)
	if !ok {
		return
	}
	chunkTokens := embedder.GetModel().ChunkTokens
	watcher.OnModelChange(func() {
		name, recipe := vectors.DescribeModel(embedder)
		state.Indexing.Model.Store(name)
		state.Indexing.Recipe.Store(recipe)
		if worker != nil {
			worker.WakeSources()
		}
		if tokens := embedder.GetModel().ChunkTokens; tokens != chunkTokens {
			chunkTokens = tokens
			if recut != nil {
				recut()
			}
		}
	})
}
