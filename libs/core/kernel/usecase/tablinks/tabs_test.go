package tablinks

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
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

func (s *timers) fireAll() {
	for _, one := range s.pending {
		if !one.isCancelled {
			one.isCancelled = true
			one.run()
		}
	}
}

const grace = 30 * time.Second

func newTestRegistry(t *testing.T) (*registry, *timers) {
	t.Helper()
	clock := &timers{}
	r, err := newRegistry(grace, clock.schedule)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func plex(id, title string) tablink.Tab  { return tablink.Tab{ID: id, Kind: "plex", Title: title} }
func files(id, title string) tablink.Tab { return tablink.Tab{ID: id, Kind: "files", Title: title} }
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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "Orbits")})
	writeTabs(r, "window-b", []tablink.Tab{files("files:1", "Files")})

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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "Orbits")})

	r.Link("plex:1", "plex:ghost")
	r.Link("plex:1", "plex:1")

	if groups := r.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestLinkAddsTabToGroupOfEitherSide(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C"), plex("plex:4", "D")})

	r.Link("plex:1", "plex:2")
	r.Link("plex:1", "plex:3")
	r.Link("plex:4", "plex:2")

	if want := [][]string{{"plex:1", "plex:2", "plex:3", "plex:4"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestLinkMergesTwoGroups(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:3", "C"), plex("plex:4", "D")})
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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

	got := r.GetSnapshot()
	if len(got.Groups) != 0 {
		t.Errorf("groups are %+v, want none", got.Groups)
	}
	if want := []string{"plex:1"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestCloseOfLastWatchUnlinksWindowTabsAfterGracePeriod(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	watch := r.Watch("window-b")
	writeTabs(r, "window-b", []tablink.Tab{files("files:1", "Files")})
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
	r, _ := newTestRegistry(t)
	first := r.Watch("window-a")
	second := r.Watch("window-a")
	defer second.Close()
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	first.Close()

	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("groups are %+v, want the one group", r.GetSnapshot().Groups)
	}
}

func TestDetachKeepsLinksWhenNewWindowAcceptsTab(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	if len(r.GetSnapshot().Groups) != 1 {
		t.Fatalf("the detached tab left its group before the grace period ended")
	}
	detachID := readDetachID(r, "window-b", "plex:2")
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "B")})
	r.Accept("window-b", "plex:2", detachID)
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
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

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
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})

	clock.fireAll()

	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("a tab its window still reports left its group")
	}
}

func TestDetachKeepsLinksAfterOldWindowClosesUntilGracePeriodEnds(t *testing.T) {
	r, clock := newTestRegistry(t)
	watch := r.Watch("window-a")
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	writeTabs(r, "window-c", []tablink.Tab{plex("plex:3", "C")})
	r.Link("plex:1", "plex:3")
	r.Detach(tablink.SavedTab{ID: "plex:1", Kind: "plex"})

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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
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
	r, _ := newTestRegistry(t)
	first, second := r.Watch("window-a"), r.Watch("window-b")
	defer first.Close()
	defer second.Close()
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	writeTabs(r, "window-b", []tablink.Tab{files("files:1", "Files")})
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
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

	if r.SyncPath("plex:1", "notes/Orbits.md", "") {
		t.Error("a tab in no group was synced")
	}
}

func TestCancelEndsEveryWatchOfTheWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	mine, other := r.Watch("window-a"), r.Watch("window-b")
	defer mine.Close()
	defer other.Close()

	r.Cancel("window-a")
	r.Cancel("window-a")

	select {
	case <-mine.Stop:
	default:
		t.Error("the canceled watch is still open")
	}
	select {
	case <-other.Stop:
		t.Error("the watch of another window was canceled")
	default:
	}
}

type received struct {
	links Snapshot
	sync  PathSync
}

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

func writeTabs(r *registry, windowID string, tabs []tablink.Tab) {
	r.state.mu.Lock()
	isWatched := r.state.roster.presence.hasWatch(windowID)
	r.state.mu.Unlock()
	if !isWatched {
		r.watches[windowID] = r.Watch(windowID)
	}
	r.WriteTabs(Window{ID: windowID}, tabs)
}

func TestWriteTabsIgnoresAWindowWithNoWatch(t *testing.T) {
	r, _ := newTestRegistry(t)
	watch := r.Watch("window-a")
	watch.Close()

	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:1", "A")})
	r.WriteTabs(Window{ID: "window-b"}, []tablink.Tab{plex("plex:2", "B")})

	if tabs := r.GetSnapshot().Tabs; len(tabs) != 0 {
		t.Errorf("tabs are %+v, want none", tabs)
	}
}

