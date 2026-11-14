package index

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

const today = "2026-11-14"

// reader is the vault of the test with the reads counted, and optionally held until released.
type reader struct {
	*hosttest.Vault
	mu         sync.Mutex
	reads      map[string]int
	inFlight   int
	mostAtOnce int
	entered    chan string
	release    chan struct{}
}

func newReader(vault *hosttest.Vault) *reader {
	return &reader{Vault: vault, reads: map[string]int{}}
}

func (r *reader) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	r.mu.Lock()
	r.reads[path]++
	r.inFlight++
	r.mostAtOnce = max(r.mostAtOnce, r.inFlight)
	entered, release := r.entered, r.release
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.inFlight--
		r.mu.Unlock()
	}()
	if entered != nil {
		entered <- path
		<-release
	}
	for range 50 {
		runtime.Gosched()
	}
	return r.Vault.Read(ctx, path)
}

func (r *reader) readsOf(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reads[path]
}

// parser is todo.ParseNote with the calls counted.
type parser struct {
	mu    sync.Mutex
	calls int
}

func (p *parser) parse(raw []byte) ([]todo.Task, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	return todo.ParseNote(raw)
}

func (p *parser) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type fixture struct {
	vault  *hosttest.Vault
	reader *reader
	parser *parser
	index  *Index
}

func newFixture(notes map[string]string) *fixture {
	vault := hosttest.New("notes")
	for path, text := range notes {
		vault.File(path, text)
	}
	files := newReader(vault)
	counted := new(parser)
	return &fixture{vault: vault, reader: files, parser: counted, index: New(files, vault, counted.parse)}
}

func (f *fixture) scan(t *testing.T) {
	t.Helper()
	if err := f.index.Scan(t.Context()); err != nil {
		t.Fatalf("Scan: %v", err)
	}
}

func (f *fixture) refresh(t *testing.T, paths ...string) {
	t.Helper()
	if err := f.index.Refresh(t.Context(), paths, nil); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
}

func texts(listing Listing) []string {
	out := []string{}
	for _, task := range listing.Tasks {
		out = append(out, task.Path+": "+task.Text)
	}
	return out
}

func (f *fixture) list(filters ...filter.Filter) Listing {
	chosen := filter.Filter{}
	if len(filters) > 0 {
		chosen = filters[0]
	}
	return f.index.List(today, chosen, 1000)
}

func TestScanIndexesTheTasksOfEveryPlainNoteWithItsPath(t *testing.T) {
	f := newFixture(map[string]string{
		"Daily.md":         "- [ ] call mom\n",
		"Projects/Home.md": "intro\n- [ ] fix the tap due:2026-11-20\n",
		"Empty.md":         "nothing to do",
	})

	f.scan(t)

	want := []string{"Projects/Home.md: fix the tap", "Daily.md: call mom"}
	if got := texts(f.list()); !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
	if tasks := f.list().Tasks; len(tasks) == 0 || tasks[0].Line != 2 || tasks[0].Due != "2026-11-20" {
		t.Errorf("tasks = %+v, want the first at line 2 due 2026-11-20", tasks)
	}
}

func TestScanFollowedByAScanOfANewIndexGivesTheSameTasks(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] one\n- [x] two\n", "b.md": "- [ ] three +p @c (A)"})
	f.scan(t)

	again := New(f.reader, f.vault, todo.ParseNote)
	if err := again.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}

	all := filter.Filter{Status: []todo.Status{todo.StatusOpen, todo.StatusDone, todo.StatusOther}}
	if got, want := again.List(today, all, 100).Tasks, f.index.List(today, all, 100).Tasks; !reflect.DeepEqual(got, want) || len(got) != 3 {
		t.Errorf("second index = %+v, first index = %+v", got, want)
	}
}

func TestSummarizeCountsOpenTasksOverdueDueTodayAndWithNoDate(t *testing.T) {
	f := newFixture(map[string]string{
		"a.md": "- [ ] late due:2026-11-01\n- [ ] late too due:2026-11-13\n- [ ] now due:2026-11-14\n- [ ] later due:2026-11-30\n",
		"b.md": "- [ ] floating\n- [x] finished due:2026-11-01\n- [x] finished too\n- [ ] bad due:2026-02-30\n",
	})
	f.scan(t)

	got := f.index.Summarize(today)

	want := Summary{IsReady: true, OverdueCount: 2, TodayCount: 1, NoDateCount: 2, UnreadableCount: 0}
	if got != want {
		t.Errorf("summary = %+v, want %+v", got, want)
	}
}

