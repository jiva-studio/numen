package indexing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// What each pass is called in the list of what is being done. One name each, so
// a pass that reports itself again replaces itself.
const (
	WalkingNotes    = "walking the notes"
	ExtractingBooks = "extracting the books"
	MakingVectors   = "making the vectors"
)

const taskExtractingBooks = "Extracting books"

// QuietPeriod is how long the vault has to have been still before the notes
// written into it are embedded.
const QuietPeriod = 8 * time.Second

// Events is what the worker tells whoever is showing the vault.
type Events interface {
	// ReportTask puts one piece of work in the list of what is being done, and
	// FinishTask takes it out.
	ReportTask(offset run.Task)
	FinishTask(id string)
	// ReportError says why a pass could not run.
	ReportError(err error)
	// ReportNotes says the notes of a vault were read.
	ReportNotes(v domain.Vault, notes int)
	// ReportBooks says how many books were read and cut.
	ReportBooks(v domain.Vault, res source.ExtractResult)
}

// OpenReading is the three passes over a vault, for the embedder given.
type OpenReading func(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vault.ReadWholeVault, error)

// ReadNotes is the first walk over a vault. It answers how many notes it read.
type ReadNotes func(ctx context.Context, during func()) (int, error)

// GetOwed answers how many chunks the vault holds that can carry a vector, and how
// many carry one.
type GetOwed func(ctx context.Context, v domain.Vault) (held, embedded int64, err error)

// WaitForModel blocks until the embedder's model has turned up, and says what
// stopped it.
type WaitForModel func(ctx context.Context) error

// Config is what a worker is made from.
type Config struct {
	Reading  OpenReading
	Embedder embeddingport.Embedder
	// Wait is nil for an embedder whose model is always there.
	Wait WaitForModel
	// Owed is nil where nothing says how many vectors a vault owes, and then
	// every pass assumes it owes some.
	Owed   GetOwed
	Events Events
	// Quiet is how long the vault has to have been still before a note that was
	// written is embedded, and After is the timer that counts it.
	Quiet time.Duration
	After func(time.Duration) <-chan time.Time
}

// Worker brings the index of the vault a window shows level with it, one pass
// at a time, and never while another is running.
//
// Work reaches it in three ways once the first pass is over: a book, which is
// found and cut before anything is embedded, a note, which arrives already cut
// and owes only its vectors, and a source a recognition wrote more of.
type Worker struct {
	reading  OpenReading
	embedder embeddingport.Embedder
	wait     WaitForModel
	owed     GetOwed
	events   Events
	debounce time.Duration
	after    func(time.Duration) <-chan time.Time

	sources chan struct{}
	notes   chan struct{}
	read    chan struct{}
}

func NewWorker(cfg Config) *Worker {
	return &Worker{
		reading:  cfg.Reading,
		embedder: cfg.Embedder,
		wait:     cfg.Wait,
		owed:     cfg.Owed,
		events:   cfg.Events,
		debounce: cfg.Quiet,
		after:    cfg.After,
		sources:  make(chan struct{}, 1),
		notes:    make(chan struct{}, 1),
		read:     make(chan struct{}, 1),
	}
}

// WakeSources leaves one pass over the books waiting.
func (w *Worker) WakeSources() { signal(w.sources) }

// WakeNotes leaves one pass over the notes' vectors waiting. Several changes at
// once are one pass.
func (w *Worker) WakeNotes() { signal(w.notes) }

