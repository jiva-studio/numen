package indexing

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

const taskIndexingTexts = "Indexing file texts"

// QuietPeriod is how long the vault has to have been still before the notes
// written into it are embedded.
const QuietPeriod = 8 * time.Second

// Events receives the progress and results of indexing a vault.
type Events interface {
	// ReportTask puts one piece of work in the list of what is being done, and
	// FinishTask takes it out.
	ReportTask(offset work.Task)
	FinishTask(id string)
	// ReportError reports why a pass could not run.
	ReportError(err error)
	// ReportNotes reports that the notes of a vault were read.
	ReportNotes(v domain.Vault, notes int)
	// ReportTexts reports how many file texts were cut.
	ReportTexts(v domain.Vault, res vaultreading.IndexResult)
}

// OpenReading is the three passes over a vault, for the search source given.
type OpenReading func(
	ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
) (ReadWholeVault, error)

// ReadNotes is the first walk over a vault. It answers how many notes it read.
type ReadNotes func(ctx context.Context, during func()) (int, error)

// GetVectorCounts answers how many chunks the vault holds that can carry a vector, and how
// many carry one.
type GetVectorCounts func(ctx context.Context, v domain.Vault) (held, embedded int64, err error)

// SearchSourceReady is notified that the search source has indexed every chunk of a vault, but those of the
// sources whose indexing a person canceled: those stay out until they change. It is notified after every pass
// that leaves no other chunk pending, so it is notified more than once for a vault.
type SearchSourceReady func(ctx context.Context, v domain.Vault)

// Config is what a worker is made from.
type Config struct {
	Reading           OpenReading
	Source            searchsourceport.Source
	VectorCounts      GetVectorCounts
	SearchSourceReady SearchSourceReady
	Events            Events
	Quiet             time.Duration
	After             func(time.Duration) <-chan time.Time
}

// Worker keeps the index of a vault in sync with its content, one pass at a
// time, and never while another pass is running.
// After the first pass, work arrives as a file with text, found and cut before
// anything is embedded; a note, which arrives already cut and needs only its
// vectors; and a file a plugin extended with more text. A search source that has not described itself is
// not indexed for yet: the change of its description starts the pass that indexes for it.
type Worker struct {
	reading           OpenReading
	source            searchsourceport.Source
	vectorCounts      GetVectorCounts
	searchSourceReady SearchSourceReady
	events            Events
	debounce          time.Duration
	after             func(time.Duration) <-chan time.Time
	wakes             *channelWakes
	skips             *vaultreading.Skips
}

func NewWorker(cfg Config) *Worker {
	return &Worker{
		reading:           cfg.Reading,
		source:            cfg.Source,
		vectorCounts:      cfg.VectorCounts,
		searchSourceReady: cfg.SearchSourceReady,
		events:            cfg.Events,
		debounce:          cfg.Quiet,
		after:             cfg.After,
		wakes:             newChannelWakes(),
		skips:             vaultreading.NewSkips(),
	}
}

// WakeSources leaves one pass over the file texts waiting.
func (w *Worker) WakeSources() { w.wakes.wakeSources() }

// WakeNotes leaves one pass over the notes' vectors waiting. Several changes at
// once are one pass.
func (w *Worker) WakeNotes() { w.wakes.wakeNotes() }

// ClearSkips indexes again the sources at the paths given whose indexing a
// person canceled. The vault calls it for every path it reports changed.
func (w *Worker) ClearSkips(paths []string) { w.skips.Clear(paths) }

func (w *Worker) openReading(
	ctx context.Context, v domain.Vault, rebuild bool,
) (ReadWholeVault, error) {
	return w.reading(ctx, w.source, v, rebuild)
}

func (w *Worker) hasSource() bool { return w.source != nil && w.source.Describe().IsDescribed() }

func (w *Worker) waitForQuiet() <-chan time.Time { return w.after(w.debounce) }

func (w *Worker) wakeRead() { w.wakes.wakeRead() }

func (w *Worker) notifySearchSourceReady(ctx context.Context, v domain.Vault) {
	if w.searchSourceReady == nil || ctx.Err() != nil {
		return
	}
	w.searchSourceReady(ctx, v)
}

func (w *Worker) newPass(s *Session, v domain.Vault, rebuild bool) *pass {
	return &pass{env: w, events: w.events, waits: w.wakes, skips: w.skips, session: s, v: v, isRebuild: rebuild}
}

// wakeWaits waits for the signals that something arrived for a pass to do.
type wakeWaits interface {
	sourcesWoken() <-chan struct{}
	notesWoken() <-chan struct{}
	readWakeChannel() <-chan struct{}
	// stopOnWake returns a context that ends when any of the signals arrives, and leaves that signal held.
	stopOnWake(ctx context.Context) (context.Context, func())
}

// channelWakes are the signals that something arrived for a pass to do. Each is held until a pass takes it,
// and several of the same kind are one.
type channelWakes struct {
	sources chan struct{}
	notes   chan struct{}
	read    chan struct{}
}

func newChannelWakes() *channelWakes {
	return &channelWakes{sources: make(chan struct{}, 1), notes: make(chan struct{}, 1), read: make(chan struct{}, 1)}
}

func (c *channelWakes) wakeSources() { signal(c.sources) }

func (c *channelWakes) wakeNotes() { signal(c.notes) }

func (c *channelWakes) wakeRead() { signal(c.read) }

func (c *channelWakes) sourcesWoken() <-chan struct{} { return c.sources }

func (c *channelWakes) notesWoken() <-chan struct{} { return c.notes }

func (c *channelWakes) readWakeChannel() <-chan struct{} { return c.read }

func (c *channelWakes) stopOnWake(ctx context.Context) (context.Context, func()) {
	underCtx, end := context.WithCancel(ctx)
	var watchers sync.WaitGroup
	for _, wake := range []chan struct{}{c.read, c.sources, c.notes} {
		watchers.Go(func() {
			select {
			case <-underCtx.Done():
			case <-wake:
				signal(wake)
				end()
			}
		})
	}
	return underCtx, func() {
		end()
		watchers.Wait()
	}
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

// sessionHost is the worker a session runs its passes for.
type sessionHost interface {
	wakeRead()
	newPass(s *Session, v domain.Vault, rebuild bool) *pass
}

// Session is the work of one vault: the sources a recognition has written more
// of than their chunks account for, and the passes over the vault.
type Session struct {
	host sessionHost

	mu      sync.Mutex
	pending map[string]domain.Vault
}

// NewSession is the work of a vault that has not been started.
func (w *Worker) NewSession() *Session {
	return &Session{host: w}
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
	s.host.wakeRead()
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
	s.host.newPass(s, v, rebuild).run(ctx, read)
}
