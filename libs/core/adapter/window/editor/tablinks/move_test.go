package tablinks

import (
	"reflect"
	"testing"
	"time"
)

// newOrbits returns the tab that the tests move.
func newOrbits() MovedTab {
	return MovedTab{ID: "plex:2", Kind: "plex", Path: "notes/Orbits.md", Title: "Orbits", State: `{"focus":"Kepler"}`}
}

// readEvent returns the first matching event of a watch, and fails the test
// when none arrives.
func readEvent(t *testing.T, watch *Watch, isWanted func(Event) bool) Event {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event := <-watch.Events:
			if isWanted(event) {
				return event
			}
		case <-timeout:
			t.Fatal("no wanted event arrived")
		}
	}
}

// readOutcome returns the result of a move, and fails the test when there is none yet.
func readOutcome(t *testing.T, result <-chan MoveError) MoveError {
	t.Helper()
	select {
	case outcome := <-result:
		return outcome
	default:
		t.Fatal("the move has no outcome")
		return MoveErrorNone
	}
}

func isMoveFor(windowID string) func(Event) bool {
	return func(e Event) bool { return e.Move != nil && e.IsFor(windowID) }
}

func TestMoveChangesTheOwnerAfterAccept(t *testing.T) {
	r, _ := newTestRegistry(t)
	target := r.Watch("window-b")
	defer target.Close()
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")

	result := r.Move("window-a", "window-b", newOrbits())
	event := readEvent(t, target, isMoveFor("window-b"))
	if event.Move.Tab != newOrbits() {
		t.Fatalf("the target received %+v, want %+v", event.Move.Tab, newOrbits())
	}
	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Fatalf("plex:2 is in window %q before the target accepted it, want window-a", got)
	}

	if got := r.Accept("window-b", "plex:2"); got != MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != MoveErrorNone {
		t.Errorf("the move ended with %v, want none", got)
	}
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	got := r.GetSnapshot()
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(got), want)
	}
}

func TestMoveAndHighlightEventsAreForTheTargetWindowOnly(t *testing.T) {
	event := Event{Move: &TabMove{WindowID: "window-b", Tab: newOrbits()}}
	if event.IsFor("window-a") || !event.IsFor("window-b") {
		t.Error("a move is for a window other than its target")
	}
	highlight := Event{Highlight: &Highlight{WindowID: "window-b", IsOn: true}}
	if highlight.IsFor("window-a") || !highlight.IsFor("window-b") {
		t.Error("a highlight is for a window other than its target")
	}
	if !(Event{Snapshot: &Snapshot{}}).IsFor("window-a") {
		t.Error("a snapshot is not for every window")
	}
}

func TestMoveIntoAWindowWithNoWatchReturnsWindowClosed(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:2", "Orbits")})

	if got := readOutcome(t, r.Move("window-a", "window-b", newOrbits())); got != MoveErrorWindowClosed {
		t.Errorf("a move into a closed window ended with %v, want window closed", got)
	}
	if got := readOutcome(t, r.Move("window-a", "window-a", newOrbits())); got != MoveErrorWindowClosed {
		t.Errorf("a move into its own window ended with %v, want window closed", got)
	}
}

func TestMoveIntoAWindowEndsWhenItsWatchCloses(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", []Tab{plex("plex:3", "Tides")})
	target := r.watches["window-b"][0]
	result := r.Move("window-a", "window-b", newOrbits())

	target.Close()

	if got := readOutcome(t, result); got != MoveErrorWindowClosed {
		t.Errorf("the move ended with %v before the grace period, want window closed", got)
	}
	if got := r.Accept("window-b", "plex:2"); got != MoveErrorTimeout {
		t.Errorf("a late Accept returned %v, want timeout", got)
	}
	if got := getTabIDs(r.GetSnapshot()); !reflect.DeepEqual(got, []string{"plex:2", "plex:3"}) {
		t.Errorf("tabs within the grace period are %v, want plex:2 and plex:3", got)
	}
	if got := r.GetSnapshot().Tabs[0].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q, want window-a", got)
	}
	clock.fireAll()
	if got := getTabIDs(r.GetSnapshot()); !reflect.DeepEqual(got, []string{"plex:2"}) {
		t.Errorf("tabs after the grace period are %v, want plex:2", got)
	}
}

func TestMoveReturnsTimeoutWithoutAccept(t *testing.T) {
	r, clock := newTestRegistry(t)
	target := r.Watch("window-b")
	defer target.Close()
	writeTabs(r, "window-a", []Tab{plex("plex:2", "Orbits")})
	result := r.Move("window-a", "window-b", newOrbits())

	clock.fireAll()

	if got := readOutcome(t, result); got != MoveErrorTimeout {
		t.Errorf("the move ended with %v, want timeout", got)
	}
	if clock.pending[0].after != MoveTimeout {
		t.Errorf("the move waits %v, want %v", clock.pending[0].after, MoveTimeout)
	}
	if got := r.Accept("window-b", "plex:2"); got != MoveErrorTimeout {
		t.Errorf("a late Accept returned %v, want timeout", got)
	}
}