func signal(wake chan struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

// Session is the work of one vault: the sources a recognition has written more
// of than their chunks account for, and the passes over the vault.
type Session struct {
	worker *Worker

	mu      sync.Mutex
	pending map[string]domain.Vault
}

// NewSession is the work of a vault that has not been started.
func (w *Worker) NewSession() *Session {
	return &Session{worker: w}
}

// RequestCut asks for the source at a path to be cut again from its text. A
// source appears once however many batches it wrote.
func (s *Session) RequestCut(v domain.Vault, path string) {
	s.mu.Lock()
	if s.pending == nil {
		s.pending = map[string]domain.Vault{}
	}
	s.pending[path] = v
	s.mu.Unlock()
	signal(s.worker.read)
}

// ReportVersionsChanged asks for the file to be cut again from its live text
// version, which is what the index does whenever the versions of a file change.
func (s *Session) ReportVersionsChanged(v domain.Vault, path string) {
	s.RequestCut(v, path)
}

// drain is every source waiting, and leaves none behind.
func (s *Session) drain() map[string]domain.Vault {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.pending
	s.pending = nil
	return target
}

// Run is the first indexing of a vault, and then every indexing that arrives
// until ctx ends. rebuild indexes every file again in the first pass over the
// books.
func (s *Session) Run(ctx context.Context, v domain.Vault, rebuild bool, read ReadNotes) {
	p := &pass{worker: s.worker, session: s, v: v, owes: rebuild}
	p.run(ctx, read)
}

// pass is the indexing of one vault, on the one goroutine that runs it.
type pass struct {
	worker  *Worker
	session *Session
	v       domain.Vault
	// owes is whether the next pass over the books indexes every file again.
	owes bool
}

func (p *pass) run(ctx context.Context, read ReadNotes) {
	// The books come after the notes: a vault is useful the moment its notes
	// answer, and a library takes minutes to cut and hours to embed. An index is
	// a cache, so neither has to finish.
	if p.readNotes(ctx, read) {
		p.indexSources(ctx)
	}

	// A vault whose first indexing failed is one somebody goes on writing in, so
	// the waiting stands whatever that indexing did.
	var debounceTimer <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.worker.sources:
			p.indexSources(ctx)
		case <-p.worker.read:
			p.embedRecognised(ctx)
		case <-p.worker.notes:
			// Every write puts the pass off again.
			debounceTimer = p.worker.after(p.worker.debounce)
		case <-debounceTimer:
			debounceTimer = nil
			p.embed(ctx)
		}
	}
}

// readNotes is the walk, and the notes written while it ran read once more. It
// answers whether the vault was read.
func (p *pass) readNotes(ctx context.Context, read ReadNotes) bool {
	events := p.worker.events
	defer events.FinishTask(WalkingNotes)

	events.ReportTask(run.Task{ID: WalkingNotes, Title: "Indexing the vault"})
	notes, err := read(ctx, nil)

	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		// Asked to stop. What it stored is correct as far as it got.
		return false
	default:
		events.ReportError(err)
		return false
	}

	events.ReportNotes(p.v, notes)
	return true
}

// indexSources indexes the books, again in full where the first launch asked for it.
func (p *pass) indexSources(ctx context.Context) {
	request := p.owes
	p.owes = false
	p.readBooks(ctx, request)
}

// embedRecognised cuts and embeds what has been read of a document while the
// rest of it is still being read.
func (p *pass) embedRecognised(ctx context.Context) {
	target := p.session.drain()
	if len(target) == 0 {
		return
	}
	for path, of := range target {
		p.cutSource(ctx, of, path)
	}
	p.embed(ctx)
}

// readBooks takes the text out of every book in the vault and then embeds what
// was cut. A book that will not parse is one book, and an embedder that is not
// configured is the ordinary case.
func (p *pass) readBooks(ctx context.Context, rebuild bool) {
	events := p.worker.events
	making, err := p.worker.reading(ctx, p.worker.embedder, p.v, rebuild)
	if err != nil {
		events.ReportTask(run.Task{ID: ExtractingBooks, Title: taskExtractingBooks, Error: err.Error()})
		return
	}
	making.OnBooksProgress = func(res source.ExtractResult) {
		events.ReportTask(run.Task{
			ID: ExtractingBooks, Title: taskExtractingBooks, About: res.CurrentSource,
			// Every book the walk found leaves this pass one of four ways, and all
			// four count as done.
			Count: int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total: int64(res.Seen),
		})
	}

	events.ReportTask(run.Task{ID: ExtractingBooks, Title: taskExtractingBooks})
	res, read := making.ReadBooks(ctx, p.v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			events.ReportBooks(p.v, res)
		}
		events.FinishTask(ExtractingBooks)
	case errors.Is(read, context.Canceled):
		// Asked to stop. What it cut is correct as far as it got.
		events.FinishTask(ExtractingBooks)
	default:
		// A failed pass stays in the list until whoever is shown it takes it out.
		events.ReportTask(run.Task{ID: ExtractingBooks, Title: taskExtractingBooks, Error: read.Error()})
	}

	p.embed(ctx)
}

