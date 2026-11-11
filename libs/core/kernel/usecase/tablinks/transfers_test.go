package tablinks

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

func newOrbits() tablink.SavedTab {
	return tablink.SavedTab{ID: "plex:2", Kind: "plex", Path: "notes/Orbits.md", Title: "Orbits", State: `{"focus":"Kepler"}`}
}

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

func readOutcome(t *testing.T, result <-chan tablink.MoveError) tablink.MoveError {
	t.Helper()
	select {
	case outcome := <-result:
		return outcome
	default:
		t.Fatal("the move has no outcome")
		return tablink.MoveErrorNone
	}
}

func isMoveFor(windowID string) func(Event) bool {
	return func(e Event) bool { return e.Move != nil && e.IsFor(windowID) }
}

func TestMoveChangesTheOwnerAfterAccept(t *testing.T) {
	r, _ := newTestRegistry(t)
	target := r.Watch("window-b")
	defer target.Close()
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")

	result := r.Move("window-a", "window-b", newOrbits())
	event := readEvent(t, target, isMoveFor("window-b"))
	if event.Move.Tab != newOrbits() {
		t.Fatalf("the target received %+v, want %+v", event.Move.Tab, newOrbits())
	}
	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Fatalf("plex:2 is in window %q before the target accepted it, want window-a", got)
	}

	if got := r.Accept("window-b", "plex:2", getMoveID(r, "plex:2")); got != tablink.MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Errorf("the move ended with %v, want none", got)
	}
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

	got := r.GetSnapshot()
	if got.Tabs[1].WindowID != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got.Tabs[1].WindowID)
	}
	if want := [][]string{{"plex:1", "plex:2"}}; !reflect.DeepEqual(getGroupTabs(got), want) {
		t.Errorf("groups are %v, want %v", getGroupTabs(got), want)
	}
}

func TestMoveAndHighlightEventsAreForTheTargetWindowOnly(t *testing.T) {
	event := Event{Move: &MovedTab{WindowID: "window-b", Tab: newOrbits()}}
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
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})

	if got := readOutcome(t, r.Move("window-a", "window-b", newOrbits())); got != tablink.MoveErrorWindowClosed {
		t.Errorf("a move into a closed window ended with %v, want window closed", got)
	}
	if got := readOutcome(t, r.Move("window-a", "window-a", newOrbits())); got != tablink.MoveErrorWindowClosed {
		t.Errorf("a move into its own window ended with %v, want window closed", got)
	}
}

func TestMoveIntoAWindowEndsWhenItsWatchCloses(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:3", "Tides")})
	target := r.watches["window-b"]
	result := r.Move("window-a", "window-b", newOrbits())

	target.Close()

	if got := readOutcome(t, result); got != tablink.MoveErrorWindowClosed {
		t.Errorf("the move ended with %v before the grace period, want window closed", got)
	}
	if got := r.Accept("window-b", "plex:2", getMoveID(r, "plex:2")); got != tablink.MoveErrorTimeout {
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
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Move("window-a", "window-b", newOrbits())

	clock.fireAll()

	if got := readOutcome(t, result); got != tablink.MoveErrorTimeout {
		t.Errorf("the move ended with %v, want timeout", got)
	}
	if clock.pending[0].after != MoveTimeout {
		t.Errorf("the move waits %v, want %v", clock.pending[0].after, MoveTimeout)
	}
	if got := r.Accept("window-b", "plex:2", getMoveID(r, "plex:2")); got != tablink.MoveErrorTimeout {
		t.Errorf("a late Accept returned %v, want timeout", got)
	}
}

func TestTargetReportBeforeAcceptDoesNotChangeTheOwner(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	writeTabs(r, "window-b", nil)
	result := r.Move("window-a", "window-b", newOrbits())

	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})
	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Fatalf("plex:2 is in window %q before the target accepted it, want window-a", got)
	}
	clock.fireAll()
	writeTabs(r, "window-b", nil)

	if got := readOutcome(t, result); got != tablink.MoveErrorTimeout {
		t.Errorf("the move ended with %v, want timeout", got)
	}
	if len(r.GetSnapshot().Groups) != 1 {
		t.Errorf("the tab left its group after a move that did not complete")
	}
}

