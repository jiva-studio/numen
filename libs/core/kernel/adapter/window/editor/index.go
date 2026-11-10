package editor

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/protocol"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/triggerpasses"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"

	textport "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors/embeddingcontract"
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

func (e indexEvents) ReportTask(pos vaultreading.Task) { e.state.reportTask(pos) }

func (e indexEvents) FinishTask(id string) { e.state.finishTask(id) }

func (e indexEvents) ReportError(err error) { e.state.Error.Store(err.Error()) }

func (e indexEvents) ReportNotes(v domain.Vault, notes int) {
	writeDiagnostic(e.state.reportTask, e.out, "%s: %d notes\n", v.Name, notes)
	e.state.Ready.Store(true)
}

func (e indexEvents) ReportTexts(v domain.Vault, res vaultreading.IndexResult) {
	writeDiagnostic(e.state.reportTask, e.out, "%s: %d file texts, %d chunks\n", v.Name, res.Extracted, res.Chunks)
}

// errNoProgress is the error of a window that cannot report vector progress.
var errNoProgress = errors.New("index progress is unknown")

func newIndexWorker(
	reading ReadVaultFunc, state *State, embedder embeddingport.Embedder, idleDuration time.Duration, out io.Writer,
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:  indexing.OpenReading(reading),
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
			if state.EmitEvent != nil {
				state.EmitEvent(protocol.GetVectorsReadyEvent(string(v.ID)))
			}
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

func startIndexing(
	ctx context.Context, source SessionSource, triggers triggerpasses.Source, state *State, job indexJob,
) (vaults.Refresh, func()) {
	v, rebuild, worker, session, out := job.vault, job.rebuild, job.worker, job.session, job.out
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	state.reportTask(vaultreading.Task{ID: walkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	triggerPasses := triggerpasses.NewPasses()
	checkTexts := func(paths []string) {
		if err := source.ReportTextFilesChanged(ctx, v, paths); err != nil {
			writeDiagnostic(state.reportTask, out, "check the text versions of %s: %v\n", v.Name, err)
		}
	}
	read, run, refresh, unwatched := source.StartVault(
		ctx, v, rebuild, newChangeReporter(state, worker, triggerPasses, checkTexts), newErrorReporter(state))
	if unwatched != nil {
		writeDiagnostic(state.reportTask, out, "not watching %s: %v\n", v.Name, unwatched)
		state.WatchError.Store(unwatched.Error())
	}
	running.Go(func() {
		watchVault(ctx, state, run, unwatched)
	})

	running.Go(func() {
		session.Run(ctx, v, rebuild, read)
	})

	if runTriggers := triggers.Start(ctx, v, triggerPasses, func(err error) {
		writeDiagnostic(state.reportTask, out, "run the triggers of %s: %v\n", v.Name, err)
	}); runTriggers != nil {
		running.Go(runTriggers)
	}

	return refresh, running.Wait
}

func newErrorReporter(state *State) window.ErrorHandler {
	return func(err error) {
		if err == nil {
			state.Error.Store("")
			return
		}
		state.Error.Store(err.Error())
	}
}

func newChangeReporter(
	state *State, worker *indexing.Worker, triggers *triggerpasses.Passes, checkTexts func(paths []string),
) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client shows every file in the vault, so an asset is reported as a note is.
		changePaths := slices.Concat(paths, assets)
		if reload {
			triggers.RequestAll()
		} else {
			triggers.RequestPaths(changePaths)
		}
		state.ChangeEvents.publish(change{paths: changePaths, shouldReload: reload})
		checkTexts(changePaths)
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

// ReadVaultFunc reads a whole vault into the index.
type ReadVaultFunc func(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vaultreading.ReadWholeVault, error)

// StartVaultFunc starts the walk and the watch of one vault.
type StartVaultFunc func(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
)

// SessionSource is what the passes behind one vault run over, made by the composition root from the
// installation's adapters. ReportTextFilesChanged reports to the text version listener each changed
// file of the vault whose live text version changed. GetRefresh brings named notes up to date.
type SessionSource struct {
	StartVault             StartVaultFunc
	SetTextVersionListener func(listener textport.VersionListener)
	ReportTextFilesChanged func(ctx context.Context, v domain.Vault, paths []string) error
	GetRefresh             func() vaults.Refresh
}
