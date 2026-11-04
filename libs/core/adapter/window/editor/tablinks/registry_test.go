package tablinks

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

// timers is a Schedule that a test fires manually.
type timers struct {
	pending []*timer
}

type timer struct {
	after       time.Duration
	run         func()
	isCancelled bool
}

func (s *timers) schedule(d time.Duration, f func()) func() bool {
	one := &timer{after: d, run: f}
	s.pending = append(s.pending, one)
	return func() bool {
		was := !one.isCancelled
		one.isCancelled = true
		return was
	}
}

// fireAll runs every scheduled func that is not cancelled.
func (s *timers) fireAll() {
	for _, one := range s.pending {
		if !one.isCancelled {
			one.isCancelled = true
			one.run()
		}
	}
}

const grace = 30 * time.Second

func newTestRegistry() (*Registry, *timers) {
	clock := &timers{}
	r, err := NewRegistry(grace, clock.schedule)
	if err != nil {
		panic(err)
	}
	return r, clock
}

func plex(id, title string) Tab  { return Tab{ID: id, Kind: "plex", Title: title} }
func files(id, title string) Tab { return Tab{ID: id, Kind: "files", Title: title} }
func getTabIDs(s Snapshot) []string {
	out := make([]string, 0, len(s.Tabs))
	for _, one := range s.Tabs {
		out = append(out, one.ID)
	}
	return out
}

func getGroupTabs(s Snapshot) [][]string {
	out := make([][]string, 0, len(s.Groups))
	for _, one := range s.Groups {
		out = append(out, one.TabIDs)
	}
	return out
}

