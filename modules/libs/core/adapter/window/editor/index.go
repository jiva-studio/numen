package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// quietPeriod is how long the vault has to have been still before the notes written
// into it are embedded. It is longer than the bound in the editor window's
// note/tab.ts, which writes an unfinished edit every five seconds while a
// person goes on typing.
const quietPeriod = 8 * time.Second

// indexTriggers are the two ways work reaches the indexing behind the window once the
// first pass is over: a book, which is found and cut before anything is
// embedded, and a note, which arrives already cut and owes only its vectors.
type indexTriggers struct {
	sources chan struct{}
	notes   chan struct{}
	// read is one source with more text than its chunks account for, which is
	// what a recognition leaves behind every batch of pages.
	read chan struct{}
	// still is how long the vault has to have been quiet before a note that was
	// written is embedded.
	still time.Duration
}

func newIndexTriggers(still time.Duration) indexTriggers {
	return indexTriggers{
		sources: make(chan struct{}, 1),
		notes:   make(chan struct{}, 1),
		read:    make(chan struct{}, 1),
		still:   still,
	}
}

// fire leaves one trigger waiting. What owes work is asked of the index, so
// several changes at once are one pass.
func fire(nudge chan struct{}) {
	select {
	case nudge <- struct{}{}:
	default:
	}
}

// pendingSources is the sources a recognition has written more of than their chunks
// account for.
//
// A source stands here once, however many batches it wrote, and several stand
// at a time. It belongs to the vault that was being read and goes with it.
type pendingSources struct {
	mu    sync.Mutex
	paths map[string]domain.Vault
}

func (p *pendingSources) add(v domain.Vault, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paths == nil {
		p.paths = map[string]domain.Vault{}
	}
	p.paths[path] = v
}

// drain is everything waiting, and leaves nothing behind.
func (p *pendingSources) drain() map[string]domain.Vault {
	p.mu.Lock()
	defer p.mu.Unlock()
	held := p.paths
	p.paths = nil
	return held
}

// startIndexing starts the watch and the first scan together, and answers with what
// waits for both to stop.
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
	api *API,
	rebuild bool,
	embedder port.Embedder,
	wake indexTriggers,
	owed *pendingSources,
	out io.Writer,
) (vaults.Refresh, func()) {
	// Said before the goroutine starts, so a window that opens on a fresh vault
	// is shown the walk from its first moment.
	api.reportTask(task.Task{ID: walkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	read, run, refresh, unwatched := made.StartVault(
		ctx, v, rebuild, newChangeReporter(api, wake), newErrorReporter(api))
	if unwatched != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", v.Name, unwatched)
		api.WatchError.Store(unwatched.Error())
	}
	running.Add(1)
	go func() {
		defer running.Done()
		watchVault(ctx, api, run, unwatched)
	}()

	pass := indexer{
		made: made, api: api, v: v, embedder: embedder, wake: wake, owed: owed, out: out,
		// Indexing every file again is what this launch was asked for, and one
		// pass makes it. Every pass after it indexes what changed.
		owes: rebuild,
	}
	running.Add(1)
	go func() {
		defer running.Done()
		pass.run(ctx, read)
	}()

	return refresh, running.Wait
}

// newErrorReporter tells the window what went wrong along the way, and tells it
// that nothing does once a later attempt succeeds.
func newErrorReporter(api *API) port.ErrorHandler {
	return func(err error) {
		if err == nil {
			api.Error.Store("")
			return
		}
		api.Error.Store(err.Error())
	}
}

// newChangeReporter tells the window what changed, and wakes the passes that owe
// work for it.
func newChangeReporter(api *API, wake indexTriggers) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client draws every file the vault holds, so an asset is named to it
		// the way a note is.
		api.ChangeEvents.publish(change{paths: slices.Concat(paths, assets), shouldReload: reload})
		if reload || len(assets) > 0 {
			// A book dropped into an open vault is read without anybody asking.
			fire(wake.sources)
		}
		if len(paths) > 0 {
			// A note written is a note cut again, and its chunks owe their
			// vectors. Which ones is not carried: the debt is in the index.
			fire(wake.notes)
		}
	}
}

// watchVault runs the watch until it stops.
func watchVault(ctx context.Context, api *API, run func(context.Context), unwatched error) {
	run(ctx)
	// Nothing reaches the window once the watch stops, so from here on the
	// vault is one that is not being followed.
	if unwatched == nil && ctx.Err() == nil {
		api.WatchError.Store("the watch stopped")
	}
}

