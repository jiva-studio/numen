package index

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/snapshot"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

type memorySnapshots struct {
	mu      sync.Mutex
	kept    *ports.Snapshot
	saves   int
	failure error
}

func (m *memorySnapshots) Load(context.Context) (ports.Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.kept == nil {
		return ports.Snapshot{}, false
	}
	return *m.kept, true
}

func (m *memorySnapshots) Save(snapshot ports.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failure != nil {
		return m.failure
	}
	m.kept = &snapshot
	m.saves++
	return nil
}

func (m *memorySnapshots) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saves
}

type timers struct {
	mu      sync.Mutex
	pending []func()
}

func (c *timers) after(_ time.Duration, fn func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending = append(c.pending, fn)
	at := len(c.pending) - 1
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.pending[at] = nil
	}
}

func (c *timers) fire() int {
	c.mu.Lock()
	pending := c.pending
	c.pending = nil
	c.mu.Unlock()
	fired := 0
	for _, fn := range pending {
		if fn != nil {
			fn()
			fired++
		}
	}
	return fired
}

func persisted(vault *hosttest.Vault, snapshots *memorySnapshots, clock *timers) (*Index, *reader) {
	files := newReader(vault)
	return NewPersistent(files, vault, vault, todo.ParseNote, Persistence{Snapshots: snapshots, Feed: vault, After: clock.after, Cursors: snapshot.NewCursorStore, Log: new(recordingLog)}), files
}

type recordingLog struct {
	mu       sync.Mutex
	messages []string
}

func (l *recordingLog) Error(message string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, message+": "+err.Error())
}

func (l *recordingLog) reported() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.messages)
}

func flush(t *testing.T, one *Index) {
	t.Helper()
	if err := one.Flush(t.Context()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
}

func start(t *testing.T, one *Index) {
	t.Helper()
	if err := one.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func listed(one *Index) []string {
	return texts(one.List(today, filter.Filter{}, 1000))
}

func TestAWarmStartReadsNoUnchangedNoteAndAnswersWithTheSameTasksAndFields(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one +home @phone\n")
	vault.WriteFile("b.md", "- [ ] two +work\n")
	snapshots, clock := new(memorySnapshots), new(timers)
	first, _ := persisted(vault, snapshots, clock)
	start(t, first)
	flush(t, first)

	second, files := persisted(vault, snapshots, clock)
	start(t, second)

	if got := files.readsOf("a.md") + files.readsOf("b.md"); got != 0 {
		t.Errorf("reads on a warm start = %d, want 0", got)
	}
	if got, want := listed(second), listed(first); !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
	if got, want := formatCounts(second.ListFields().Projects), []string{"home=1", "work=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("projects = %v, want %v", got, want)
	}
	if !second.GetState().IsReady {
		t.Error("the index is not ready after a warm start")
	}
}

func TestAWarmStartReadsOnlyTheNotesChangedAddedOrRemovedWhileTheAppWasClosed(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("keep.md", "- [ ] kept\n")
	vault.WriteFile("edit.md", "- [ ] before\n")
	vault.WriteFile("gone.md", "- [ ] gone +old\n")
	snapshots, clock := new(memorySnapshots), new(timers)
	first, _ := persisted(vault, snapshots, clock)
	start(t, first)
	flush(t, first)

	vault.WriteFile("edit.md", "- [ ] after\n")
	vault.WriteFile("new.md", "- [ ] fresh +new\n")
	vault.Remove("gone.md")
	second, files := persisted(vault, snapshots, clock)
	start(t, second)

	if got := files.readsOf("keep.md"); got != 0 {
		t.Errorf("reads of the unchanged note = %d, want 0", got)
	}
	if files.readsOf("edit.md") != 1 || files.readsOf("new.md") != 1 || files.readsOf("gone.md") != 0 {
		t.Errorf("reads = edit %d, new %d, gone %d", files.readsOf("edit.md"), files.readsOf("new.md"), files.readsOf("gone.md"))
	}
	want := []string{"edit.md: after", "keep.md: kept", "new.md: fresh"}
	if got := listed(second); !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
	if got, want := formatCounts(second.ListFields().Projects), []string{"new=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("projects = %v, want %v", got, want)
	}
}

func TestAStartWithNoSnapshotReadsEveryNote(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one\n")
	second, files := persisted(vault, new(memorySnapshots), new(timers))

	start(t, second)

	if files.readsOf("a.md") != 1 || len(listed(second)) != 1 {
		t.Errorf("reads = %d, tasks = %v", files.readsOf("a.md"), listed(second))
	}
}

func TestAStartWhenTheFeedCannotContinueFromTheSnapshotReadsEveryNoteAgain(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one\n")
	snapshots, clock := new(memorySnapshots), new(timers)
	first, _ := persisted(vault, snapshots, clock)
	start(t, first)
	flush(t, first)
	vault.ResetFeed()

	second, files := persisted(vault, snapshots, clock)
	start(t, second)

	if files.readsOf("a.md") != 1 || len(listed(second)) != 1 {
		t.Errorf("reads = %d, tasks = %v", files.readsOf("a.md"), listed(second))
	}
}

func TestPersist_WritesOnceAfterTheDelayAndOnlyOnceReady(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one\n")
	snapshots, clock := new(memorySnapshots), new(timers)
	one, _ := persisted(vault, snapshots, clock)
	start(t, one)

	vault.WriteFile("a.md", "- [ ] two\n")
	vault.WriteFile("b.md", "- [ ] three\n")
	if err := one.Refresh(t.Context(), []string{"a.md", "b.md"}, nil); err != nil {
		t.Fatal(err)
	}
	vault.WriteFile("c.md", "- [ ] four\n")
	if err := one.Refresh(t.Context(), []string{"c.md"}, nil); err != nil {
		t.Fatal(err)
	}

	if got := snapshots.count(); got != 0 {
		t.Fatalf("saves before the delay = %d, want 0", got)
	}
	if fired := clock.fire(); fired != 1 {
		t.Fatalf("timers fired = %d, want 1", fired)
	}
	if got := snapshots.count(); got != 1 {
		t.Errorf("saves after the delay = %d, want 1", got)
	}
	if snapshots.kept == nil || len(snapshots.kept.Notes) != 3 {
		t.Errorf("snapshot = %+v, want three notes", snapshots.kept)
	}
}

func TestPersist_ReportsAFailedSaveAfterTheDelayToTheLog(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one\n")
	snapshots, clock, log := &memorySnapshots{failure: errors.New("disk full")}, new(timers), new(recordingLog)
	one := NewPersistent(newReader(vault), vault, vault, todo.ParseNote, Persistence{
		Snapshots: snapshots, Feed: vault, After: clock.after, Cursors: snapshot.NewCursorStore, Log: log,
	})
	start(t, one)
	vault.WriteFile("a.md", "- [ ] two\n")
	if err := one.Refresh(t.Context(), []string{"a.md"}, nil); err != nil {
		t.Fatal(err)
	}

	clock.fire()

	if got := log.reported(); !reflect.DeepEqual(got, []string{"save the task snapshot: disk full"}) {
		t.Errorf("reported = %v, want the failed save", got)
	}
}

func TestPersist_FlushReturnsAFailedSave(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "- [ ] one\n")
	failure := errors.New("disk full")
	one, _ := persisted(vault, &memorySnapshots{failure: failure}, new(timers))
	start(t, one)

	err := one.Flush(t.Context())

	if !errors.Is(err, failure) {
		t.Errorf("Flush = %v, want the failed save", err)
	}
}