func TestLinkGroupsTabsOfTwoWindows(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "Orbits")})
	writeTabs(r, "window-b", []Tab{files("files:1", "Files")})

	r.Link("plex:1", "files:1")

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "files:1"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Groups[0].ColorIndex != 1 {
		t.Errorf("the first group has color %d, want 1", got.Groups[0].ColorIndex)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("files:1 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
}

func TestLinkIgnoresUnreportedTab(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "Orbits")})

	r.Link("plex:1", "plex:ghost")
	r.Link("plex:1", "plex:1")

	if groups := r.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestLinkAddsTabToGroupOfEitherSide(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C"), plex("plex:4", "D")})

	r.Link("plex:1", "plex:2")
	r.Link("plex:1", "plex:3")
	r.Link("plex:4", "plex:2")

	if want := [][]string{{"plex:1", "plex:2", "plex:3", "plex:4"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestLinkMergesTwoGroups(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-b", []Tab{plex("plex:3", "C"), plex("plex:4", "D")})
	r.Link("plex:1", "plex:2")
	r.Link("plex:3", "plex:4")
	if colors := []int{r.GetSnapshot().Groups[0].ColorIndex, r.GetSnapshot().Groups[1].ColorIndex}; !reflect.DeepEqual(colors, []int{1, 2}) {
		t.Fatalf("the two groups have colors %v, want [1 2]", colors)
	}

	r.Link("plex:2", "plex:3")

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2", "plex:3", "plex:4"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Groups[0].ColorIndex != 1 {
		t.Errorf("the merged group has color %d, want the source group's 1", got.Groups[0].ColorIndex)
	}
}

func TestUnlinkDeletesGroupWithOneTabLeft(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
	r.Link("plex:1", "plex:2")
	r.Link("plex:1", "plex:3")

	r.Unlink("plex:3")
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}

	r.Unlink("plex:1")
	if groups := r.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestWriteTabsUnlinksTabMissingFromReport(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	got := r.GetSnapshot()
	if len(got.Groups) != 0 {
		t.Errorf("groups are %+v, want none", got.Groups)
	}
	if want := []string{"plex:1"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestCloseOfLastWatchUnlinksWindowTabsAfterGracePeriod(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	watch := r.Watch("window-b")
	writeTabs(r, "window-b", []Tab{files("files:1", "Files")})
	r.Link("plex:1", "files:1")
	r.Link("plex:2", "files:1")

	watch.Close()
	if want := [][]string{{"plex:1", "files:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Fatalf("groups are %v before the grace period ended, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if want := []string{"plex:1", "plex:2"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestCloseKeepsTabsOfWindowWithAnotherWatch(t *testing.T) {
	r, _ := newTestRegistry()
	first := r.Watch("window-a")
	second := r.Watch("window-a")
	defer second.Close()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	first.Close()

	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("groups are %+v, want the one group", r.GetSnapshot().Groups)
	}
}

func TestDetachKeepsLinksWhenNewWindowReportsTab(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	r.Detach("plex:2")
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})
	if len(r.GetSnapshot().Groups) != 1 {
		t.Fatalf("the detached tab left its group before the grace period ended")
	}
	writeTabs(r, "window-b", []Tab{plex("plex:2", "B")})
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
	if clock.pending[0].after != grace {
		t.Errorf("the grace period is %v, want %v", clock.pending[0].after, grace)
	}
}

func TestDetachUnlinksUnreportedTabAfterGracePeriod(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	clock.fireAll()

	got := r.GetSnapshot()
	if len(got.Groups) != 0 {
		t.Errorf("groups are %+v, want none", got.Groups)
	}
	if want := []string{"plex:1"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestFailedDetachKeepsTabInItsWindow(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")

	clock.fireAll()

	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("a tab its window still reports left its group")
	}
}

func TestDetachKeepsLinksAfterOldWindowClosesUntilGracePeriodEnds(t *testing.T) {
	r, clock := newTestRegistry()
	watch := r.Watch("window-a")
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})
	writeTabs(r, "window-c", []Tab{plex("plex:3", "C")})
	r.Link("plex:1", "plex:3")
	r.Detach("plex:1")

	watch.Close()
	if len(r.GetSnapshot().Groups) != 1 {
		t.Fatalf("the detached tab left with its old window")
	}
	clock.fireAll()

	if len(r.GetSnapshot().Groups) != 0 {
		t.Errorf("groups are %+v, want none once the grace period ended", r.GetSnapshot().Groups)
	}
}

func TestWatchStartsFromCurrentState(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	watch := r.Watch("window-b")
	defer watch.Close()

	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(watch.Initial), want) {
		t.Errorf("the first snapshot has groups %v, want %v", getGroupTabs(watch.Initial), want)
	}
	if want := []string{"plex:1", "plex:2"}; !reflect.DeepEqual(getTabIDs(watch.Initial), want) {
		t.Errorf("the first snapshot has tabs %v, want %v", getTabIDs(watch.Initial), want)
	}
}

func TestEveryWatchReceivesEachChangeAndSync(t *testing.T) {
	r, _ := newTestRegistry()
	first, second := r.Watch("window-a"), r.Watch("window-b")
	defer first.Close()
	defer second.Close()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})
	writeTabs(r, "window-b", []Tab{files("files:1", "Files")})
	r.Link("plex:1", "files:1")

	if !r.SyncPath("files:1", "notes/Orbits.md", "Orbits") {
		t.Fatal("a linked tab was not synced")
	}

	for name, watch := range map[string]*Watch{"first": first, "second": second} {
		snapshot := readUntilSync(t, watch)
		if want := [][]string{{"plex:1", "files:1"}}; !reflect.DeepEqual(getGroupTabs(snapshot.links), want) {
			t.Errorf("%s watch: groups are %v, want %v", name, getGroupTabs(snapshot.links), want)
		}
		want := PathSync{GroupID: snapshot.links.Groups[0].ID, SourceTabID: "files:1", Path: "notes/Orbits.md", Title: "Orbits"}
		if snapshot.sync != want {
			t.Errorf("%s watch: sync is %+v, want %+v", name, snapshot.sync, want)
		}
	}
}

func TestSyncPathOfUnlinkedTabSendsNothing(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	if r.SyncPath("plex:1", "notes/Orbits.md", "") {
		t.Error("a tab in no group was synced")
	}
}

func TestCancelEndsEveryWatchOfTheWindow(t *testing.T) {
	r, _ := newTestRegistry()
	mine, other := r.Watch("window-a"), r.Watch("window-b")
	defer mine.Close()
	defer other.Close()

	r.Cancel("window-a")
	r.Cancel("window-a")

	select {
	case <-mine.Stop:
	default:
		t.Error("the cancelled watch is still open")
	}
	select {
	case <-other.Stop:
		t.Error("the watch of another window was cancelled")
	default:
	}
}

type received struct {
	links Snapshot
	sync  PathSync
}

// readUntilSync reads a watch up to its first sync and returns the sync and
// the last snapshot before it.
func readUntilSync(t *testing.T, watch *Watch) received {
	t.Helper()
	var out received
	for {
		select {
		case event := <-watch.Events:
			if event.Snapshot != nil {
				out.links = *event.Snapshot
			}
			if event.Sync != nil {
				out.sync = *event.Sync
				return out
			}
		case <-time.After(5 * time.Second):
			t.Fatal("no sync arrived")
		}
	}
}

// writeTabs writes the tabs of a window and opens a watch if the window has none.
func writeTabs(r *Registry, windowID string, tabs []Tab) {
	r.mu.Lock()
	isWatched := len(r.watches[windowID]) > 0
	r.mu.Unlock()
	if !isWatched {
		r.Watch(windowID)
	}
	r.WriteTabs(windowID, tabs)
}

func TestWriteTabsIgnoresAWindowWithNoWatch(t *testing.T) {
	r, _ := newTestRegistry()
	watch := r.Watch("window-a")
	watch.Close()

	r.WriteTabs("window-a", []Tab{plex("plex:1", "A")})
	r.WriteTabs("window-b", []Tab{plex("plex:2", "B")})

	if tabs := r.GetSnapshot().Tabs; len(tabs) != 0 {
		t.Errorf("tabs are %+v, want none", tabs)
	}
}

func TestWriteTabsKeepsTabOfAnotherLiveWindowOutsideDetach(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	writeTabs(r, "window-b", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})

	got := r.GetSnapshot()
	if got.Tabs[0].WindowID != "window-a" {
		t.Errorf("plex:1 is in window %q, want window-a", got.Tabs[0].WindowID)
	}
	writeTabs(r, "window-b", []Tab{plex("plex:2", "B")})
	if want := []string{"plex:1", "plex:2"}; !reflect.DeepEqual(getTabIDs(r.GetSnapshot()), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(r.GetSnapshot()), want)
	}
}