func TestTargetReportBeforeAcceptIsNoClaim(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	source := r.watches["window-a"]
	writeTabs(r, "window-b", nil)
	r.Move("window-a", "window-b", newOrbits())
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})

	source.Close()

	if got := r.GetSnapshot().Tabs[1].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q after the source closed, want window-a", got)
	}
}

func TestStaleSourceReportAfterAcceptDoesNotChangeTheOwner(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	r.Link("plex:1", "plex:2")
	writeTabs(r, "window-b", nil)
	r.Move("window-a", "window-b", newOrbits())
	r.Accept("window-b", "plex:2", getMoveID(r, "plex:2"))

	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A"), plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:1", "A")})

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
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", nil)
	writeTabs(r, "window-c", nil)
	result := r.Move("window-a", "window-b", newOrbits())

	if got := r.Accept("window-c", "plex:2", getMoveID(r, "plex:2")); got != tablink.MoveErrorTimeout {
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
	settings := tablink.SavedTab{ID: "settings:1", Kind: "settings"}
	result := r.Move("window-a", "window-b", settings)

	if got := r.Accept("window-b", "settings:1", getMoveID(r, "settings:1")); got != tablink.MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Errorf("the move ended with %v, want none", got)
	}
	if tabs := r.GetSnapshot().Tabs; len(tabs) != 0 {
		t.Errorf("tabs are %+v, want none", tabs)
	}
}

func TestReadDetachedReturnsTheTabUntilTheGracePeriodEnds(t *testing.T) {
	r, clock := newTestRegistry(t)

	r.Detach(newOrbits())

	got, _, ok := r.ReadDetached("window-b", "plex:2")
	if !ok || got != newOrbits() {
		t.Fatalf("ReadDetached returned %+v, %v, want %+v", got, ok, newOrbits())
	}
	if _, _, ok := r.ReadDetached("window-b", "plex:9"); ok {
		t.Error("ReadDetached returned a tab with no pending detach")
	}
	clock.fireAll()
	if _, _, ok := r.ReadDetached("window-b", "plex:2"); ok {
		t.Error("the tab is still kept after the grace period")
	}
}

func TestCancelDetachDropsTheStoredTab(t *testing.T) {
	r, _ := newTestRegistry(t)
	result := r.Detach(newOrbits())

	r.CancelDetach("plex:2")

	if got := readOutcome(t, result); got != tablink.MoveErrorCanceled {
		t.Errorf("the detach ended with %v, want canceled", got)
	}

	if got, _, ok := r.ReadDetached("window-b", "plex:2"); ok {
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

func TestSnapshotListsMainWindowFirst(t *testing.T) {
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

func readDetachID(r *registry, windowID, tabID string) string {
	_, moveID, _ := r.ReadDetached(windowID, tabID)
	return moveID
}

func getMoveID(r *registry, tabID string) string {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if pending, ok := r.state.transit.getMove(tabID); ok {
		return pending.id
	}
	return ""
}

func TestMoveRejectsATabOwnedByAnotherWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-c", nil)

	if got := readOutcome(t, r.Move("window-b", "window-c", newOrbits())); got != tablink.MoveErrorNotOwner {
		t.Errorf("a move from a window that only claims the tab ended with %v, want not owner", got)
	}
	if got := r.GetSnapshot().Tabs[0].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q, want window-a", got)
	}
}

func TestAcceptOfAnOlderMoveDoesNotCompleteTheNewerMove(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", nil)
	first := r.Move("window-a", "window-b", newOrbits())
	firstID := getMoveID(r, "plex:2")
	second := r.Move("window-a", "window-b", newOrbits())

	if got := r.Accept("window-b", "plex:2", firstID); got != tablink.MoveErrorTimeout {
		t.Errorf("Accept of the older move returned %v, want timeout", got)
	}
	if got := readOutcome(t, first); got != tablink.MoveErrorCanceled {
		t.Errorf("the older move ended with %v, want canceled", got)
	}
	if got := r.Accept("window-b", "plex:2", getMoveID(r, "plex:2")); got != tablink.MoveErrorNone {
		t.Errorf("Accept of the newer move returned %v, want none", got)
	}
	if got := readOutcome(t, second); got != tablink.MoveErrorNone {
		t.Errorf("the newer move ended with %v, want none", got)
	}
}

func TestAcceptRepeatedAfterTheMoveCompletedReturnsNone(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", nil)
	r.Move("window-a", "window-b", newOrbits())
	moveID := getMoveID(r, "plex:2")
	r.Accept("window-b", "plex:2", moveID)

	if got := r.Accept("window-b", "plex:2", moveID); got != tablink.MoveErrorNone {
		t.Errorf("a repeated Accept returned %v, want none", got)
	}
	if got := r.Accept("window-c", "plex:2", moveID); got != tablink.MoveErrorTimeout {
		t.Errorf("an Accept from another window returned %v, want timeout", got)
	}
	clock.fireAll()
	if got := r.Accept("window-b", "plex:2", moveID); got != tablink.MoveErrorTimeout {
		t.Errorf("an Accept after the grace period returned %v, want timeout", got)
	}
}

func TestCancelMoveRejectsALaterAccept(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-b", nil)
	result := r.Move("window-a", "window-b", newOrbits())
	moveID := getMoveID(r, "plex:2")

	r.CancelMove("plex:2", result)

	if got := readOutcome(t, result); got != tablink.MoveErrorCanceled {
		t.Errorf("the canceled move ended with %v, want canceled", got)
	}
	if got := r.Accept("window-b", "plex:2", moveID); got != tablink.MoveErrorTimeout {
		t.Errorf("Accept after cancel returned %v, want timeout", got)
	}
	if got := r.GetSnapshot().Tabs[0].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q, want window-a", got)
	}
}

