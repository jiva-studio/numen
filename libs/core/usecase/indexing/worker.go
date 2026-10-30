package indexing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// What each pass is called in the list of what is being done. One name each, so
// a pass that reports itself again replaces itself.
const (
	WalkingNotes  = "walking the notes"
	IndexingTexts = "indexing the file texts"
	MakingVectors = "making the vectors"
	// CalibratingFloor is the measurement of the similarity floor that follows the
	// last vector of a vault.
	CalibratingFloor = "calibrating the floor"
)

const taskIndexingTexts = "Indexing file texts"

// QuietPeriod is how long the vault has to have been still before the notes
// written into it are embedded.
const QuietPeriod = 8 * time.Second

// Events receives the progress and results of indexing a vault.
type Events interface {
	// ReportTask puts one piece of work in the list of what is being done, and
	// FinishTask takes it out.
	ReportTask(offset vaultreading.Task)
	FinishTask(id string)
	// ReportError reports why a pass could not run.
	ReportError(err error)
	// ReportNotes reports that the notes of a vault were read.
	ReportNotes(v domain.Vault, notes int)
	// ReportTexts reports how many file texts were cut.
	ReportTexts(v domain.Vault, res vaultreading.IndexResult)
}

// OpenReading is the three passes over a vault, for the embedder given.
type OpenReading func(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vaultreading.ReadWholeVault, error)

// ReadNotes is the first walk over a vault. It answers how many notes it read.
type ReadNotes func(ctx context.Context, during func()) (int, error)

// GetVectorCounts answers how many chunks the vault holds that can carry a vector, and how
// many carry one.
type GetVectorCounts func(ctx context.Context, v domain.Vault) (held, embedded int64, err error)

// Calibrate measures the similarity floor of a vault once all its vectors exist,
// and does nothing where the floor is already measured or set.
type Calibrate func(ctx context.Context, v domain.Vault) error

// Config is what a worker is made from.
type Config struct {
	Reading  OpenReading
	Embedder embeddingport.Embedder
	// VectorCounts is nil when the number of pending vectors is unknown; every pass then
	// assumes some vectors are pending.
	VectorCounts GetVectorCounts
	// Calibrate is nil where no floor is calibrated.
	Calibrate Calibrate
	Events    Events
	// Quiet is how long the vault has to have been still before a note that was
	// written is embedded, and After is the timer that counts it.
	Quiet time.Duration
	After func(time.Duration) <-chan time.Time
}

// Worker keeps the index of a vault in sync with its content, one pass at a
// time, and never while another pass is running.
//
// After the first pass, work arrives in three forms: a file with text, which is
// found and cut before anything is embedded; a note, which arrives already cut
// and needs only its vectors; and a file a plugin extended with more text.
type Worker struct {
	reading      OpenReading
	embedder     embeddingport.Embedder
	vectorCounts GetVectorCounts
	calibrate    Calibrate
	events       Events
	debounce     time.Duration
	after        func(time.Duration) <-chan time.Time

	sources chan struct{}
	notes   chan struct{}
	read    chan struct{}
}

func NewWorker(cfg Config) *Worker {
	return &Worker{
		reading:      cfg.Reading,
		embedder:     cfg.Embedder,
		vectorCounts: cfg.VectorCounts,
		calibrate:    cfg.Calibrate,
		events:       cfg.Events,
		debounce:     cfg.Quiet,
		after:        cfg.After,
		sources:      make(chan struct{}, 1),
		notes:        make(chan struct{}, 1),
		read:         make(chan struct{}, 1),
	}
}

// WakeSources leaves one pass over the file texts waiting.
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

// RequestCut requests that the source at a path be cut again from its text. A
// source is cut once however many batches wrote it.
func (s *Session) RequestCut(v domain.Vault, path string) {
	s.mu.Lock()
	if s.pending == nil {
		s.pending = map[string]domain.Vault{}
	}
	s.pending[path] = v
	s.mu.Unlock()
	signal(s.worker.read)
}

// ReportVersionsChanged requests that the file be cut again from its live text
// version. The index issues it whenever the versions of a file change.
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
// file texts.
func (s *Session) Run(ctx context.Context, v domain.Vault, rebuild bool, read ReadNotes) {
	p := &pass{worker: s.worker, session: s, v: v, isRebuild: rebuild}
	p.run(ctx, read)
}

// pass is the indexing of one vault, on the one goroutine that runs it.
type pass struct {
	worker  *Worker
	session *Session
	v       domain.Vault
	// isRebuild is true when the next pass over the file texts indexes every file again.
	isRebuild bool
}

func (p *pass) run(ctx context.Context, read ReadNotes) {
	// The file texts come after the notes: a vault is useful the moment its
	// notes answer, and a library takes minutes to cut and hours to embed. An index is
	// a cache, so neither has to finish.
	if p.readNotes(ctx, read) {
		p.indexSources(ctx)
	}

	// A vault whose first indexing failed still receives writes, so the wait
	// proceeds regardless of the indexing outcome.
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

	events.ReportTask(vaultreading.Task{ID: WalkingNotes, Title: "Indexing the vault"})
	notes, err := read(ctx, nil)

	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		// Stopped on request. The stored data is correct up to that point.
		return false
	default:
		events.ReportError(err)
		return false
	}

	events.ReportNotes(p.v, notes)
	return true
}