func TestDetachIgnoresStaleOldWindowReportAfterNewWindow(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")

	writeTabs(r, "window-b", []Tab{plex("plex:2", "B")})
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
}

func TestDetachSurvivesOldWindowReportBeforeNewWindow(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")

	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})
	writeTabs(r, "window-b", []Tab{plex("plex:2", "B")})
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
}

func TestSlowWatchReceivesLastSyncOfEachGroup(t *testing.T) {
	r, _ := newTestRegistry()
	watch := r.Watch("window-a")
	defer watch.Close()
	const groups = 20
	var tabs []Tab
	for index := range groups * 2 {
		tabs = append(tabs, plex("plex:"+strconv.Itoa(index), ""))
	}
	writeTabs(r, "window-a", tabs)
	for index := 0; index < groups*2; index += 2 {
		r.Link(tabs[index].ID, tabs[index+1].ID)
	}

	for index := 0; index < groups*2; index += 2 {
		r.SyncPath(tabs[index].ID, "first.md", "")
		r.SyncPath(tabs[index].ID, "last.md", "")
	}

	paths := map[string]string{}
	for len(paths) < groups {
		select {
		case event := <-watch.Events:
			if event.Sync != nil {
				paths[event.Sync.GroupID] = event.Sync.Path
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("syncs arrived for %d groups, want %d", len(paths), groups)
		}
	}
	for group, path := range paths {
		if path != "last.md" {
			t.Errorf("group %s last moved to %q, want last.md", group, path)
		}
	}
}

func TestNewRegistryRequiresASchedule(t *testing.T) {
	if _, err := NewRegistry(grace, nil); err == nil {
		t.Error("a registry with no schedule was made")
	}
}

func TestWriteTabsAfterReloadKeepsTabsOfTheNewWindow(t *testing.T) {
	r, _ := newTestRegistry()
	old := r.Watch("window-old")
	r.WriteTabs("window-old", []Tab{plex("plex:1", "A")})
	reloaded := r.Watch("window-new")
	defer reloaded.Close()

	r.WriteTabs("window-new", []Tab{plex("plex:1", "A")})
	old.Close()

	got := r.GetSnapshot()
	if want := []string{"plex:1"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Fatalf("tabs are %v, want %v", getTabIDs(got), want)
	}
	if got.Tabs[0].WindowID != "window-new" {
		t.Errorf("plex:1 is in window %q, want window-new", got.Tabs[0].WindowID)
	}
}

func TestRemovalOfReloadedWindowKeepsTabsOfTheNewWindow(t *testing.T) {
	r, clock := newTestRegistry()
	old := r.Watch("window-old")
	r.WriteTabs("window-old", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	old.Close()
	reloaded := r.Watch("window-new")
	defer reloaded.Close()

	r.WriteTabs("window-new", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	clock.fireAll()

	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestWatchDeliversSnapshotBeforeLaterSync(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
	r.Link("plex:1", "plex:2")
	watch := r.Watch("window-b")
	defer watch.Close()

	r.SyncPath("plex:1", "first.md", "")
	r.Link("plex:1", "plex:3")
	r.SyncPath("plex:1", "second.md", "")

	isLinkReceived := false
	for {
		select {
		case event := <-watch.Events:
			if event.Snapshot != nil && reflect.DeepEqual(getGroupTabs(*event.Snapshot), [][]string{{"plex:1", "plex:2", "plex:3"}}) {
				isLinkReceived = true
			}
			if event.Sync != nil && event.Sync.Path == "second.md" {
				if !isLinkReceived {
					t.Fatal("the sync to second.md arrived before the snapshot that adds plex:3")
				}
				return
			}
		case <-time.After(5 * time.Second):
			t.Fatal("no sync to second.md arrived")
		}
	}
}

func TestReconnectKeepsLinksOfTheWindow(t *testing.T) {
	r, clock := newTestRegistry()
	first := r.Watch("window-a")
	r.WriteTabs("window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	first.Close()
	second := r.Watch("window-a")
	defer second.Close()
	r.WriteTabs("window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	clock.fireAll()

	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestFailedDetachDropsTabClosedInItsWindow(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
	r.Link("plex:1", "plex:2")
	r.Link("plex:1", "plex:3")
	r.Detach("plex:2")

	r.CancelDetach("plex:2")
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:3", "C")})

	if want := []string{"plex:1", "plex:3"}; !reflect.DeepEqual(getTabIDs(r.GetSnapshot()), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(r.GetSnapshot()), want)
	}
}

func TestCancelDetachDropsTabItsWindowNoLongerReports(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	r.CancelDetach("plex:2")

	if len(r.GetSnapshot().Groups) != 0 {
		t.Errorf("groups are %+v, want none", r.GetSnapshot().Groups)
	}
}

// expiries is a Schedule backed by time.AfterFunc whose runs a test can wait for.
type expiries struct{ running sync.WaitGroup }

func (s *expiries) schedule(d time.Duration, f func()) func() bool {
	s.running.Add(1)
	timer := time.AfterFunc(d, func() {
		defer s.running.Done()
		f()
	})
	return func() bool {
		if !timer.Stop() {
			return false
		}
		s.running.Done()
		return true
	}
}

func TestConcurrentWatchWriteDetachAndCloseDoNotRace(t *testing.T) {
	clock := &expiries{}
	r, err := NewRegistry(time.Millisecond, clock.schedule)
	if err != nil {
		t.Fatal(err)
	}
	var windows sync.WaitGroup
	for index := range 40 {
		windows.Go(func() {
			window := "window-" + strconv.Itoa(index)
			watch := r.Watch(window)
			drained := make(chan struct{})
			go func() {
				defer close(drained)
				for range watch.Events {
				}
			}()
			own := plex("plex:"+strconv.Itoa(index), "T")
			shared := plex("plex:shared", "S")
			for range 50 {
				r.WriteTabs(window, []Tab{own, shared})
				r.Link(own.ID, "plex:shared")
				r.SyncPath(own.ID, "a.md", "")
				r.Detach(own.ID)
				r.CancelDetach(own.ID)
				r.Detach(own.ID)
				r.Unlink(own.ID)
			}
			r.Cancel(window)
			watch.Close()
			<-drained
		})
	}
	windows.Wait()
	clock.running.Wait()

	if tabs := r.GetSnapshot().Tabs; len(tabs) != 0 {
		t.Errorf("tabs are %+v, want none once every window closed", tabs)
	}
}
