package indexing

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
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

// VectorsReady is notified that all the vectors of a vault exist, but those of the sources whose indexing
// a person cancelled: those stay without vectors until they change. It is notified after every pass that
// leaves no other vector pending, so it is notified more than once for a vault.
type VectorsReady func(ctx context.Context, v domain.Vault)

// Config is what a worker is made from.
type Config struct {
	Reading      OpenReading
	Embedder     embeddingport.Embedder
	VectorCounts GetVectorCounts
	VectorsReady VectorsReady
	Events       Events
	Quiet        time.Duration
	After        func(time.Duration) <-chan time.Time
}

// Worker keeps the index of a vault in sync with its content, one pass at a
// time, and never while another pass is running.
// After the first pass, work arrives as a file with text, found and cut before
// anything is embedded; a note, which arrives already cut and needs only its
// vectors; and a file a plugin extended with more text.
type Worker struct {
	reading      OpenReading
	embedder     embeddingport.Embedder
	vectorCounts GetVectorCounts
	vectorsReady VectorsReady
	events       Events
	debounce     time.Duration
	after        func(time.Duration) <-chan time.Time

	sources chan struct{}
	notes   chan struct{}
	read    chan struct{}

	skips *skipSet
}

func NewWorker(cfg Config) *Worker {
	return &Worker{
		reading:      cfg.Reading,
		embedder:     cfg.Embedder,
		vectorCounts: cfg.VectorCounts,
		vectorsReady: cfg.VectorsReady,
		events:       cfg.Events,
		debounce:     cfg.Quiet,
		after:        cfg.After,
		sources:      make(chan struct{}, 1),
		notes:        make(chan struct{}, 1),
		read:         make(chan struct{}, 1),
		skips:        newSkipSet(),
	}
}

// WakeSources leaves one pass over the file texts waiting.
func (w *Worker) WakeSources() { signal(w.sources) }

// WakeNotes leaves one pass over the notes' vectors waiting. Several changes at
// once are one pass.
func (w *Worker) WakeNotes() { signal(w.notes) }

// ClearSkips indexes again the sources at the paths given whose indexing a
// person cancelled. The vault calls it for every path it reports changed.
func (w *Worker) ClearSkips(paths []string) { w.skips.clear(paths) }

func (w *Worker) openReading(
	ctx context.Context, v domain.Vault, rebuild bool,
) (vaultreading.ReadWholeVault, error) {
	making, err := w.reading(ctx, w.embedder, v, rebuild)
	if err != nil {
		return making, err
	}
	making.IsTextSkipped = w.skips.isTextSkipped
	making.IsVectorSkipped = w.skips.isVectorSkipped
	return making, nil
}

// taskCancel ends the context of one pass, and records whether a person asked
// for it through the task.
type taskCancel struct {
	stop        context.CancelFunc
	isRequested atomic.Bool
}

func newTaskCancel(ctx context.Context) (context.Context, *taskCancel) {
	inner, stop := context.WithCancel(ctx)
	return inner, &taskCancel{stop: stop}
}

func (c *taskCancel) request() {
	c.isRequested.Store(true)
	c.stop()
}

// currentSources are the sources a pass reports it is working on. Nothing reported leaves the last
// ones in place.
type currentSources struct {
	mu    sync.Mutex
	paths []string
}

func (c *currentSources) set(paths []string) {
	if len(paths) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths = slices.Clone(paths)
}

func (c *currentSources) get() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.paths)
}

func getEmbeddingSources(res vaultreading.EmbedResult) []string {
	if len(res.Active) > 0 {
		return res.Active
	}
	if res.CurrentSource != "" {
		return []string{res.CurrentSource}
	}
	return nil
}

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
	worker    *Worker
	session   *Session
	v         domain.Vault
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

func (p *pass) indexSources(ctx context.Context) {
	request := p.isRebuild
	p.isRebuild = false
	p.indexTexts(ctx, request)
}

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