func TestReadDetachedReturnsTheTabToTheFirstReadingWindowOnly(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.Detach(newOrbits())

	if _, _, ok := r.ReadDetached("window-b", "plex:2"); !ok {
		t.Fatal("the first window read no tab")
	}
	if got, _, ok := r.ReadDetached("window-c", "plex:2"); ok {
		t.Errorf("a second window read %+v", got)
	}
	if _, _, ok := r.ReadDetached("window-b", "plex:2"); !ok {
		t.Error("the first window read no tab the second time")
	}
}

func TestDetachEndsWhenTheReadingWindowAcceptsTheTab(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Detach(newOrbits())
	detachID := readDetachID(r, "window-b", "plex:2")

	if got := r.Accept("window-c", "plex:2", detachID); got != tablink.MoveErrorTimeout {
		t.Errorf("Accept from a window that did not read the detach returned %v, want timeout", got)
	}
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})
	select {
	case outcome := <-result:
		t.Fatalf("the detach ended with %v before the new window accepted the tab", outcome)
	default:
	}
	if got := r.Accept("window-b", "plex:2", detachID); got != tablink.MoveErrorNone {
		t.Fatalf("Accept returned %v, want none", got)
	}

	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Errorf("the detach ended with %v, want none", got)
	}
	if got := r.GetSnapshot().Tabs[0].WindowID; got != "window-b" {
		t.Errorf("plex:2 is in window %q, want window-b", got)
	}
}

func TestDetachOfAnUnknownTabEndsOnAccept(t *testing.T) {
	r, _ := newTestRegistry(t)
	settings := tablink.SavedTab{ID: "settings:1", Kind: "settings"}
	result := r.Detach(settings)

	if got := r.Accept("window-b", "settings:1", readDetachID(r, "window-b", "settings:1")); got != tablink.MoveErrorNone {
		t.Errorf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Errorf("the detach ended with %v, want none", got)
	}
}

func TestDetachEndsWithTimeoutAfterTheGracePeriod(t *testing.T) {
	r, clock := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Detach(newOrbits())

	clock.fireAll()

	if got := readOutcome(t, result); got != tablink.MoveErrorTimeout {
		t.Errorf("the detach ended with %v, want timeout", got)
	}
	if got := r.GetSnapshot().Tabs[0].WindowID; got != "window-a" {
		t.Errorf("plex:2 is in window %q, want window-a", got)
	}
}

func TestDetachEndsWhenTheReadingWindowStopsWatching(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Detach(newOrbits())
	r.ReadDetached("window-b", "plex:2")
	watch := r.Watch("window-b")

	watch.Close()

	if got := readOutcome(t, result); got != tablink.MoveErrorWindowClosed {
		t.Errorf("the detach ended with %v, want window closed", got)
	}
}