// indexer brings the index of one vault level with it: its notes first, then its
// books, then the vectors both owe.
type indexer struct {
	made     Assembly
	api      *API
	v        domain.Vault
	embedder port.Embedder
	wake     indexTriggers
	owed     *pendingSources
	out      io.Writer
	// owes is whether the next pass indexes every file again. The ask is spent
	// on the one goroutine that runs the indexer, so it is read and written in
	// one place.
	owes bool
}

// run is the vault's first indexing, and then every indexing that arrives while
// the window is open: one at a time, and never while another is running.
func (p *indexer) run(ctx context.Context, read func(ctx context.Context, during func()) (int, error)) {
	// Indexing the sources comes after the notes: a vault is useful the moment
	// its notes answer, and a library takes minutes to cut and hours to embed.
	// Neither stops the window, and neither has to finish: an index is a cache.
	if p.readNotes(ctx, read) {
		p.indexSources(ctx)
	}

	// A vault whose first indexing failed is one somebody goes on writing in, so
	// the waiting stands whatever that indexing did.
	var quiet <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.wake.sources:
			p.indexSources(ctx)
		case <-p.wake.read:
			p.embedRecognisedPages(ctx)
		case <-p.wake.notes:
			// Every write puts the pass off again: what was typed is embedded
			// once the vault has been still.
			quiet = time.After(p.wake.still)
		case <-quiet:
			quiet = nil
			embedSources(ctx, p.made, p.api, p.v, p.embedder, p.wake.read)
		}
	}
}

// readNotes is the vault's first indexing: the walk, and the notes written while
// it ran read once more. It answers whether the vault was read.
func (p *indexer) readNotes(ctx context.Context, read func(ctx context.Context, during func()) (int, error)) bool {
	defer p.api.finishTask(walkingNotes)

	// The walk runs behind the window, which answers from what it has reached. A
	// later one is the index being brought level with a vault that moved under
	// it.
	p.api.reportTask(task.Task{ID: walkingNotes, Title: "Indexing the vault"})
	notes, err := read(ctx, nil)

	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		// Asked to stop. What it stored is correct as far as it got, and there
		// is nothing to report.
		return false
	default:
		p.api.Error.Store(err.Error())
		return false
	}

	fmt.Fprintf(p.out, "%s: %d notes\n", p.v.Name, notes)
	p.api.Ready.Store(true)
	return true
}

// indexSources indexes the books, again in full where this launch asked for it.
func (p *indexer) indexSources(ctx context.Context) {
	asked := p.owes
	p.owes = false
	readSources(ctx, p.made, p.api, p.v, p.embedder, asked, p.wake.read, p.out)
}

// embedRecognisedPages cuts and embeds what has been read of a document while the
// rest of it is still being read, once a batch of pages is on disk.
func (p *indexer) embedRecognisedPages(ctx context.Context) {
	held := p.owed.drain()
	if len(held) == 0 {
		return
	}
	for path, of := range held {
		rechunkSource(ctx, p.made, p.api, p.embedder, of, path)
	}
	embedSources(ctx, p.made, p.api, p.v, p.embedder, p.wake.read)
}

// readSources takes the text out of every book in the vault and then embeds what
// was cut, reporting what it is extracting as it goes.
//
// Every way is allowed to fail without the window minding. A book that will
// not parse is one book; an embedder that is not configured is the ordinary case,
// and search answers on words alone until one is.
func readSources(
	ctx context.Context,
	made Assembly,
	api *API,
	v domain.Vault,
	embedder port.Embedder,
	rebuild bool,
	nudge chan struct{},
	out io.Writer,
) {
	making, err := made.ReadWholeVault(ctx, embedder, v, rebuild)
	if err != nil {
		api.reportTask(task.Task{ID: extractingBooks, Title: "Extracting books", Error: err.Error()})
		return
	}
	making.OnBooksProgress = func(res source.ExtractResult) {
		api.reportTask(task.Task{
			ID: extractingBooks, Title: "Extracting books", About: res.CurrentSource,
			// Every book the walk found leaves this pass one of several ways, and
			// all four count as done.
			Count: int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total: int64(res.Seen),
		})
	}

	api.reportTask(task.Task{ID: extractingBooks, Title: "Extracting books"})
	res, read := making.ReadBooks(ctx, v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			fmt.Fprintf(out, "%s: %d books, %d chunks\n", v.Name, res.Extracted, res.Chunks)
		}
		api.finishTask(extractingBooks)
	case errors.Is(read, context.Canceled):
		// Asked to stop. What it cut is correct as far as it got.
		api.finishTask(extractingBooks)
	default:
		// A failed pass stays in the list until whoever is shown it takes it
		// out.
		api.reportTask(task.Task{ID: extractingBooks, Title: "Extracting books", Error: read.Error()})
	}

	embedSources(ctx, made, api, v, embedder, nudge)
}

