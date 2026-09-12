package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// settled is how long the vault has to have been still before the notes written
// into it are embedded. It is longer than the bound in the editor window's
// note/tab.ts, which writes an unfinished edit every five seconds while a
// person goes on typing.
const settled = 8 * time.Second

// nudges are the two ways work reaches the reading behind the window once the
// first pass is over: a book, which is found and cut before anything is
// embedded, and a note, which arrives already cut and owes only its vectors.
type nudges struct {
	sources chan struct{}
	notes   chan struct{}
	// read is one source with more text than its chunks account for, which is
	// what a recognition leaves behind every batch of pages.
	read chan struct{}
	// still is how long the vault has to have been quiet before a note that was
	// written is embedded.
	still time.Duration
}

func waking(still time.Duration) nudges {
	return nudges{
		sources: make(chan struct{}, 1),
		notes:   make(chan struct{}, 1),
		read:    make(chan struct{}, 1),
		still:   still,
	}
}

// raise leaves one nudge waiting. What owes work is asked of the index, so
// several changes at once are one pass.
func raise(nudge chan struct{}) {
	select {
	case nudge <- struct{}{}:
	default:
	}
}

// pending is the sources a recognition has written more of than their chunks
// account for.
//
// A source stands here once, however many batches it wrote, and several stand
// at a time. It belongs to the vault that was being read and goes with it.
type pending struct {
	mu    sync.Mutex
	paths map[string]domain.Vault
}

func (p *pending) put(v domain.Vault, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paths == nil {
		p.paths = map[string]domain.Vault{}
	}
	p.paths[path] = v
}

// take is everything waiting, and leaves nothing behind.
func (p *pending) take() map[string]domain.Vault {
	p.mu.Lock()
	defer p.mu.Unlock()
	held := p.paths
	p.paths = nil
	return held
}

// begin starts the watch and the first scan together, and answers with what
// waits for both to stop.
//
// The watch is acted on alongside the scan: a change reaches the window through
// it and through nothing else.
//
// A vault that cannot be watched is still a vault: the application keeps
// working, and says that changes will not appear by themselves.
func begin(
	ctx context.Context,
	v domain.Vault,
	cfg container.Config,
	db *container.Index,
	api *API,
	opening *container.VaultOpener,
	readers port.VaultReaders,
	embedder port.Embedder,
	wake nudges,
	owed *pending,
	out io.Writer,
) func() {
	trouble := func(err error) {
		if err == nil {
			api.Failed.Store("")
			return
		}
		api.Failed.Store(err.Error())
	}
	opening.Trouble = trouble
	opening.Told = func(m container.VaultChanges) {
		// A client draws every file the vault holds, so an asset is named to it
		// the way a note is.
		api.Listeners.tell(change{paths: slices.Concat(m.Paths, m.Assets), reload: m.Reload})
		if m.Reading() {
			// A book dropped into an open vault is read without anybody asking.
			raise(wake.sources)
		}
		if len(m.Paths) > 0 {
			// A note written is a note cut again, and its chunks owe their
			// vectors. Which ones is not carried: the debt is in the index.
			raise(wake.notes)
		}
	}

	// Said before the goroutine starts, so a window that opens on a fresh vault
	// is shown the walk from its first moment.
	api.say(task.Task{ID: walkingNotes, Doing: "Reading the vault"})

	var running sync.WaitGroup

	open := opening.Begin(ctx, v)
	if why := open.Unwatched(); why != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", v.Name, why)
		api.Unwatched.Store(why.Error())
	}
	running.Add(1)
	go func() {
		defer running.Done()

		open.Run(ctx)
		// Nothing reaches the window once the watch stops, so from here on the
		// vault is one that is not being followed.
		if open.Unwatched() == nil && ctx.Err() == nil {
			api.Unwatched.Store("the watch stopped")
		}
	}()

	// first is the vault's first reading: the walk, and the notes written while
	// it ran read once more. It answers whether the vault was read.
	first := func() bool {
		defer api.finished(walkingNotes)

		// The walk runs behind the window, which answers from what it has
		// reached. A later one is the index being brought level with a vault
		// that moved under it.
		api.say(task.Task{ID: walkingNotes, Doing: "Reading the vault"})
		notes, err := open.Read(ctx, nil)

		switch {
		case err == nil:
		case errors.Is(err, context.Canceled):
			// Asked to stop. What it stored is correct as far as it got, and
			// there is nothing to report.
			return false
		default:
			api.Failed.Store(err.Error())
			return false
		}

		fmt.Fprintf(out, "%s: %d notes\n", v.Name, notes)
		api.Ready.Store(true)
		return true
	}

	// Reading every file again is what this launch was asked for, and one pass
	// makes it. Every pass after it reads what changed. The ask is spent on the
	// one goroutine below, so it is read and written in one place.
	rebuild := cfg.RebuildIndex
	cfg.RebuildIndex = false
	reading := func() {
		asked := cfg
		asked.RebuildIndex, rebuild = rebuild, false
		readSources(ctx, asked, db, api, v, readers, embedder, wake.read, out)
	}

	running.Add(1)
	go func() {
		defer running.Done()

		// Reading the sources comes after the notes: a vault is useful the
		// moment its notes answer, and a library takes minutes to cut and hours
		// to embed. Neither stops the window, and neither has to finish: an
		// index is a cache.
		if first() {
			reading()
		}

		// What arrives while the window is open is read where the first reading
		// was: one at a time, and never while another is running. A vault whose
		// first reading failed is one somebody goes on writing in, so the
		// waiting stands whatever that reading did.
		var quiet <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-wake.sources:
				reading()
			case <-wake.read:
				// A batch of pages is on disk. What has been read of the
				// document is cut and embedded while the rest of it is still
				// being read.
				if held := owed.take(); len(held) > 0 {
					for path, of := range held {
						cutSource(ctx, cfg, db, api, embedder, of, path)
					}
					embedSources(ctx, cfg, db, api, v, readers, embedder, wake.read)
				}
			case <-wake.notes:
				// Every write puts the pass off again: what was typed is
				// embedded once the vault has been still.
				quiet = time.After(wake.still)
			case <-quiet:
				quiet = nil
				embedSources(ctx, cfg, db, api, v, readers, embedder, wake.read)
			}
		}
	}()

	return running.Wait
}

