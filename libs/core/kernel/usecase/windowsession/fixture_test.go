package windowsession

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func garden() domain.Vault {
	return domain.Vault{ID: "v-garden", Name: "Garden", Path: "/vaults/Garden"}
}

func studio() domain.Vault {
	return domain.Vault{ID: "v-studio", Name: "Studio", Path: "/vaults/Studio"}
}

// pageSet is the pages of a window: they write at once unless one holds an unanswered question.
type pageSet struct {
	mu        sync.Mutex
	isAsking  bool
	flushes   int
	canceled  int
	log       *eventLog
	hold      *heldFlush
	releasing *heldFlush
}

// heldFlush is a flush request that waits inside the pages until it is released.
type heldFlush struct {
	entered chan struct{}
	gate    chan struct{}
}

func (p *pageSet) RequestFlush(context.Context) bool {
	p.mu.Lock()
	p.flushes++
	p.log.add("flush")
	hold := p.hold
	p.hold = nil
	isAsking := p.isAsking
	p.mu.Unlock()
	if hold != nil {
		close(hold.entered)
		<-hold.gate
	}
	return !isAsking
}

func (p *pageSet) holdNextFlush() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hold = &heldFlush{entered: make(chan struct{}), gate: make(chan struct{})}
	p.releasing = p.hold
}

func (p *pageSet) waitUntilHeld(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	hold := p.releasing
	p.mu.Unlock()
	select {
	case <-hold.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("no flush was requested from the pages")
	}
}

func (p *pageSet) release() {
	p.mu.Lock()
	hold := p.releasing
	p.mu.Unlock()
	close(hold.gate)
}

func (p *pageSet) CancelFlush() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.canceled++
}

func (p *pageSet) IsQuestionOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isAsking
}

// writeSet counts the writes of a window like the sealed door of a vault does.
type writeSet struct {
	mu       sync.Mutex
	isSealed bool
	reopened int
}

func (w *writeSet) BeginWrite() bool { return true }

func (w *writeSet) FinishWrite() {}

func (w *writeSet) Seal() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isSealed = true
	idle := make(chan struct{})
	close(idle)
	return idle
}

func (w *writeSet) Open() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isSealed, w.reopened = false, w.reopened+1
}

type taskList struct {
	mu    sync.Mutex
	tasks map[string]work.Task
}

func (l *taskList) AddTask(task work.Task) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.tasks == nil {
		l.tasks = map[string]work.Task{}
	}
	l.tasks[task.ID] = task
}

func (l *taskList) RemoveTask(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tasks, id)
}

func (l *taskList) has(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.tasks[id]
	return ok
}

type changeLog struct {
	mu      sync.Mutex
	changes []window.Change
}

func (c *changeLog) Publish(change window.Change) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changes = append(c.changes, change)
}

func (c *changeLog) all() []window.Change {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]window.Change(nil), c.changes...)
}

type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *eventLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

// knownVaults is the list the vaults of the tests are on.
type knownVaults struct{}

func (knownVaults) FindVault(nameOrPath string) (domain.Vault, bool, error) {
	for _, v := range []domain.Vault{garden(), studio()} {
		if nameOrPath == string(v.ID) || nameOrPath == v.Name || nameOrPath == v.Path {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}

// identity is the identity of the folders of the tests: each carries the identity of its vault, unless mismatched says the folder cannot be read.
type identity struct{ mismatched error }

func (identity) GetName(root string) string { return root }

func (i identity) CheckReadable(string) error { return i.mismatched }

func (identity) GetVaultID(root string) (domain.VaultID, bool, error) {
	v, isFound, err := knownVaults{}.FindVault(root)
	return v.ID, isFound, err
}

func (identity) EnsureIdentity(string, time.Time) (domain.VaultID, error) { return "", nil }

type openRecord struct{ log *eventLog }

func (r openRecord) RecordOpened(id domain.VaultID) error {
	r.log.add("recorded " + string(id))
	return nil
}

// runs is the passes started for vaults; each one runs until its context ends.
type runs struct {
	mu       sync.Mutex
	started  []domain.Vault
	rebuilds []bool
	report   []Reporters
	retired  int
	log      *eventLog
}

func (r *runs) Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) StartedPasses {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = append(r.started, v)
	r.rebuilds = append(r.rebuilds, rebuild)
	r.report = append(r.report, report)
	r.log.add("start " + string(v.ID))
	return StartedPasses{
		Read: func(context.Context, func()) (vaultreading.ScanResult, error) { return vaultreading.ScanResult{}, nil },
		Run:  func(ctx context.Context) { <-ctx.Done() },
		Retire: func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.retired++
			r.log.add("retire " + string(v.ID))
		},
	}
}

func (r *runs) vaults() []domain.Vault {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Vault(nil), r.started...)
}

type wakeLog struct {
	mu      sync.Mutex
	notes   int
	sources int
	skipped [][]string
}

func (w *wakeLog) WakeNotes()   { w.mu.Lock(); w.notes++; w.mu.Unlock() }
func (w *wakeLog) WakeSources() { w.mu.Lock(); w.sources++; w.mu.Unlock() }
func (w *wakeLog) ClearSkips(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.skipped = append(w.skipped, paths)
}

func (w *wakeLog) counts() (notes, sources int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.notes, w.sources
}

type door struct{ log *eventLog }

func (d door) Close() { d.log.add("door closed") }

type tabFront struct {
	mu  sync.Mutex
	ids []string
}

func (f *tabFront) Publish(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ids = append(f.ids, id)
}

