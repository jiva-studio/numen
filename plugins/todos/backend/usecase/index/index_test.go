package index

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
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
		vault.WriteFile(path, text)
	}
	files := newReader(vault)
	counted := new(parser)
	return &fixture{vault: vault, reader: files, parser: counted, index: New(files, vault, vault, counted.parse)}
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

func texts(listing TaskList) []string {
	out := []string{}
	for _, task := range listing.Tasks {
		out = append(out, task.Path+": "+task.Text)
	}
	return out
}

func (f *fixture) list(filters ...filter.Filter) TaskList {
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

func TestAScanOfANewIndexReturnsTheSameTasks(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "- [ ] one\n- [x] two\n", "b.md": "- [ ] three +p @c (A)"})
	f.scan(t)

	again := New(f.reader, f.vault, f.vault, todo.ParseNote)
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
		"a.md": "- [ ] late due:2026-11-01\n- [ ] late too due:2026-11-13\n- [ ] now due:2026-11-14\n- [ ] soon due:2026-11-16\n- [ ] later due:2026-11-30\n",
		"b.md": "- [ ] floating\n- [x] finished due:2026-11-01\n- [x] finished too\n- [ ] bad due:2026-02-30\n",
	})
	f.scan(t)

	got := f.index.Summarize(today)

	want := Summary{IsReady: true, OverdueCount: 2, TodayCount: 1, NextSevenDaysCount: 1, NoDateCount: 2, UnreadableCount: 0}
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
	f.reader.entered, f.reader.release = make(chan string, 64), make(chan struct{})
	finished := make(chan error, 1)
	go func() { finished <- f.index.Scan(t.Context()) }()

	select {
	case <-f.reader.entered:
	case err := <-finished:
		t.Fatalf("the scan ended with %v before it read a note", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the scan read no note")
	}
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

	f.vault.WriteFile("a.md", "- [ ] first changed")
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
	if parsedOnce := f.parser.count(); parsedOnce != 1 {
		t.Fatalf("parses after the scan = %d, want 1", parsedOnce)
	}
	parsed, revision := f.parser.count(), f.index.GetState().Revision

	f.vault.WriteFile("a.md", "- [ ] first")
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

	f.vault.WriteFile("b.md", "- [ ] second\n- [ ] third")
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
	f.vault.WriteFile("new.md", "- [ ] moved")
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

	f.vault.WriteFile("a.md", "- [ ] first changed")
	f.vault.WriteFile("c.md", "- [ ] third")
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

func TestANoteInAnIgnoredFolderIsSkippedAndIsIndexedAgainWhenTheFolderIsRemovedFromTheList(t *testing.T) {
	f := newFixture(map[string]string{
		"Daily.md":             "- [ ] daily",
		"templates/Weekly.md":  "- [ ] template",
		"archive/old/Year.md":  "- [ ] archived",
		"templates-two/Own.md": "- [ ] not ignored",
	})
	f.scan(t)

	if err := f.index.SetIgnoredFolders(t.Context(), []string{" templates", "archive/old/ ", "", "/"}); err != nil {
		t.Fatal(err)
	}
	if got, want := texts(f.list()), []string{"Daily.md: daily", "templates-two/Own.md: not ignored"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tasks with the folders ignored = %v, want %v", got, want)
	}

	if err := f.index.SetIgnoredFolders(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got := len(f.list().Tasks); got != 4 {
		t.Errorf("tasks with nothing ignored = %d, want 4", got)
	}
}

func TestAFolderIgnoredBeforeTheScanIsNotRead(t *testing.T) {
	f := newFixture(map[string]string{"Daily.md": "- [ ] daily", "templates/Weekly.md": "- [ ] template"})
	if err := f.index.SetIgnoredFolders(t.Context(), []string{"templates"}); err != nil {
		t.Fatal(err)
	}

	f.scan(t)

	if got := f.reader.readsOf("templates/Weekly.md"); got != 0 {
		t.Errorf("reads of the ignored note = %d, want 0", got)
	}
}

func TestListLimitsTheResultAndReportsTheTotal(t *testing.T) {
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

	f.vault.WriteFile("a.md", "- [ ] first\n- [ ] second")
	f.refresh(t, "a.md")
	if second := next(); second.Revision <= first.Revision {
		t.Errorf("second state = %+v, want a revision after %d", second, first.Revision)
	}

	stop()
	if err := <-finished; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Watch = %v", err)
	}
}

func formatCounts(counts []FieldCount) []string {
	out := []string{}
	for _, one := range counts {
		out = append(out, fmt.Sprintf("%s=%d", one.Name, one.Count))
	}
	return out
}

func TestListFieldsCountsProjectsAndContextsAndFollowsAddChangeAndRemoveOfANote(t *testing.T) {
	f := newFixture(map[string]string{
		"a.md": "- [ ] one +home @phone\n- [ ] two +home +work\n",
		"b.md": "- [ ] three +work @phone @desk\n",
	})
	f.scan(t)

	got := f.index.ListFields()
	if want := []string{"home=2", "work=2"}; !reflect.DeepEqual(formatCounts(got.Projects), want) {
		t.Errorf("projects = %v, want %v", formatCounts(got.Projects), want)
	}
	if want := []string{"phone=2", "desk=1"}; !reflect.DeepEqual(formatCounts(got.Contexts), want) {
		t.Errorf("contexts = %v, want %v", formatCounts(got.Contexts), want)
	}

	f.vault.WriteFile("a.md", "- [ ] one +garden\n")
	f.vault.WriteFile("c.md", "- [ ] four +garden @phone\n")
	f.refresh(t, "a.md", "c.md")
	got = f.index.ListFields()
	if want := []string{"garden=2", "work=1"}; !reflect.DeepEqual(formatCounts(got.Projects), want) {
		t.Errorf("projects after change = %v, want %v", formatCounts(got.Projects), want)
	}
	if want := []string{"phone=2", "desk=1"}; !reflect.DeepEqual(formatCounts(got.Contexts), want) {
		t.Errorf("contexts after change = %v, want %v", formatCounts(got.Contexts), want)
	}

	f.vault.Remove("b.md")
	f.refresh(t, "b.md")
	got = f.index.ListFields()
	if want := []string{"garden=2"}; !reflect.DeepEqual(formatCounts(got.Projects), want) {
		t.Errorf("projects after remove = %v, want %v", formatCounts(got.Projects), want)
	}
	if want := []string{"phone=1"}; !reflect.DeepEqual(formatCounts(got.Contexts), want) {
		t.Errorf("contexts after remove = %v, want %v", formatCounts(got.Contexts), want)
	}
}

func TestListFieldsOrdersTiesByNameAndKeepsTheMostUsedTwoHundred(t *testing.T) {
	var body strings.Builder
	for n := range 205 {
		fmt.Fprintf(&body, "- [ ] task +p%03d\n", n)
	}
	body.WriteString("- [ ] again +p204\n")
	f := newFixture(map[string]string{"a.md": body.String()})
	f.scan(t)

	got := f.index.ListFields().Projects

	if len(got) != 200 {
		t.Fatalf("projects = %d, want 200", len(got))
	}
	if got[0].Name != "p204" || got[0].Count != 2 || got[1].Name != "p000" || got[199].Name != "p198" {
		t.Errorf("order = %v ... %v", got[:2], got[199])
	}
}

func TestListFieldsDropsTheCountsOfANoteThatIsRenamedAwayAndStaysTheSameWhenItMoves(t *testing.T) {
	f := newFixture(map[string]string{"old.md": "- [ ] moved +home"})
	f.scan(t)

	f.vault.Remove("old.md")
	f.vault.WriteFile("new.md", "- [ ] moved +home")
	if err := f.index.Refresh(t.Context(), []string{"new.md"}, []Rename{{From: "old.md", To: "new.md"}}); err != nil {
		t.Fatal(err)
	}

	if got, want := formatCounts(f.index.ListFields().Projects), []string{"home=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("projects = %v, want %v", got, want)
	}
}

func TestListPathsReturnsTheSortedNotesWithATaskTheFilterAccepts(t *testing.T) {
	f := newFixture(map[string]string{
		"b.md": "- [ ] one due:2026-11-10\n",
		"a.md": "- [ ] two due:2026-11-10\n- [ ] three\n",
		"c.md": "- [x] four\n",
		"d.md": "- [ ] five due:2026-12-01\n",
	})
	before := f.index.ListPaths("2026-11-14", filter.Filter{Due: filter.DueOverdue})
	f.scan(t)

	overdue := f.index.ListPaths("2026-11-14", filter.Filter{Due: filter.DueOverdue})
	done := f.index.ListPaths("2026-11-14", filter.Filter{Status: []todo.Status{todo.StatusDone}})

	if before.IsReady || len(before.Paths) != 0 {
		t.Errorf("before the scan = %+v, want no paths and not ready", before)
	}
	if !overdue.IsReady || !slices.Equal(overdue.Paths, []string{"a.md", "b.md"}) {
		t.Errorf("overdue = %+v, want a.md and b.md", overdue)
	}
	if !slices.Equal(done.Paths, []string{"c.md"}) {
		t.Errorf("done = %+v, want c.md", done)
	}
}

func TestTheGroupsOfTheSummaryAreDisjointAndMatchTheListsOfTheirFilters(t *testing.T) {
	f := newFixture(map[string]string{
		"a.md": "- [ ] yesterday due:2026-11-13\n- [ ] now due:2026-11-14\n- [ ] tomorrow due:2026-11-15\n- [ ] in six days due:2026-11-20\n",
		"b.md": "- [ ] in seven days due:2026-11-21\n- [ ] in eight days due:2026-11-22\n- [ ] floating\n- [x] finished due:2026-11-15\n- [x] finished today due:2026-11-14\n",
	})
	f.scan(t)

	groups := map[string]filter.DueFilter{
		"overdue":       filter.DueOverdue,
		"today":         filter.DueToday,
		"nextSevenDays": filter.DueNextSevenDays,
		"noDate":        filter.DueNone,
	}
	want := map[string][]string{
		"overdue":       {"a.md: yesterday"},
		"today":         {"a.md: now"},
		"nextSevenDays": {"a.md: tomorrow", "a.md: in six days"},
		"noDate":        {"b.md: floating"},
	}
	for name, due := range groups {
		got := texts(f.list(filter.Filter{Due: due}))
		if !reflect.DeepEqual(got, want[name]) {
			t.Errorf("%s = %v, want %v", name, got, want[name])
		}
	}
	got := f.index.Summarize(today)
	if got.OverdueCount != 1 || got.TodayCount != 1 || got.NextSevenDaysCount != 2 || got.NoDateCount != 1 {
		t.Errorf("summary = %+v, want overdue 1, today 1, next seven days 2, no date 1", got)
	}
}

func TestMatchPathsKeepsTheAskedOrderAndDropsNotesWithNoAcceptedTask(t *testing.T) {
	f := newFixture(map[string]string{
		"b.md": "- [ ] one due:2026-11-10\n",
		"a.md": "- [ ] two\n",
		"c.md": "- [x] four due:2026-11-10\n",
	})
	f.scan(t)

	got := f.index.MatchPaths("2026-11-14", filter.Filter{Due: filter.DueOverdue}, []string{"c.md", "b.md", "a.md", "gone.md", "b.md"})

	if want := []string{"b.md", "b.md"}; !got.IsReady || !slices.Equal(got.Paths, want) {
		t.Errorf("MatchPaths = %+v, want %v", got, want)
	}
}