// readSources takes the text out of every book in the vault and then embeds what
// was cut, reporting what it is reading as it goes.
//
// Every way is allowed to fail without the window minding. A book that will
// not parse is one book; an embedder that is not configured is the ordinary case,
// and search answers on words alone until one is.
func readSources(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	v domain.Vault,
	readers port.VaultReaders,
	embedder port.Embedder,
	nudge chan struct{},
	out io.Writer,
) {
	making, err := cfg.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", Failed: err.Error()})
		return
	}
	making.Books.OnProgress = func(res source.ExtractResult) {
		api.say(task.Task{
			ID: readingBooks, Doing: "Reading books", About: res.Reading,
			// Every book the walk found leaves this pass one of several ways, and
			// all four count as done.
			Count: int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total: int64(res.Seen),
		})
	}

	api.say(task.Task{ID: readingBooks, Doing: "Reading books"})
	res, read := making.ReadBooks(ctx, v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			fmt.Fprintf(out, "%s: %d books, %d chunks\n", v.Name, res.Extracted, res.Chunks)
		}
		api.finished(readingBooks)
	case errors.Is(read, context.Canceled):
		// Asked to stop. What it cut is correct as far as it got.
		api.finished(readingBooks)
	default:
		// A failed pass stays in the list until whoever is shown it takes it
		// out.
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", Failed: read.Error()})
	}

	embedSources(ctx, cfg, db, api, v, readers, embedder, nudge)
}

// cutSource cuts one source again from whatever its text now says.
//
// A recognition writes a batch of pages and asks for this, so a book being read
// answers questions about the pages that have been read. Failing is one source:
// the next batch asks again.
func cutSource(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	embedder port.Embedder,
	v domain.Vault,
	path string,
) {
	cut := func(err error) {
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", About: path, Failed: err.Error()})
	}

	making, err := cfg.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		cut(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// The pages that were read are cut, and a cut that failed before this
		// one is over.
		api.finished(readingBooks)
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
	cfg container.Config,
	db *container.Index,
	api *API,
	v domain.Vault,
	readers port.VaultReaders,
	embedder port.Embedder,
	nudge chan struct{},
) {
	if embedder == nil {
		return
	}

	// What this pass owes, asked once before it starts: the chunks that can
	// carry a vector and do not. The pass finds them a few hundred at a time,
	// and a total that grows as it goes is a count that never settles.
	//
	// It is asked before the model is waited for. Every path into this pass runs
	// on the one goroutine that also reads the books and cuts what a recognition
	// wrote, and a vault owing no vector holds that goroutine for nothing.
	owing := int64(0)
	if api.Indexing.Progress != nil {
		held, embedded, err := api.Indexing.Progress.Progress(ctx, v.ID, text(&api.Indexing.Recipe))
		if err == nil {
			owing = max(0, held-embedded)
			if owing == 0 {
				return
			}
		}
	}

	// The nudge is put back where it was found, so the loop that reads it next
	// still has it. Nothing else takes from it while this pass runs: the loop is
	// inside this call.
	under, aside := context.WithCancel(ctx)
	nudged := make(chan struct{})
	go func() {
		defer close(nudged)
		select {
		case <-under.Done():
		case <-nudge:
			raise(nudge)
			aside()
		}
	}()
	defer func() {
		aside()
		<-nudged
	}()

	// Fetching the model and preparing it is a step of its own, and it stands in
	// the list under its own name. Nothing is indexed until it is over, and a
	// model that never arrived is said under that name.
	if arrival, ok := embedder.(embedding.WaitingEmbedder); ok {
		if err := arrival.Wait(under); err != nil {
			return
		}
	}

	indexing := func(err error) {
		api.say(task.Task{ID: makingVectors, Doing: "Indexing", Failed: err.Error()})
	}

	making, err := cfg.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		indexing(err)
		return
	}
	// Indexing is always of something, and the source open now is what it is of.
	// The pass enters the list when it opens the first of them, so the row is
	// never a word with nothing under it. A share is drawn once a vector has
	// been made, counted over the work in hand and not the size of the vault.
	making.Vectors.OnProgress = func(res source.EmbedResult) {
		at := task.Task{ID: makingVectors, Doing: "Indexing", About: res.Reading}
		if res.Embedded > 0 {
			at.Count, at.Total = int64(res.Embedded), owing
		}
		api.say(at)
	}

	switch _, err := making.MakeVectors(under, v); {
	case err == nil, errors.Is(err, context.Canceled):
		api.finished(makingVectors)
	default:
		// A vault short of the vectors it owes is searched by its words alone,
		// and the reason for it stands in the list.
		indexing(err)
	}
}