// The source window detaches a tab and, while the new window starts, moves the
// same tab into another open window.
func TestDetachAndMoveOfOneTabDoNotBothComplete(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-c", nil)
	detached := r.Detach(newOrbits())
	detachID := readDetachID(r, "window-b", "plex:2")
	moved := r.Move("window-a", "window-c", newOrbits())

	detachAccept := r.Accept("window-b", "plex:2", detachID)
	moveAccept := r.Accept("window-c", "plex:2", getMoveID(r, "plex:2"))

	detachOutcome, moveOutcome := readOutcome(t, detached), readOutcome(t, moved)
	if detachAccept == tablink.MoveErrorNone && moveAccept == tablink.MoveErrorNone {
		t.Errorf("both windows accepted plex:2 (detach %v, move %v): the tab is open in window-b and window-c",
			detachOutcome, moveOutcome)
	}
}

// The source window closes before the new window accepts the tab, so the
// report of the new window takes the tab first.
func TestReportOfTheReadingWindowCompletesTheDetach(t *testing.T) {
	r, _ := newTestRegistry(t)
	source := r.Watch("window-a")
	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Detach(newOrbits())
	detachID := readDetachID(r, "window-b", "plex:2")

	source.Close()
	writeTabs(r, "window-b", []tablink.Tab{plex("plex:2", "Orbits")})

	if got := r.Accept("window-b", "plex:2", detachID); got != tablink.MoveErrorNone {
		t.Errorf("Accept returned %v, want none", got)
	}
	if got := readOutcome(t, result); got != tablink.MoveErrorNone {
		t.Errorf("the detach ended with %v, want none", got)
	}
}

func TestDetachEndsAPendingMoveOfTheSameTab(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-c", nil)
	moved := r.Move("window-a", "window-c", newOrbits())
	moveID := getMoveID(r, "plex:2")

	r.Detach(newOrbits())

	if got := readOutcome(t, moved); got != tablink.MoveErrorCanceled {
		t.Errorf("the move ended with %v, want canceled", got)
	}
	if got := r.Accept("window-c", "plex:2", moveID); got == tablink.MoveErrorNone {
		t.Error("window-c accepted a move that the detach ended")
	}
}

func TestMoveEndsAPendingDetachOfTheSameTab(t *testing.T) {
	r, _ := newTestRegistry(t)
	writeTabs(r, "window-a", []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-c", nil)
	detached := r.Detach(newOrbits())
	detachID := readDetachID(r, "window-b", "plex:2")

	r.Move("window-a", "window-c", newOrbits())

	if got := readOutcome(t, detached); got != tablink.MoveErrorCanceled {
		t.Errorf("the detach ended with %v, want canceled", got)
	}
	if got := r.Accept("window-b", "plex:2", detachID); got == tablink.MoveErrorNone {
		t.Error("window-b accepted a detach that the move ended")
	}
}

func TestDetachEndsWithCanceledOnAReportOfAnotherWindow(t *testing.T) {
	r, _ := newTestRegistry(t)
	source := r.Watch("window-a")
	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Detach(newOrbits())
	readDetachID(r, "window-b", "plex:2")

	source.Close()
	writeTabs(r, "window-c", []tablink.Tab{plex("plex:2", "Orbits")})

	if got := readOutcome(t, result); got != tablink.MoveErrorCanceled {
		t.Errorf("the detach ended with %v, want canceled", got)
	}
}

func TestMoveEndsWithCanceledWhenTheTabOwnerChanges(t *testing.T) {
	r, _ := newTestRegistry(t)
	source := r.Watch("window-a")
	r.WriteTabs(Window{ID: "window-a"}, []tablink.Tab{plex("plex:2", "Orbits")})
	writeTabs(r, "window-t", nil)
	writeTabs(r, "window-c", []tablink.Tab{plex("plex:2", "Orbits")})
	result := r.Move("window-a", "window-t", newOrbits())
	moveID := getMoveID(r, "plex:2")

	source.Close()

	if got := readOutcome(t, result); got != tablink.MoveErrorCanceled {
		t.Errorf("the move ended with %v, want canceled", got)
	}
	if got := r.Accept("window-t", "plex:2", moveID); got == tablink.MoveErrorNone {
		t.Error("window-t accepted plex:2 after window-c took it")
	}
}
