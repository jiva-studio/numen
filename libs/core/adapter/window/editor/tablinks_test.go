package editor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/tablinks"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// newTabLinksClient serves the workspace with detach expiries disabled and
// returns a client for it.
func newTabLinksClient(t *testing.T, state *State) numenv1connect.WorkspaceServiceClient {
	t.Helper()
	return serveTabLinks(t, state, func(time.Duration, func()) func() bool {
		return func() bool { return true }
	})
}

// serveTabLinks serves the workspace with tab links on the schedule and returns
// a client for it.
func serveTabLinks(t *testing.T, state *State, schedule tablinks.Schedule) numenv1connect.WorkspaceServiceClient {
	t.Helper()
	tabLinks, err := tablinks.NewRegistry(linkGrace, schedule)
	if err != nil {
		t.Fatal(err)
	}
	state.TabLinks = tabLinks
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewWorkspaceServiceHandler(&workspaceHandler{state: state}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewWorkspaceServiceClient(server.Client(), server.URL)
}

func writeTabs(t *testing.T, client numenv1connect.WorkspaceServiceClient, windowID string, tabs ...*v1.LinkableTab) {
	t.Helper()
	_, err := client.WriteTabLinks(t.Context(), connect.NewRequest(&v1.WriteTabLinksRequest{
		WindowId: windowID,
		Change:   &v1.WriteTabLinksRequest_Tabs{Tabs: &v1.WindowTabs{Tabs: tabs}},
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func linkTabs(t *testing.T, client numenv1connect.WorkspaceServiceClient, source, target string) {
	t.Helper()
	_, err := client.WriteTabLinks(t.Context(), connect.NewRequest(&v1.WriteTabLinksRequest{
		WindowId: "window-a",
		Change:   &v1.WriteTabLinksRequest_Link{Link: &v1.TabLink{SourceTabId: source, TargetTabId: target}},
	}))
	if err != nil {
		t.Fatal(err)
	}
}

// readLinksUntil reads a stream until a snapshot matches the predicate.
func readLinksUntil(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchTabLinksResponse], isWanted func(*v1.TabLinks) bool,
) *v1.TabLinks {
	t.Helper()
	for stream.Receive() {
		if links := stream.Msg().GetLinks(); links != nil && isWanted(links) {
			return links
		}
	}
	t.Fatalf("the stream ended before the snapshot arrived: %v", stream.Err())
	return nil
}

func hasGroupOf(size int) func(*v1.TabLinks) bool {
	return func(links *v1.TabLinks) bool {
		return len(links.GetGroups()) == 1 && len(links.GetGroups()[0].GetTabIds()) == size
	}
}

func TestWatchTabLinksSendsGroupAndPathSyncToTwoWindows(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	ctx, cancel := context.WithTimeout(t.Context(), testsupport.Patience)
	defer cancel()

	first, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: "window-a"}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, first)
	second, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: "window-b"}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, second)
	if !first.Receive() || !second.Receive() {
		t.Fatal("a watch sent no snapshot")
	}

	writeTabs(t, client, "window-a", &v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"})
	writeTabs(t, client, "window-b", &v1.LinkableTab{Id: "files:1", Kind: "files", Title: "Files"})
	linkTabs(t, client, "plex:1", "files:1")

	for _, stream := range []*connect.ServerStreamForClient[v1.WatchTabLinksResponse]{first, second} {
		links := readLinksUntil(t, stream, hasGroupOf(2))
		if got := links.GetTabs()[1].GetWindowId(); got != "window-b" {
			t.Errorf("files:1 is in window %q, want window-b", got)
		}
	}

	_, err = client.SyncTabPath(ctx, connect.NewRequest(&v1.SyncTabPathRequest{
		SourceTabId: "files:1", Path: "notes/Orbits.md", Title: "Orbits",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, stream := range []*connect.ServerStreamForClient[v1.WatchTabLinksResponse]{first, second} {
		for stream.Receive() {
			if sync := stream.Msg().GetSync(); sync != nil {
				if sync.GetSourceTabId() != "files:1" || sync.GetPath() != "notes/Orbits.md" || sync.GetTitle() != "Orbits" {
					t.Errorf("the sync is %+v", sync)
				}
				break
			}
		}
	}
}

func TestDetachTabKeepsGroupInNewWindow(t *testing.T) {
	state := &State{DetachTab: func(context.Context, DetachedTab) error { return nil }}
	client := newTabLinksClient(t, state)
	openWatch(t, client, "window-a")
	openWatch(t, client, "window-b")
	writeTabs(t, client, "window-a",
		&v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"},
		&v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"})
	linkTabs(t, client, "plex:1", "plex:2")

	if _, err := client.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{Tab: &v1.Tab{Id: "plex:2", Kind: "plex"}})); err != nil {
		t.Fatal(err)
	}
	writeTabs(t, client, "window-a", &v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"})
	writeTabs(t, client, "window-b", &v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"})

	got := state.TabLinks.GetSnapshot()
	if len(got.Groups) != 1 || len(got.Groups[0].TabIDs) != 2 {
		t.Errorf("groups are %+v, want plex:1 and plex:2 in one", got.Groups)
	}
}

func TestCancelWatchTabLinksUnlinksWindowTabsAfterGrace(t *testing.T) {
	state := &State{}
	expiries := make(chan func(), 8)
	client := serveTabLinks(t, state, func(_ time.Duration, f func()) func() bool {
		select {
		case expiries <- f:
		default:
		}
		return func() bool { return false }
	})
	ctx, cancel := context.WithTimeout(t.Context(), testsupport.Patience)
	defer cancel()
	stream, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: "window-b"}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	if !stream.Receive() {
		t.Fatal("the watch sent no snapshot")
	}
	openWatch(t, client, "window-a")
	writeTabs(t, client, "window-a", &v1.LinkableTab{Id: "plex:1", Kind: "plex"})
	writeTabs(t, client, "window-b", &v1.LinkableTab{Id: "files:1", Kind: "files"})
	linkTabs(t, client, "plex:1", "files:1")
	readLinksUntil(t, stream, hasGroupOf(2))

	if _, err := client.CancelWatchTabLinks(ctx, connect.NewRequest(&v1.CancelWatchTabLinksRequest{WindowId: "window-b"})); err != nil {
		t.Fatal(err)
	}

	queued := 0
	for stream.Receive() {
		queued++
	}
	if err := stream.Err(); err != nil {
		t.Errorf("the stream ended with %v after %d messages", err, queued)
	}
	select {
	case expire := <-expiries:
		if len(state.TabLinks.GetSnapshot().Groups) != 1 {
			t.Fatal("the tabs of window-b left their group before the grace period ended")
		}
		expire()
	case <-ctx.Done():
		t.Fatal("the end of the watch scheduled no removal of window-b")
	}
	if groups := state.TabLinks.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestWatchTabLinksRequiresAWindowID(t *testing.T) {
	client := newTabLinksClient(t, &State{})

	stream, err := client.WatchTabLinks(t.Context(), connect.NewRequest(&v1.WatchTabLinksRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	if stream.Receive() {
		t.Fatal("a watch with no window id was served")
	}
	if connect.CodeOf(stream.Err()) != connect.CodeInvalidArgument {
		t.Errorf("got %v, want invalid argument", stream.Err())
	}
}

func TestTabLinksReturnUnimplementedWithoutRegistry(t *testing.T) {
	api := &workspaceHandler{state: &State{}}

	_, err := api.WriteTabLinks(t.Context(), connect.NewRequest(&v1.WriteTabLinksRequest{WindowId: "window-a"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("got %v, want unimplemented", err)
	}
}

// openWatch opens a tab links watch for a window until the test ends and
// returns after the first snapshot.
func openWatch(t *testing.T, client numenv1connect.WorkspaceServiceClient, windowID string) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	stream, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: windowID}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, stream) })
	if !stream.Receive() {
		t.Fatalf("the watch of %s sent no snapshot: %v", windowID, stream.Err())
	}
}

func TestFailedDetachTabDropsTabClosedInItsWindow(t *testing.T) {
	errNoWindow := errors.New("no window opened")
	state := &State{DetachTab: func(context.Context, DetachedTab) error { return errNoWindow }}
	client := newTabLinksClient(t, state)
	openWatch(t, client, "window-a")
	writeTabs(t, client, "window-a",
		&v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"},
		&v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"})
	linkTabs(t, client, "plex:1", "plex:2")

	_, err := client.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{Tab: &v1.Tab{Id: "plex:2", Kind: "plex"}}))
	if err == nil {
		t.Fatal("the detach succeeded with no window opened")
	}
	writeTabs(t, client, "window-a", &v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"})

	if groups := state.TabLinks.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none once plex:2 closed", groups)
	}
	if tab, ok := state.TabLinks.ReadDetached("plex:2"); ok {
		t.Errorf("the failed detach kept %+v", tab)
	}
}

// watchAs opens a tab links stream for a window until the test ends, and
// returns it after the first snapshot.
func watchAs(
	t *testing.T, client numenv1connect.WorkspaceServiceClient, windowID string,
) *connect.ServerStreamForClient[v1.WatchTabLinksResponse] {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), testsupport.Patience)
	t.Cleanup(cancel)
	stream, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: windowID}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, stream) })
	if !stream.Receive() {
		t.Fatalf("the watch of %s sent no snapshot: %v", windowID, stream.Err())
	}
	return stream
}