func TestWriteTabsKeepsTabOfAnotherLiveWindowOutsideDetach(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

	writeTabs(r, "window-b", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})

	got := r.GetSnapshot()
	if got.Tabs[0].WindowID != "window-a" {
		t.Errorf("plex:1 is in window %q, want window-a", got.Tabs[0].WindowID)
	}
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "B")})
	if want := []string{"plex:1", "plex:2"}; !reflect.DeepEqual(getTabIDs(r.GetSnapshot()), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(r.GetSnapshot()), want)
	}
}

func TestDetachIgnoresStaleOldWindowReportAfterNewWindow(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})

	detachID := readDetachID(r, "window-b", "plex:2")
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "B")})
	r.Accept("window-b", "plex:2", detachID)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
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
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})

	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	detachID := readDetachID(r, "window-b", "plex:2")
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "B")})
	r.Accept("window-b", "plex:2", detachID)
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
}

func TestNewRegistryRequiresASchedule(t *testing.T) {
	if _, err := newRegistry(grace, nil); err == nil {
		t.Error("a registry with no schedule was made")
	}
}

func TestWriteTabsAfterReloadKeepsTabsOfTheNewWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	old := r.Watch("window-old")
	r.WriteTabs(Window{ID: "window-old"}, []tablink.Tab{plex("plex:1", "A")})
	reloaded := r.Watch("window-new")
	defer reloaded.Close()

	r.WriteTabs(Window{ID: "window-new"}, []tablink.Tab{plex("plex:1", "A")})
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
	r, clock := newTestRegistry(t)
	old := r.Watch("window-old")
	r.WriteTabs(Window{ID: "window-old"}, []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	old.Close()
	reloaded := r.Watch("window-new")
	defer reloaded.Close()

	r.WriteTabs(Window{ID: "window-new"}, []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	clock.fireAll()

	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestWatchDeliversSnapshotBeforeLaterSync(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
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
	r, clock := newTestRegistry(t)
	first := r.Watch("window-a")
	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")

	first.Close()
	second := r.Watch("window-a")
	defer second.Close()
	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	clock.fireAll()

	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(r.GetSnapshot()), want)
	}
}

func TestFailedDetachDropsTabClosedInItsWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C")})
	r.Link("plex:1", "plex:2")
	r.Link("plex:1", "plex:3")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})

	r.CancelDetach("plex:2")
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:3", "C")})

	if want := []string{"plex:1", "plex:3"}; !reflect.DeepEqual(getTabIDs(r.GetSnapshot()), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(r.GetSnapshot()), want)
	}
}

func TestCancelDetachDropsTabItsWindowNoLongerReports(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	r.Detach(tablink.SavedTab{ID: "plex:2", Kind: "plex"})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

	r.CancelDetach("plex:2")

	if len(r.GetSnapshot().Groups) != 0 {
		t.Errorf("groups are %+v, want none", r.GetSnapshot().Groups)
	}
}

// realTimers is a Schedule backed by time.AfterFunc whose runs a test can wait for.
type realTimers struct{ running sync.WaitGroup }

func (s *realTimers) schedule(d time.Duration, f func()) func() bool {
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

func TestConcurrentWatchWriteDetachMoveAndCloseDoNotRace(t *testing.T) {
	clock := &realTimers{}
	r, err := newRegistry(time.Millisecond, clock.schedule)
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
				r.WriteTabs(Window{ID: window}, []tablink.Tab{own, shared})
				r.Link(own.ID, "plex:shared")
				r.SyncPath(own.ID, "a.md", "")
				r.Detach(tablink.SavedTab{ID: own.ID, Kind: own.Kind})
				r.CancelDetach(own.ID)
				r.Detach(tablink.SavedTab{ID: own.ID, Kind: own.Kind})
				r.Unlink(own.ID)
				r.Move(window, "window-0", tablink.SavedTab{ID: own.ID, Kind: own.Kind})
				r.Accept("window-0", own.ID, getMoveID(r, own.ID))
				if _, detachID, ok := r.ReadDetached(window, own.ID); ok {
					r.Accept(window, own.ID, detachID)
				}
				r.Highlight("window-0", true)
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

func TestSnapshotCarriesTheGroupPath(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B")})
	r.Link("plex:1", "plex:2")
	if group := r.GetSnapshot().Groups[0]; group.Path != "" || group.Title != "" {
		t.Fatalf("a group with no sync has path %q and title %q, want none", group.Path, group.Title)
	}

	r.SyncPath("plex:2", "notes/Tides.md", "Tides")

	reconnected := r.Watch("window-b")
	defer reconnected.Close()
	group := reconnected.Initial.Groups[0]
	if group.Path != "notes/Tides.md" || group.Title != "Tides" {
		t.Errorf("the first snapshot has path %q and title %q, want notes/Tides.md and Tides", group.Path, group.Title)
	}
}

func TestLinkMergesTwoGroupsOnTheSourcePath(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "B"), plex("plex:3", "C"), plex("plex:4", "D")})
	r.Link("plex:1", "plex:2")
	r.Link("plex:3", "plex:4")
	r.SyncPath("plex:3", "notes/Orbits.md", "Orbits")

	r.Link("plex:1", "plex:3")

	group := r.GetSnapshot().Groups[0]
	if group.Path != "notes/Orbits.md" || group.Title != "Orbits" {
		t.Errorf("the merged group has path %q and title %q, want the synced notes/Orbits.md", group.Path, group.Title)
	}
}