func TestANoteWithAnUnclosedFrontmatterIsCountedAsUnreadableAndHasNoTasks(t *testing.T) {
	f := newFixture(map[string]string{"bad.md": "---\ntitle: x\n- [ ] hidden\n", "good.md": "- [ ] shown"})

	f.scan(t)

	if got := texts(f.list()); !reflect.DeepEqual(got, []string{"good.md: shown"}) {
		t.Errorf("tasks = %v", got)
	}
	if got := f.index.Summarize(today).UnreadableCount; got != 1 {
		t.Errorf("unreadable = %d, want 1", got)
	}
}

func TestANoteThatFailsToReadIsCountedAsUnreadableAndTheScanGoesOn(t *testing.T) {
	f := newFixture(map[string]string{"locked.md": "- [ ] secret", "open.md": "- [ ] visible"})
	f.vault.Refuse("locked.md", errors.New("permission denied"))

	f.scan(t)

	if got := texts(f.list()); !reflect.DeepEqual(got, []string{"open.md: visible"}) {
		t.Errorf("tasks = %v", got)
	}
	if summary := f.index.Summarize(today); summary.UnreadableCount != 1 || !summary.IsReady {
		t.Errorf("summary = %+v, want one unreadable note and ready", summary)
	}
}

func TestTheIndexIsNotReadyBeforeTheFirstScanEndsAndAnswersWithWhatItHas(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] one"})
	f.reader.entered, f.reader.release = make(chan string), make(chan struct{})
	finished := make(chan error, 1)
	go func() { finished <- f.index.Scan(context.Background()) }()

	<-f.reader.entered
	if summary := f.index.Summarize(today); summary.IsReady {
		t.Errorf("summary = %+v, want not ready while a note is being read", summary)
	}
	if got := f.list().Tasks; len(got) != 0 {
		t.Errorf("tasks = %v, want none yet", got)
	}
	if f.index.GetState().IsReady {
		t.Error("state is ready while a note is being read")
	}

	close(f.reader.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if !f.index.Summarize(today).IsReady || !f.index.GetState().IsReady {
		t.Error("the index is not ready after the scan")
	}
}

func TestScanReadsNoMoreThanEightNotesAtOnce(t *testing.T) {
	notes := map[string]string{}
	for n := range 64 {
		notes[fmt.Sprintf("n%02d.md", n)] = fmt.Sprintf("- [ ] task %d", n)
	}
	f := newFixture(notes)

	f.scan(t)

	if got := len(f.list().Tasks); got != 64 {
		t.Errorf("tasks = %d, want 64", got)
	}
	if f.reader.mostAtOnce > 8 {
		t.Errorf("%d notes were read at once, want at most 8", f.reader.mostAtOnce)
	}
}

func TestRefreshReadsAChangedNoteAgainAndNotAnUnchangedOne(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] first", "b.md": "- [ ] second"})
	f.scan(t)

	f.vault.File("a.md", "- [ ] first changed")
	f.refresh(t, "a.md", "b.md")

	if got := []int{f.reader.readsOf("a.md"), f.reader.readsOf("b.md")}; !reflect.DeepEqual(got, []int{2, 1}) {
		t.Errorf("reads of a.md and b.md = %v, want [2 1]", got)
	}
	if got := texts(f.list()); !reflect.DeepEqual(got, []string{"a.md: first changed", "b.md: second"}) {
		t.Errorf("tasks = %v", got)
	}
}

func TestANoteWhoseBytesAreTheSameButWhoseModTimeMovedIsNotParsedAgain(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] first"})
	f.scan(t)
	parsed, revision := f.parser.count(), f.index.GetState().Revision

	f.vault.File("a.md", "- [ ] first")
	f.refresh(t, "a.md")

	if got := f.parser.count(); got != parsed {
		t.Errorf("parses = %d, want %d", got, parsed)
	}
	if got := f.index.GetState().Revision; got != revision {
		t.Errorf("revision = %d, want %d", got, revision)
	}
}

func TestTheRevisionChangesWhenATaskListChangedAndOnlyThen(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] first", "b.md": "- [ ] second"})
	f.scan(t)
	before := f.index.GetState().Revision

	f.refresh(t, "b.md")
	if got := f.index.GetState().Revision; got != before {
		t.Errorf("revision after a refresh with no change = %d, want %d", got, before)
	}

	f.vault.File("b.md", "- [ ] second\n- [ ] third")
	f.refresh(t, "b.md")
	if got := f.index.GetState().Revision; got <= before {
		t.Errorf("revision after a change = %d, want more than %d", got, before)
	}
	if got := f.list().Revision; got != f.index.GetState().Revision {
		t.Errorf("listing revision = %d, state revision = %d", got, f.index.GetState().Revision)
	}
}