func TestMoveTabSendsTheTabToTheTargetAndRespondsAfterAccept(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	openWatch(t, client, "window-a")
	target := watchAs(t, client, "window-b")
	tab := &v1.Tab{Id: "plex:2", Kind: "plex", Path: "notes/Orbits.md", Title: "Orbits", State: `{"focus":"Kepler"}`}

	answer := make(chan *v1.MoveTabResponse, 1)
	go func() {
		response, err := client.MoveTab(t.Context(), connect.NewRequest(&v1.MoveTabRequest{
			WindowId: "window-a", TargetWindowId: "window-b", Tab: tab,
		}))
		if err != nil {
			t.Error(err)
		}
		answer <- response.Msg
	}()

	var moved *v1.Tab
	for moved == nil && target.Receive() {
		moved = target.Msg().GetMovedTab()
	}
	if moved.GetId() != "plex:2" || moved.GetState() != tab.GetState() || moved.GetPath() != tab.GetPath() {
		t.Fatalf("the target received %+v", moved)
	}
	accepted, err := client.AcceptTabMove(t.Context(), connect.NewRequest(&v1.AcceptTabMoveRequest{WindowId: "window-b", TabId: "plex:2"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := accepted.Msg.GetError(); got != v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED {
		t.Errorf("AcceptTabMove answered %v", got)
	}
	select {
	case response := <-answer:
		if got := response.GetError(); got != v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED {
			t.Errorf("MoveTab answered %v", got)
		}
	case <-time.After(testsupport.Patience):
		t.Fatal("MoveTab did not answer")
	}
}

func TestMoveTabToAWindowWithoutAWatchReturnsWindowClosed(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	openWatch(t, client, "window-a")

	response, err := client.MoveTab(t.Context(), connect.NewRequest(&v1.MoveTabRequest{
		WindowId: "window-a", TargetWindowId: "window-b", Tab: &v1.Tab{Id: "plex:2", Kind: "plex"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetError(); got != v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_WINDOW_CLOSED {
		t.Errorf("MoveTab answered %v, want window closed", got)
	}
}

func TestMoveTabRequiresBothWindowIDs(t *testing.T) {
	client := newTabLinksClient(t, &State{})

	_, err := client.MoveTab(t.Context(), connect.NewRequest(&v1.MoveTabRequest{WindowId: "window-a"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got %v, want invalid argument", err)
	}
}

func TestReadDetachedTabReturnsTheTabWithItsState(t *testing.T) {
	client := newTabLinksClient(t, &State{DetachTab: func(context.Context, DetachedTab) error { return nil }})
	tab := &v1.Tab{Id: "plex:2", Kind: "plex", Path: "notes/Orbits.md", Title: "Orbits", State: `{"focus":"Kepler"}`}

	if _, err := client.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{Tab: tab})); err != nil {
		t.Fatal(err)
	}
	read, err := client.ReadDetachedTab(t.Context(), connect.NewRequest(&v1.ReadDetachedTabRequest{TabId: "plex:2"}))
	if err != nil {
		t.Fatal(err)
	}
	got := read.Msg.GetTab()
	if got.GetKind() != "plex" || got.GetPath() != tab.GetPath() || got.GetTitle() != "Orbits" || got.GetState() != tab.GetState() {
		t.Errorf("the detached tab is %+v, want %+v", got, tab)
	}

	missing, err := client.ReadDetachedTab(t.Context(), connect.NewRequest(&v1.ReadDetachedTabRequest{TabId: "plex:9"}))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Msg.GetTab() != nil {
		t.Errorf("ReadDetachedTab returned %+v for a tab with no pending detach", missing.Msg.GetTab())
	}
}

func TestHighlightWindowSendsAnEventOnTheStreamOfThatWindow(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	target := watchAs(t, client, "window-b")

	if _, err := client.HighlightWindow(t.Context(), connect.NewRequest(&v1.HighlightWindowRequest{WindowId: "window-b", IsOn: true})); err != nil {
		t.Fatal(err)
	}

	for target.Receive() {
		if highlight := target.Msg().GetHighlight(); highlight != nil {
			if !highlight.GetIsOn() {
				t.Error("the highlight is off, want on")
			}
			return
		}
	}
	t.Fatalf("the stream ended before the highlight: %v", target.Err())
}

func TestSnapshotContainsTheWindowReports(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	stream := watchAs(t, client, "window-a")
	_, err := client.WriteTabLinks(t.Context(), connect.NewRequest(&v1.WriteTabLinksRequest{
		WindowId: "window-a",
		Change: &v1.WriteTabLinksRequest_Tabs{Tabs: &v1.WindowTabs{
			Window: &v1.TabWindow{IsMain: true, ActiveTitle: "Orbits", ActiveKind: "note", TabCount: 3},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	links := readLinksUntil(t, stream, func(links *v1.TabLinks) bool { return len(links.GetWindows()) == 1 })
	got := links.GetWindows()[0]
	if got.GetId() != "window-a" || !got.GetIsMain() || got.GetActiveTitle() != "Orbits" || got.GetActiveKind() != "note" || got.GetTabCount() != 3 {
		t.Errorf("the window is %+v", got)
	}
}

func TestConvertMoveErrorToWireCoversEveryTabMoveErrorCode(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.TabMoveErrorCode]tablinks.MoveError{
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_WINDOW_CLOSED: tablinks.MoveErrorWindowClosed,
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_TIMEOUT:       tablinks.MoveErrorTimeout,
	}, convertMoveErrorToWire)
}

func TestWatchTabLinksSendsTheGroupPathInTheSnapshot(t *testing.T) {
	client := newTabLinksClient(t, &State{})
	ctx, cancel := context.WithTimeout(t.Context(), testsupport.Patience)
	defer cancel()
	openWatch(t, client, "window-a")
	writeTabs(t, client, "window-a",
		&v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"},
		&v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"},
	)
	linkTabs(t, client, "plex:1", "plex:2")
	_, err := client.SyncTabPath(ctx, connect.NewRequest(&v1.SyncTabPathRequest{
		SourceTabId: "plex:2", Path: "notes/Tides.md", Title: "Tides",
	}))
	if err != nil {
		t.Fatal(err)
	}

	stream, err := client.WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: "window-b"}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	group := readLinksUntil(t, stream, hasGroupOf(2)).GetGroups()[0]

	if group.GetPath() != "notes/Tides.md" || group.GetTitle() != "Tides" {
		t.Errorf("the snapshot has path %q and title %q, want notes/Tides.md and Tides", group.GetPath(), group.GetTitle())
	}
}
