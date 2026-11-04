package tablinks

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

// timers is a Schedule whose runs a test fires by hand.
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

// fireAll runs every run not cancelled.
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
	return NewRegistry(grace, clock.schedule), clock
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

func TestLinkIgnoresATabNoWindowReported(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "Orbits")})

	r.Link("plex:1", "plex:ghost")
	r.Link("plex:1", "plex:1")

	if groups := r.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestLinkAddsATabToTheGroupOfEitherSide(t *testing.T) {
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

func TestUnlinkDropsAGroupLeftWithOneTab(t *testing.T) {
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

func TestWriteTabsDropsATabMissingFromTheReport(t *testing.T) {
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

func TestClosingTheLastWatchDropsTheTabsOfTheWindow(t *testing.T) {
	r, _ := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	watch := r.Watch("window-b")
	writeTabs(r, "window-b", []Tab{files("files:1", "Files")})
	r.Link("plex:1", "files:1")
	r.Link("plex:2", "files:1")

	watch.Close()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if want := []string{"plex:1", "plex:2"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestAWindowWithAnotherWatchKeepsItsTabs(t *testing.T) {
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

func TestADetachedTabKeepsItsLinksWhenTheNewWindowReportsIt(t *testing.T) {
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

func TestADetachedTabNoWindowReportsIsDroppedAfterTheGracePeriod(t *testing.T) {
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

func TestADetachThatFailedKeepsTheTabInItsWindow(t *testing.T) {
	r, clock := newTestRegistry()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach("plex:2")

	clock.fireAll()

	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("a tab its window still reports left its group")
	}
}

func TestADetachedTabOutlivesItsOldWindowForTheGracePeriod(t *testing.T) {
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

func TestALateWatchStartsFromTheCurrentState(t *testing.T) {
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

func TestEveryWatchReceivesEachChangeAndEachSync(t *testing.T) {
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

func TestSyncOfATabInNoGroupSendsNothing(t *testing.T) {
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

// readUntilSync reads a watch up to its first sync, and returns it with the
// last snapshot before it.
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

// writeTabs reports the tabs of a window, opening a watch for a window that has none.
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

func TestATabOfAnotherLiveWindowIsNotTakenOutsideADetach(t *testing.T) {
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

func TestAStaleReportOfTheOldWindowAfterTheNewWindowKeepsTheDetachedTab(t *testing.T) {
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

func TestAReportOfTheOldWindowBeforeTheNewWindowKeepsTheDetach(t *testing.T) {
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

func TestAWatchThatLagsReceivesTheLastSyncOfEveryGroup(t *testing.T) {
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