func TestARenameMovesTheTasksOfANoteAndADeleteDropsThem(t *testing.T) {
	f := newFixture(map[string]string{"old.md": "- [ ] moved", "gone.md": "- [ ] deleted", "stay.md": "- [ ] kept"})
	f.scan(t)

	f.vault.Remove("old.md")
	f.vault.File("new.md", "- [ ] moved")
	if err := f.index.Refresh(t.Context(), []string{"new.md"}, []Rename{{From: "old.md", To: "new.md"}}); err != nil {
		t.Fatal(err)
	}
	f.vault.Remove("gone.md")
	f.refresh(t, "gone.md")

	want := []string{"new.md: moved", "stay.md: kept"}
	if got := texts(f.list()); !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
}

func TestARefreshWithNoPathAndNoRenameChecksTheWholeVault(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] first", "b.md": "- [ ] second"})
	f.scan(t)

	f.vault.File("a.md", "- [ ] first changed")
	f.vault.File("c.md", "- [ ] third")
	f.vault.Remove("b.md")
	if err := f.index.Refresh(t.Context(), nil, nil); err != nil {
		t.Fatal(err)
	}

	want := []string{"a.md: first changed", "c.md: third"}
	if got := texts(f.list()); !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
	if got := f.reader.readsOf("c.md"); got != 1 {
		t.Errorf("reads of c.md = %d, want 1", got)
	}
}

func TestANoteInAnIgnoredFolderIsSkippedAndComesBackWhenTheFolderIsRemovedFromTheList(t *testing.T) {
	f := newFixture(map[string]string{
		"Daily.md":             "- [ ] daily",
		"templates/Weekly.md":  "- [ ] template",
		"archive/old/Year.md":  "- [ ] archived",
		"templates-two/Own.md": "- [ ] not ignored",
	})
	f.scan(t)

	if err := f.index.SetIgnoredFolders(t.Context(), " templates, archive/old/ ,,"); err != nil {
		t.Fatal(err)
	}
	if got, want := texts(f.list()), []string{"Daily.md: daily", "templates-two/Own.md: not ignored"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tasks with the folders ignored = %v, want %v", got, want)
	}

	if err := f.index.SetIgnoredFolders(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if got := len(f.list().Tasks); got != 4 {
		t.Errorf("tasks with nothing ignored = %d, want 4", got)
	}
}

func TestAFolderIgnoredBeforeTheScanIsNotRead(t *testing.T) {
	f := newFixture(map[string]string{"Daily.md": "- [ ] daily", "templates/Weekly.md": "- [ ] template"})
	if err := f.index.SetIgnoredFolders(t.Context(), "templates"); err != nil {
		t.Fatal(err)
	}

	f.scan(t)

	if got := f.reader.readsOf("templates/Weekly.md"); got != 0 {
		t.Errorf("reads of the ignored note = %d, want 0", got)
	}
}

func TestListCutsTheResultToTheLimitAndReportsTheTotal(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] 1\n- [ ] 2\n- [ ] 3\n- [ ] 4\n- [ ] 5"})
	f.scan(t)

	got := f.index.List(today, filter.Filter{}, 2)

	if len(got.Tasks) != 2 || got.Total != 5 {
		t.Errorf("tasks = %d, total = %d, want 2 and 5", len(got.Tasks), got.Total)
	}
}

func TestWatchSendsTheStateAtOnceAndAfterEachChangeOfTheIndex(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] first"})
	f.scan(t)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	states := make(chan State, 8)
	finished := make(chan error, 1)
	go func() {
		finished <- f.index.Watch(ctx, func(state State) error {
			states <- state
			return nil
		})
	}()
	next := func() State {
		t.Helper()
		select {
		case state := <-states:
			return state
		case <-time.After(10 * time.Second):
			t.Fatal("no state was sent")
			return State{}
		}
	}

	first := next()
	if !first.IsReady || first.Revision != f.index.GetState().Revision {
		t.Errorf("first state = %+v, want the current one", first)
	}

	f.vault.File("a.md", "- [ ] first\n- [ ] second")
	f.refresh(t, "a.md")
	if second := next(); second.Revision <= first.Revision {
		t.Errorf("second state = %+v, want a revision after %d", second, first.Revision)
	}

	stop()
	if err := <-finished; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Watch = %v", err)
	}
}