func (p *pass) indexTexts(ctx context.Context, rebuild bool) {
	events := p.worker.events
	making, err := p.worker.openReading(ctx, p.v, rebuild)
	if err != nil {
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, Error: err.Error()})
		return
	}
	texts, cancel := newTaskCancel(ctx)
	defer cancel.stop()
	var current currentSources
	making.OnTextsProgress = func(res vaultreading.IndexResult) {
		if res.CurrentSource != "" {
			current.set([]string{res.CurrentSource})
		}
		events.ReportTask(vaultreading.Task{
			ID: IndexingTexts, Title: taskIndexingTexts, About: res.CurrentSource,
			// Every file the walk found leaves this pass one of four ways, and all
			// four count as done.
			Count:  int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total:  int64(res.Seen),
			Cancel: cancel.request,
		})
	}

	events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, Cancel: cancel.request})
	res, read := making.IndexTexts(texts, p.v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			events.ReportTexts(p.v, res)
		}
		events.FinishTask(IndexingTexts)
	case errors.Is(read, context.Canceled):
		// Stopped on request. The cut data is correct up to that point, and a
		// source a person cancelled is left uncut until it changes.
		if cancel.isRequested.Load() {
			for _, path := range current.get() {
				p.worker.skips.addTexts(path)
			}
		}
		events.FinishTask(IndexingTexts)
	default:
		// A failed pass stays in the task list until the user dismisses it.
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, Error: read.Error()})
	}

	p.embed(ctx)
}

func (p *pass) cutSource(ctx context.Context, v domain.Vault, path string) {
	if p.worker.skips.isTextSkipped(path) {
		return
	}
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(vaultreading.Task{ID: IndexingTexts, Title: taskIndexingTexts, About: path, Error: err.Error()})
	}

	making, err := p.worker.openReading(ctx, v, false)
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

func (p *pass) embed(ctx context.Context) {
	w := p.worker
	if w.embedder == nil {
		return
	}
	plan := w.planVectors(ctx, p.v)
	if !plan.hasPending {
		w.events.FinishTask(MakingVectors)
		p.reportVectorsReady(ctx)
		return
	}

	limit, stop := stopOnWake(ctx, w.read, w.sources, w.notes)
	defer stop()

	if p.makeVectors(ctx, limit, plan) {
		p.reportVectorsReady(ctx)
	}
}

func (p *pass) reportVectorsReady(ctx context.Context) {
	w := p.worker
	if w.vectorsReady == nil || ctx.Err() != nil {
		return
	}
	w.vectorsReady(ctx, p.v)
}

// vectorPlan is what a vector pass must create, computed once before the pass starts. Chunks are
// discovered a few hundred at a time, so a total derived during the pass would keep growing.
type vectorPlan struct {
	pending    int64
	hasPending bool
	skipped    int64
	generation uint64
}

func (w *Worker) planVectors(ctx context.Context, v domain.Vault) vectorPlan {
	plan := vectorPlan{generation: w.skips.getGeneration()}
	plan.skipped = w.skips.getSkippedChunks()
	if w.vectorCounts == nil {
		plan.hasPending = true
		return plan
	}
	target, embedded, err := w.vectorCounts(ctx, v)
	if err != nil {
		plan.hasPending = true
		return plan
	}
	plan.skipped = min(plan.skipped, max(0, target-embedded))
	plan.pending = max(0, target-embedded-plan.skipped)
	plan.hasPending = plan.pending > 0
	return plan
}

func (plan vectorPlan) getTotal(res vaultreading.EmbedResult) int64 {
	if plan.pending == 0 {
		return 0
	}
	found := max(0, int64(res.Skipped)-plan.skipped)
	return max(int64(res.Embedded), plan.pending-found)
}

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

func (p *pass) makeVectors(ctx, items context.Context, plan vectorPlan) bool {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(vaultreading.Task{ID: MakingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := p.worker.openReading(ctx, p.v, false)
	if err != nil {
		failed(err)
		return false
	}
	// Progress is reported for the source currently open. The task enters the
	// list when the first source opens, and a progress share is reported once a
	// vector has been created, counted against the total pending.
	items, cancel := newTaskCancel(items)
	defer cancel.stop()
	var current currentSources
	making.OnVectorsProgress = func(res vaultreading.EmbedResult) {
		current.set(getEmbeddingSources(res))
		pos := vaultreading.Task{ID: MakingVectors, Title: "Indexing", About: res.CurrentSource, Cancel: cancel.request}
		if res.Embedded > 0 {
			pos.Count, pos.Total = int64(res.Embedded), plan.getTotal(res)
		}
		events.ReportTask(pos)
	}

	switch res, err := making.MakeVectors(items, p.v); {
	case err == nil:
		p.worker.skips.setSkippedChunks(res.Skipped, plan.generation)
		events.FinishTask(MakingVectors)
		return true
	case errors.Is(err, context.Canceled):
		// The source a person cancelled gets no vectors until it changes; the
		// next pass embeds the others.
		if cancel.isRequested.Load() {
			for _, path := range current.get() {
				p.worker.skips.addVectors(path)
			}
		}
		events.FinishTask(MakingVectors)
	default:
		// A vault missing vectors is searched by its text alone.
		failed(err)
	}
	return false
}
