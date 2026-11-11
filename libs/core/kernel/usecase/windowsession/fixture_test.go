package windowsession

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func garden() domain.Vault {
	return domain.Vault{ID: "v-garden", Name: "Garden", Path: "/vaults/Garden"}
}

func studio() domain.Vault {
	return domain.Vault{ID: "v-studio", Name: "Studio", Path: "/vaults/Studio"}
}

// pageSet is the pages of a window: they write at once unless one holds an unanswered question.
type pageSet struct {
	mu       sync.Mutex
	isAsking bool
	flushes  int
	canceled int
	log      *eventLog
}

func (p *pageSet) RequestFlush(context.Context) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.flushes++
	p.log.add("flush")
	return !p.isAsking
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

type identity struct{ mismatched error }

func (i identity) Execute(domain.Vault) error { return i.mismatched }

type openRecord struct{ log *eventLog }

func (r openRecord) Execute(id domain.VaultID) error {
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

func (r *runs) Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) Started {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = append(r.started, v)
	r.rebuilds = append(r.rebuilds, rebuild)
	r.report = append(r.report, report)
	r.log.add("start " + string(v.ID))
	return Started{
		Read: func(context.Context, func()) (int, error) { return 0, nil },
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

// window is one window over fakes of every port, with no vault open.
type testWindow struct {
	rt        *Runtime
	scenarios *Scenarios
	pages     *pageSet
	writes    *writeSet
	tasks     *taskList
	changes   *changeLog
	passes    *runs
	wake      *wakeLog
	front     *tabFront
	log       *eventLog
}

func newTestWindow(t *testing.T, edit func(*Deps)) *testWindow {
	t.Helper()
	return newLoggedWindow(t, &eventLog{}, edit)
}

func newLoggedWindow(t *testing.T, log *eventLog, edit func(*Deps)) *testWindow {
	t.Helper()
	w := &testWindow{
		rt:      &Runtime{},
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
			NewSession: func() IndexSession {
				return IndexSession{
					Run:                   func(ctx context.Context, _ domain.Vault, _ bool, _ ReadNotes) { <-ctx.Done() },
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
	w.scenarios = NewScenarios(w.rt, deps)
	t.Cleanup(func() {
		if err := w.scenarios.Closing.Close.Execute(t.Context()); err != nil {
			t.Log(err)
		}
	})
	return w
}

func (w *testWindow) open(t *testing.T, v domain.Vault) {
	t.Helper()
	w.scenarios.Start.Execute(t.Context(), v)
}