func TestTargetReportBeforeAcceptDoesNotChangeTheOwner(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	writeTabs(r, "window-b", nil)
	result := r.Move("window-a", "window-b", newOrbits())

	writeTabs(r, "window-b", []Tab{plex("plex:2", "Orbits")})
	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Fatalf("plex:2 is in window %q before the target accepted it, want window-a", got)
	}
	clock.fireAll()
	writeTabs(r, "window-b", nil)

	if got := readOutcome(t, result); got != MoveErrorTimeout {
		t.Errorf("the move ended with %v, want timeout", got)
	}
	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("the tab left its group after a move that did not complete")
	}
}

func TestTargetReportBeforeAcceptIsNoClaim(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	source := r.watches["window-a"][0]
	writeTabs(r, "window-b", nil)
	r.Move("window-a", "window-b", newOrbits())
	writeTabs(r, "window-b", []Tab{plex("plex:2", "Orbits")})

	source.Close()

	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q after the source closed, want window-a", got)
	}
}

func TestStaleSourceReportAfterAcceptDoesNotChangeTheOwner(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	writeTabs(r, "window-b", nil)
	r.Move("window-a", "window-b", newOrbits())
	r.Accept("window-b", "plex:2")

	writeTabs(r, "window-a", []Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", []Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-a", []Tab{plex("plex:1", "A")})

	got := r.GetSnapshot()
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
	if len(got.Groups) != 1 {
		t.Errorf("groups are %+v, want the one group", got.Groups)
	}
}

func TestAcceptFromAnotherWindowIsIgnored(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", nil)
	writeTabs(r, "window-c", nil)
	result := r.Move("window-a", "window-b", newOrbits())

	if got := r.Accept("window-c", "plex:2"); got != MoveErrorTimeout {
		t.Errorf("Accept from another window returned %v, want timeout", got)
	}
	select {
	case outcome := <-result:
		t.Errorf("the move ended with %v", outcome)
	default:
	}
}

func TestMoveOfAnUnknownTabCompletes(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-b", nil)
	settings := MovedTab{ID: "settings:1", Kind: "settings"}
	result := r.Move("window-a", "window-b", settings)

	if got := r.Accept("window-b", "settings:1"); got != MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != MoveErrorNone {
		t.Errorf("the move ended with %v, want none", got)
	}
	if tabs := r.GetSnapshot().Tabs; len(tabs) != 0 {
		t.Errorf("tabs are %+v, want none", tabs)
	}
}

func TestReadDetachedReturnsTheTabUntilTheGracePeriodEnds(t *testing.T) {
	r, clock := newTestRegistry(t)

	r.Detach(newOrbits())

	got, ok := r.ReadDetached("plex:2")
	if !ok || got != newOrbits() {
		t.Fatalf("ReadDetached returned %+v, %v, want %+v", got, ok, newOrbits())
	}
	if _, ok := r.ReadDetached("plex:9"); ok {
		t.Error("ReadDetached returned a tab with no pending detach")
	}
	clock.fireAll()
	if _, ok := r.ReadDetached("plex:2"); ok {
		t.Error("the tab is still kept after the grace period")
	}
}

func TestCancelDetachDropsTheStoredTab(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.Detach(newOrbits())

	r.CancelDetach("plex:2")

	if got, ok := r.ReadDetached("plex:2"); ok {
		t.Errorf("ReadDetached returned %+v after CancelDetach", got)
	}
}

func TestHighlightIsSentToAWatchingWindowOnly(t *testing.T) {
	r, _ := newTestRegistry(t)
	target := r.Watch("window-b")
	defer target.Close()

	r.Highlight("window-c", true)
	r.Highlight("window-b", true)

	event := readEvent(t, target, func(e Event) bool { return e.Highlight != nil })
	if *event.Highlight != (Highlight{WindowID: "window-b", IsOn: true}) {
		t.Errorf("the highlight is %+v", *event.Highlight)
	}
}

func TestTheSnapshotListsTheMainWindowFirst(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.Watch("window-b")
	r.Watch("window-a")
	r.WriteTabs(Window{ID: "window-b", ActiveTitle: "Tides", ActiveKind: "plex", TabCount: 3}, nil)
	r.WriteTabs(Window{ID: "window-a", IsMain: true, ActiveTitle: "Orbits", ActiveKind: "note", TabCount: 1}, nil)

	want := []Window{
		{ID: "window-a", IsMain: true, ActiveTitle: "Orbits", ActiveKind: "note", TabCount: 1},
		{ID: "window-b", ActiveTitle: "Tides", ActiveKind: "plex", TabCount: 3},
	}
	if got := r.GetSnapshot().Windows; !reflect.DeepEqual(got, want) {
		t.Errorf("windows are %+v, want %+v", got, want)
	}
}

func TestClosingTheLastWatchRemovesTheWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	watch := r.Watch("window-b")
	r.WriteTabs(Window{ID: "window-b", TabCount: 1}, nil)

	watch.Close()

	if got := r.GetSnapshot().Windows; len(got) != 0 {
		t.Errorf("windows within the grace period are %+v, want none", got)
	}
	r.Watch("window-b")
	want := []Window{{ID: "window-b", TabCount: 1}}
	if got := r.GetSnapshot().Windows; !reflect.DeepEqual(got, want) {
		t.Errorf("windows after a new watch are %+v, want %+v", got, want)
	}
}