// indexSources indexes the file texts, in full when the first launch requested it.
func (p *pass) indexSources(ctx context.Context) {
	request := p.isRebuild
	p.isRebuild = false
	p.indexTexts(ctx, request)
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

// indexTexts cuts the text versions of every file in the vault and then embeds
// what was cut. A file that will not read is one file, and an embedder that is
// not configured is the ordinary case.
func (p *pass) indexTexts(ctx context.Context, rebuild bool) {
	events := p.worker.events
	making, err := p.worker.reading(ctx, p.worker.embedder, p.v, rebuild)
	if err != nil {
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, Error: err.Error()})
		return
	}
	making.OnTextsProgress = func(res vaultreading.IndexResult) {
		events.ReportTask(vaultreading.Task{
			ID: IndexingTexts, Title: taskIndexingTexts, About: res.CurrentSource,
			// Every file the walk found leaves this pass one of four ways, and all
			// four count as done.
			Count: int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total: int64(res.Seen),
		})
	}

	events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts})
	res, read := making.IndexTexts(ctx, p.v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			events.ReportTexts(p.v, res)
		}
		events.FinishTask(IndexingTexts)
	case errors.Is(read, context.Canceled):
		// Stopped on request. The cut data is correct up to that point.
		events.FinishTask(IndexingTexts)
	default:
		// A failed pass stays in the task list until the user dismisses it.
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, Error: read.Error()})
	}

	p.embed(ctx)
}

// cutSource cuts one source again from its current text. A failure affects
// only that source; the next batch retries it.
func (p *pass) cutSource(ctx context.Context, v domain.Vault, path string) {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, About: path, Error: err.Error()})
	}

	making, err := p.worker.reading(ctx, p.worker.embedder, v, false)
	if err != nil {
		failed(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// A cut that failed before this one is over.
		events.FinishTask(IndexingTexts)
	case !errors.Is(err, context.Canceled):
		failed(err)
	}
}

// embed creates the vectors for the chunks of the vault that lack one, and
// reads no file the index does not already hold a chunk of.
//
// A recognition batch of pages, and a change of the sources (a book added, or
// the embedding model changed), stop the pass at its current position. The
// vectors already created are kept, and the resumed pass queries the index for
// the chunks that still lack a vector, under the recipe in force by then.
func (p *pass) embed(ctx context.Context) {
	w := p.worker
	if w.embedder == nil {
		return
	}
	pending, hasPending := w.getPendingVectors(ctx, p.v)
	if !hasPending {
		w.events.FinishTask(MakingVectors)
		p.calibrateFloor(ctx)
		return
	}

	limit, stop := stopOnWake(ctx, w.read, w.sources)
	defer stop()

	if p.makeVectors(ctx, limit, pending) {
		p.calibrateFloor(ctx)
	}
}

// calibrateFloor measures the similarity floor of the vault once the pass has
// left no vector pending. A failure is reported as a task and does not fail the pass.
func (p *pass) calibrateFloor(ctx context.Context) {
	w := p.worker
	if w.calibrate == nil || ctx.Err() != nil {
		return
	}
	switch err := w.calibrate(ctx, p.v); {
	case err == nil:
		w.events.FinishTask(CalibratingFloor)
	case errors.Is(err, context.Canceled):
	default:
		w.events.ReportTask(vaultreading.Task{ID: CalibratingFloor, Title: "Calibrating the search", Error: err.Error()})
	}
}

// getPendingVectors returns the number of vectors a pass must create, computed once
// before the pass starts. Chunks are discovered a few hundred at a time, so a
// total derived during the pass would keep growing.
func (w *Worker) getPendingVectors(ctx context.Context, v domain.Vault) (pending int64, hasPending bool) {
	if w.vectorCounts == nil {
		return 0, true
	}
	target, embedded, err := w.vectorCounts(ctx, v)
	if err != nil {
		return 0, true
	}
	pending = max(0, target-embedded)
	return pending, pending > 0
}

// stopOnWake returns the context a pass runs under, which is cancelled when a
// wake signal is pending on any of the channels given, and the function that
// ends the pass.
//
// A wake signal is restored after it is consumed, so the calling loop still
// receives it.
func stopOnWake(ctx context.Context, wakes ...chan struct{}) (context.Context, func()) {
	underCtx, end := context.WithCancel(ctx)
	var watchers sync.WaitGroup
	for _, wake := range wakes {
		watchers.Add(1)
		go func() {
			defer watchers.Done()
			select {
			case <-underCtx.Done():
			case <-wake:
				signal(wake)
				end()
			}
		}()
	}
	return underCtx, func() {
		end()
		watchers.Wait()
	}
}

// makeVectors creates the pending vectors of a vault, reporting the source being
// embedded. It reports whether the pass ended with no vector pending.
func (p *pass) makeVectors(ctx, items context.Context, pending int64) bool {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(vaultreading.Task{ID: MakingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := p.worker.reading(ctx, p.worker.embedder, p.v, false)
	if err != nil {
		failed(err)
		return false
	}
	// Progress is reported for the source currently open. The task enters the
	// list when the first source opens, and a progress share is reported once a
	// vector has been created, counted against the total pending.
	making.OnVectorsProgress = func(res vaultreading.EmbedResult) {
		pos := vaultreading.Task{ID: MakingVectors, Title: "Indexing", About: res.CurrentSource}
		if res.Embedded > 0 {
			pos.Count, pos.Total = int64(res.Embedded), pending
		}
		events.ReportTask(pos)
	}

	switch _, err := making.MakeVectors(items, p.v); {
	case err == nil:
		events.FinishTask(MakingVectors)
		return true
	case errors.Is(err, context.Canceled):
		events.FinishTask(MakingVectors)
	default:
		// A vault missing vectors is searched by its text alone.
		failed(err)
	}
	return false
}