// rechunkSource cuts one source again from whatever its text now says.
//
// A recognition writes a batch of pages and asks for this, so a book being read
// answers questions about the pages that have been read. Failing is one source:
// the next batch asks again.
func rechunkSource(
	ctx context.Context,
	made Assembly,
	api *API,
	embedder port.Embedder,
	v domain.Vault,
	path string,
) {
	cut := func(err error) {
		api.reportTask(task.Task{ID: extractingBooks, Title: "Extracting books", About: path, Error: err.Error()})
	}

	making, err := made.ReadWholeVault(ctx, embedder, v, false)
	if err != nil {
		cut(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// The pages that were read are cut, and a cut that failed before this
		// one is over.
		api.finishTask(extractingBooks)
	case !errors.Is(err, context.Canceled):
		cut(err)
	}
}

// embedSources gives the chunks of the vault the vectors they owe, and reads no
// file the index does not already hold a chunk of.
//
// It is the whole of what a note that was written owes: the chunks are cut
// where the note is stored, and what has no vector is a question for the index.
//
// A recognition's batch of pages stops the pass where it stands. The vectors it
// made are kept, and taken up again it asks the index what still owes one.
func embedSources(
	ctx context.Context,
	made Assembly,
	api *API,
	v domain.Vault,
	embedder port.Embedder,
	nudge chan struct{},
) {
	if embedder == nil {
		return
	}
	owing, owes := getOwedVectors(ctx, api, v)
	if !owes {
		return
	}

	under, stop := stopOnNudge(ctx, nudge)
	defer stop()

	// Fetching the model and preparing it is a step of its own, and it stands in
	// the list under its own name. Nothing is indexed until it is over, and a
	// model that never arrived is said under that name.
	if arrival, ok := embedder.(embedding.WaitingEmbedder); ok {
		if err := arrival.Wait(under); err != nil {
			return
		}
	}
	makeVectors(ctx, under, made, api, v, embedder, owing)
}

// getOwedVectors is what a pass owes, asked once before it starts: the chunks
// that can carry a vector and do not. The pass finds them a few hundred at a
// time, and a total that grows as it goes is a count that never settles.
//
// It is asked before the model is waited for. Every path into the pass runs on
// the one goroutine that also reads the books and cuts what a recognition
// wrote, and a vault owing no vector holds that goroutine for nothing.
func getOwedVectors(ctx context.Context, api *API, v domain.Vault) (owing int64, owes bool) {
	if api.Indexing.Progress == nil {
		return 0, true
	}
	held, embedded, err := api.Indexing.Progress.GetProgress(ctx, v.ID, text(&api.Indexing.Recipe))
	if err != nil {
		return 0, true
	}
	owing = max(0, held-embedded)
	return owing, owing > 0
}

// stopOnNudge is the context a pass runs under, which ends when somebody is
// waiting for the loop that called it, and the function that ends the pass.
//
// The nudge is put back where it was found, so the loop that reads it next
// still has it. Nothing else takes from it while the pass runs: the loop is
// inside this call.
func stopOnNudge(ctx context.Context, nudge chan struct{}) (context.Context, func()) {
	under, aside := context.WithCancel(ctx)
	nudged := make(chan struct{})
	go func() {
		defer close(nudged)
		select {
		case <-under.Done():
		case <-nudge:
			fire(nudge)
			aside()
		}
	}()
	return under, func() {
		aside()
		<-nudged
	}
}

// makeVectors makes the vectors a vault owes, reporting the source it is
// embedding as it goes.
func makeVectors(
	ctx, under context.Context,
	made Assembly,
	api *API,
	v domain.Vault,
	embedder port.Embedder,
	owing int64,
) {
	indexing := func(err error) {
		api.reportTask(task.Task{ID: makingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := made.ReadWholeVault(ctx, embedder, v, false)
	if err != nil {
		indexing(err)
		return
	}
	// Indexing is always of something, and the source open now is what it is of.
	// The pass enters the list when it opens the first of them, so the row is
	// never a word with nothing under it. A share is drawn once a vector has
	// been made, counted over the work in hand and not the size of the vault.
	making.OnVectorsProgress = func(res source.EmbedResult) {
		at := task.Task{ID: makingVectors, Title: "Indexing", About: res.CurrentSource}
		if res.Embedded > 0 {
			at.Count, at.Total = int64(res.Embedded), owing
		}
		api.reportTask(at)
	}

	switch _, err := making.MakeVectors(under, v); {
	case err == nil, errors.Is(err, context.Canceled):
		api.finishTask(makingVectors)
	default:
		// A vault short of the vectors it owes is searched by its words alone,
		// and the reason for it stands in the list.
		indexing(err)
	}
}