// cutSource cuts one source again from whatever its text now says. Failing is
// one source: the next batch asks again.
func (p *pass) cutSource(ctx context.Context, v domain.Vault, path string) {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(run.Task{ID: ExtractingBooks, Title: taskExtractingBooks, About: path, Error: err.Error()})
	}

	making, err := p.worker.reading(ctx, p.worker.embedder, v, false)
	if err != nil {
		failed(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// A cut that failed before this one is over.
		events.FinishTask(ExtractingBooks)
	case !errors.Is(err, context.Canceled):
		failed(err)
	}
}

// embed gives the chunks of the vault the vectors they owe, and reads no file
// the index does not already hold a chunk of.
//
// A recognition's batch of pages stops the pass at its current position. The vectors it
// made are kept, and taken up again it asks the index what still owes one.
func (p *pass) embed(ctx context.Context) {
	w := p.worker
	if w.embedder == nil {
		return
	}
	owing, owes := w.getOwedVectors(ctx, p.v)
	if !owes {
		return
	}

	limit, stop := stopOnWake(ctx, w.read)
	defer stop()

	// Fetching the model is a step of its own, and nothing is indexed until it
	// is over.
	if w.wait != nil {
		if err := w.wait(limit); err != nil {
			return
		}
	}
	p.makeVectors(ctx, limit, owing)
}

// getOwedVectors is what a pass owes, asked once before it starts. The pass
// finds the chunks a few hundred at a time, and a total that grows as it goes
// is a count that never settles.
//
// It is asked before the model is waited for: every path into the pass runs on
// the one goroutine that also reads the books, and a vault owing no vector
// holds that goroutine for nothing.
func (w *Worker) getOwedVectors(ctx context.Context, v domain.Vault) (owing int64, owes bool) {
	if w.owed == nil {
		return 0, true
	}
	target, embedded, err := w.owed(ctx, v)
	if err != nil {
		return 0, true
	}
	owing = max(0, target-embedded)
	return owing, owing > 0
}

// stopOnWake is the context a pass runs under, which ends when somebody is
// waiting for the loop that called it, and the function that ends the pass.
//
// The wake is put back where it was found, and the loop that reads it next
// still has it.
func stopOnWake(ctx context.Context, wake chan struct{}) (context.Context, func()) {
	underCtx, end := context.WithCancel(ctx)
	awakened := make(chan struct{})
	go func() {
		defer close(awakened)
		select {
		case <-underCtx.Done():
		case <-wake:
			signal(wake)
			end()
		}
	}()
	return underCtx, func() {
		end()
		<-awakened
	}
}

// makeVectors makes the vectors a vault owes, reporting the source it is
// embedding as it goes.
func (p *pass) makeVectors(ctx, items context.Context, owing int64) {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(run.Task{ID: MakingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := p.worker.reading(ctx, p.worker.embedder, p.v, false)
	if err != nil {
		failed(err)
		return
	}
	// Indexing is always of something, and the source open now is what it is of.
	// The pass enters the list when it opens the first of them, and a share is
	// drawn once a vector has been made, counted over the work in hand.
	making.OnVectorsProgress = func(res embedding.EmbedResult) {
		pos := run.Task{ID: MakingVectors, Title: "Indexing", About: res.CurrentSource}
		if res.Embedded > 0 {
			pos.Count, pos.Total = int64(res.Embedded), owing
		}
		events.ReportTask(pos)
	}

	switch _, err := making.MakeVectors(items, p.v); {
	case err == nil, errors.Is(err, context.Canceled):
		events.FinishTask(MakingVectors)
	default:
		// A vault short of the vectors it owes is searched by its words alone, and
		failed(err)
	}
}