// Deps is what the scenarios of a test window reach the world through.
type Deps struct {
	Window   WindowPorts
	Vault    VaultPorts
	Index    IndexPorts
	Shutdown ShutdownPorts

	Out           io.Writer
	EmbedderError error
	Closing       <-chan struct{}
	IsRebuilding  bool
	TabBound      time.Duration
}

type WindowPorts struct {
	Pages    Pages
	Writes   Writes
	Tasks    windowport.Tasks
	Changes  Changes
	TabFront TabFront
}

type VaultPorts struct {
	Identity vaultport.VaultIdentity
	Recorded vaultport.VaultRecorder
	Passes   VaultPasses
	Texts    TextVersions
}

type IndexPorts struct {
	Wake       IndexWake
	NewSession func(domain.Vault) IndexSession
	Triggers   TriggerSource
	Model      ModelSource
}

type ShutdownPorts struct {
	Door    Door
	Closers []CloseFunc
}

// UseCases holds what a test drives: vault switching under Switching, shutdown under Closing. It keeps the
// guard and identity check so a test can build another use case over the same guard.
type UseCases struct {
	Switching struct {
		Show       ShowVault
		OpenPlugin OpenPluginVault
		Rebuild    RebuildSession
	}
	Closing struct {
		Flush FlushPending
		Close CloseWindow
	}
	guard    *Guard
	identity CheckIdentity
	Start    StartWindow
	ShowTab  ShowTab
}

// NewUseCases builds the use cases of the window the way the container does.
func NewUseCases(rt *Runtime, tabs *TabReports, deps Deps) *UseCases {
	diag := NewDiagnostics(deps.Out, deps.Window.Tasks)
	runner := NewPassRunner(PassRunnerConfig{
		Failures: rt, Diag: diag, Closing: deps.Closing, NewSession: deps.Index.NewSession,
		Reporter: NewChangeReporter(deps.Window.Changes, deps.Index.Wake, deps.Vault.Texts, diag),
		Passes:   deps.Vault.Passes, Triggers: deps.Index.Triggers,
	})
	session := NewSession(rt, deps.Vault.Recorded, diag, deps.EmbedderError, runner)
	guard := &Guard{}
	flush := NewPageFlush(deps.Window.Pages, deps.Window.Writes)
	var s UseCases
	s.Switching.Rebuild = NewRebuildSession(guard, session)
	s.Switching.Show = NewShowVault(guard, session, NewCheckIdentity(deps.Vault.Identity), flush, deps.Window.Changes)
	s.Switching.OpenPlugin = NewOpenPluginVault(
		guard, knownVaults{}, NewCheckIdentity(deps.Vault.Identity), deps.Window.Pages,
		switchFunc(func(context.Context, domain.Vault, *PluginOpen) error { return errNoVaultSwitch }),
	)
	s.guard, s.identity = guard, NewCheckIdentity(deps.Vault.Identity)
	s.Closing.Flush = NewFlushPending(guard, flush)
	s.Closing.Close = NewCloseWindow(deps.Shutdown.Door, session, deps.Shutdown.Closers...)
	follow := NewModelFollow(deps.Index.Model, deps.Index.Wake, rt, s.Switching.Rebuild)
	s.Start = NewStartWindow(diag, deps.EmbedderError, follow, session, deps.IsRebuilding)
	s.ShowTab = NewShowTab(tabs, deps.Window.TabFront, deps.TabBound)
	return &s
}

// window is one window over fakes of every port, with no vault open.
type testWindow struct {
	rt       *Runtime
	tabs     *TabReports
	useCases *UseCases
	pages    *pageSet
	writes   *writeSet
	tasks    *taskList
	changes  *changeLog
	passes   *runs
	wake     *wakeLog
	front    *tabFront
	log      *eventLog
}

func newTestWindow(t *testing.T, edit func(*Deps)) *testWindow {
	t.Helper()
	return newLoggedWindow(t, &eventLog{}, edit)
}

func newLoggedWindow(t *testing.T, log *eventLog, edit func(*Deps)) *testWindow {
	t.Helper()
	w := &testWindow{
		rt:      &Runtime{},
		tabs:    &TabReports{},
		pages:   &pageSet{log: log},
		writes:  &writeSet{},
		tasks:   &taskList{},
		changes: &changeLog{},
		passes:  &runs{log: log},
		wake:    &wakeLog{},
		front:   &tabFront{},
		log:     log,
	}
	deps := Deps{
		Window: WindowPorts{Pages: w.pages, Writes: w.writes, Tasks: w.tasks, Changes: w.changes, TabFront: w.front},
		Vault:  VaultPorts{Identity: identity{}, Recorded: openRecord{log: log}, Passes: w.passes},
		Index: IndexPorts{
			Wake: w.wake,
			NewSession: func(domain.Vault) IndexSession {
				return IndexSession{
					Run:                   func(ctx context.Context, _ bool, _ ReadNotes) { <-ctx.Done() },
					ReportVersionsChanged: func(domain.Vault, string) {},
				}
			},
		},
		Shutdown: ShutdownPorts{Door: door{log: log}},
		Out:      io.Discard,
	}
	if edit != nil {
		edit(&deps)
	}
	w.useCases = NewUseCases(w.rt, w.tabs, deps)
	t.Cleanup(func() {
		if err := w.useCases.Closing.Close.Execute(t.Context()); err != nil {
			t.Log(err)
		}
	})
	return w
}

func (w *testWindow) open(t *testing.T, v domain.Vault) {
	t.Helper()
	w.useCases.Start.Execute(t.Context(), v)
}