// The window made for a detach accepts the tab during its layout start, before
// its tab links watch starts. A report of the source window sent before it
// closed the tab then arrives.
func TestDetachAcceptedBeforeTheNewWindowWatchesKeepsLinksAfterStaleSourceReport(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	result := r.Detach(newOrbits())

	detachID := readDetachID(r, "window-b", "plex:2")
	if got := r.Accept("window-b", "plex:2", detachID); got != tablink.MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Fatalf("the detach ended with %v, want none", got)
	}
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})
	clock.fireAll()

	got := r.GetSnapshot()
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Fatalf("groups are %v, want %v", getGroupTabs(got), want)
	}
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
}

func TestTabOfAWindowThatNeverWatchesIsUnlinkedAfterGracePeriod(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	r.Detach(newOrbits())
	r.Accept("window-b", "plex:2", readDetachID(r, "window-b", "plex:2"))
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})
	if len(r.GetSnapshot().Groups) != 1 {
		t.Fatalf("plex:2 left its group before the grace period of window-b ended")
	}

	clock.fireAll()

	got := r.GetSnapshot()
	if len(got.Groups) != 0 {
		t.Errorf("groups are %+v, want none", got.Groups)
	}
	if want := []string{"plex:1"}; !reflect.DeepEqual(getTabIDs(got), want) {
		t.Errorf("tabs are %v, want %v", getTabIDs(got), want)
	}
}

func TestReloadedWindowDropsGroupMatesItDoesNotReport(t *testing.T) {
	r, _ := newTestRegistry(t)
	closed := r.Watch("window-a")
	writeTabs(r, "window-a", []tablink.Tab{files("files:1", "Files"), plex("note:1", "A")})
	r.Link("files:1", "note:1")
	closed.Close()
	live := r.Watch("window-b")
	defer live.Close()
	writeTabs(r, "window-b", []tablink.Tab{files("files:2", "Files"), plex("note:1", "A")})
	r.Link("files:2", "note:1")
	if want := [][]string{{"files:2", "note:1"}}; !reflect.DeepEqual(getGroupTabs(r.GetSnapshot()), want) {
		t.Fatalf("groups are %v after the link, want %v", getGroupTabs(r.GetSnapshot()), want)
	}

	writeTabs(r, "window-b", []tablink.Tab{files("files:2", "Files")})

	if got := getGroupTabs(r.GetSnapshot()); len(got) != 0 {
		t.Errorf("groups are %v after the linked tab closed, want none", got)
	}
}

// registry is the three use cases of a test over one state.
type registry struct {
	*Links
	*Windows
	*Transfers
	state   *state
	watches map[string]*Watch
}

func newRegistry(grace time.Duration, schedule Schedule) (*registry, error) {
	tabs, err := New(grace, schedule, &testEvents{})
	if err != nil {
		return nil, err
	}
	return &registry{Links: tabs.Links, Windows: tabs.Windows, Transfers: tabs.Transfers, state: tabs.Links.state.peek(), watches: map[string]*Watch{}}, nil
}

// testEvents delivers every event to every subscriber in the order published.
type testEvents struct {
	mu          sync.Mutex
	subscribers map[chan Event]bool
}

func (e *testEvents) Publish(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for out := range e.subscribers {
		select {
		case out <- event:
		default:
		}
	}
}

func (e *testEvents) Subscribe() (<-chan Event, func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		e.subscribers = map[chan Event]bool{}
	}
	out := make(chan Event, 100000)
	e.subscribers[out] = true
	return out, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.subscribers[out] {
			delete(e.subscribers, out)
			close(out)
		}
	}
}
