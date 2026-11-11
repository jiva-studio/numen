package workspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

func newState() *editor.State { return editor.NewState(run.NewTasks(), nil, nil) }

func scheduleNothing(time.Duration, func()) func() bool {
	return func() bool { return true }
}

func newTabLinksClient(t *testing.T, state *editor.State) numenv1connect.WorkspaceServiceClient {
	t.Helper()
	client, _ := serveTabLinks(t, state, scheduleNothing)
	return client
}

func newHandler(t *testing.T, state *editor.State, schedule func(time.Duration, func()) func() bool) (*Handler, *tablinks.Tabs) {
	t.Helper()
	links := tablinks.New(LinkGrace, schedule, newLinkEvents())
	store := memoryTabs{}
	return NewHandler(state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links), links
}

func serveTabLinks(
	t *testing.T, state *editor.State, schedule func(time.Duration, func()) func() bool,
) (numenv1connect.WorkspaceServiceClient, *tablinks.Tabs) {
	t.Helper()
	handler, links := newHandler(t, state, schedule)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewWorkspaceServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewWorkspaceServiceClient(server.Client(), server.URL), links
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
	client := newTabLinksClient(t, newState())
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

func newDetachingState() (*editor.State, chan struct{}) {
	windowRequests := make(chan struct{}, 1)
	state := newState()
	state.DetachTab = func(context.Context, window.DetachedTab) error {
		windowRequests <- struct{}{}
		return nil
	}
	return state, windowRequests
}

func detachInBackground(
	t *testing.T, client numenv1connect.WorkspaceServiceClient, windowRequests <-chan struct{}, tab *v1.Tab,
) <-chan *connect.Response[v1.DetachTabResponse] {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	answer := make(chan *connect.Response[v1.DetachTabResponse], 1)
	go func() {
		response, err := client.DetachTab(ctx, connect.NewRequest(&v1.DetachTabRequest{Tab: tab}))
		if err != nil {
			if ctx.Err() == nil {
				t.Error(err)
			}
			return
		}
		answer <- response
	}()
	select {
	case <-windowRequests:
	case <-time.After(testsupport.Patience):
		t.Fatal("DetachTab created no window")
	}
	return answer
}

func TestDetachTabRespondsAfterTheNewWindowAcceptsTheTab(t *testing.T) {
	state, windowRequests := newDetachingState()
	client, links := serveTabLinks(t, state, scheduleNothing)
	openWatch(t, client, "window-a")
	openWatch(t, client, "window-b")
	writeTabs(t, client, "window-a",
		&v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"},
		&v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"})
	linkTabs(t, client, "plex:1", "plex:2")

	answer := detachInBackground(t, client, windowRequests, &v1.Tab{Id: "plex:2", Kind: "plex"})
	read, err := client.ReadDetachedTab(t.Context(), connect.NewRequest(&v1.ReadDetachedTabRequest{WindowId: "window-b", TabId: "plex:2"}))
	if err != nil {
		t.Fatal(err)
	}
	writeTabs(t, client, "window-b", &v1.LinkableTab{Id: "plex:2", Kind: "plex", Title: "Tides"})
	select {
	case <-answer:
		t.Fatal("DetachTab responded before the new window accepted the tab")
	default:
	}
	accepted, err := client.AcceptTabMove(t.Context(), connect.NewRequest(&v1.AcceptTabMoveRequest{
		WindowId: "window-b", TabId: "plex:2", MoveId: read.Msg.GetMoveId(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := accepted.Msg.GetError(); got != v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("AcceptTabMove answered %v", got)
	}

	select {
	case response := <-answer:
		if got := response.Msg.GetError(); got != v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED {
			t.Errorf("DetachTab answered %v", got)
		}
	case <-time.After(testsupport.Patience):
		t.Fatal("DetachTab did not answer")
	}
	writeTabs(t, client, "window-a", &v1.LinkableTab{Id: "plex:1", Kind: "plex", Title: "Orbits"})
	got := links.Windows.GetSnapshot()
	if len(got.Groups) != 1 || len(got.Groups[0].TabIDs) != 2 {
		t.Errorf("groups are %+v, want plex:1 and plex:2 in one", got.Groups)
	}
}

func TestDetachTabEndsTheDetachWhenTheRequestIsCanceled(t *testing.T) {
	state := newState()
	state.DetachTab = func(context.Context, window.DetachedTab) error { return nil }
	api, links := newHandler(t, state, scheduleNothing)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := api.DetachTab(ctx, connect.NewRequest(&v1.DetachTabRequest{Tab: &v1.Tab{Id: "plex:2", Kind: "plex"}}))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v, want canceled", err)
	}
	if tab, _, ok := links.Transfers.ReadDetached("window-b", "plex:2"); ok {
		t.Errorf("the canceled detach kept %+v", tab)
	}
}

func TestMoveTabEndsTheMoveWhenTheRequestIsCanceled(t *testing.T) {
	api, links := newHandler(t, newState(), scheduleNothing)
	target := links.Windows.Watch("window-b")
	t.Cleanup(target.Close)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := api.MoveTab(ctx, connect.NewRequest(&v1.MoveTabRequest{
		WindowId: "window-a", TargetWindowId: "window-b", Tab: &v1.Tab{Id: "plex:2", Kind: "plex"},
	}))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("got %v, want canceled", err)
	}
	var moveID string
	for moveID == "" {
		select {
		case event := <-target.Events:
			if event.Move != nil {
				moveID = event.Move.MoveID
			}
		case <-time.After(testsupport.Patience):
			t.Fatal("the target received no move")
		}
	}
	if got := links.Transfers.Accept("window-b", "plex:2", moveID); got != tablink.MoveErrorTimeout {
		t.Errorf("Accept after the canceled request returned %v, want timeout", got)
	}
}

func TestCancelWatchTabLinksUnlinksWindowTabsAfterGrace(t *testing.T) {
	expiries := make(chan func(), 8)
	client, links := serveTabLinks(t, newState(), func(_ time.Duration, f func()) func() bool {
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
		if len(links.Windows.GetSnapshot().Groups) != 1 {
			t.Fatal("the tabs of window-b left their group before the grace period ended")
		}
		expire()
	case <-ctx.Done():
		t.Fatal("the end of the watch scheduled no removal of window-b")
	}
	if groups := links.Windows.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none", groups)
	}
}

func TestWatchTabLinksRequiresAWindowID(t *testing.T) {
	client := newTabLinksClient(t, newState())

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
	state := newState()
	state.DetachTab = func(context.Context, window.DetachedTab) error { return errNoWindow }
	client, links := serveTabLinks(t, state, scheduleNothing)
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

	if groups := links.Windows.GetSnapshot().Groups; len(groups) != 0 {
		t.Errorf("groups are %+v, want none once plex:2 closed", groups)
	}
	if tab, _, ok := links.Transfers.ReadDetached("window-b", "plex:2"); ok {
		t.Errorf("the failed detach kept %+v", tab)
	}
}

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
	client := newTabLinksClient(t, newState())
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

	var event *v1.MovedTab
	for event == nil && target.Receive() {
		event = target.Msg().GetMovedTab()
	}
	moved := event.GetTab()
	if moved.GetId() != "plex:2" || moved.GetState() != tab.GetState() || moved.GetPath() != tab.GetPath() || event.GetMoveId() == "" {
		t.Fatalf("the target received %+v", event)
	}
	accepted, err := client.AcceptTabMove(t.Context(), connect.NewRequest(&v1.AcceptTabMoveRequest{
		WindowId: "window-b", TabId: "plex:2", MoveId: event.GetMoveId(),
	}))
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
	client := newTabLinksClient(t, newState())
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
	client := newTabLinksClient(t, newState())

	_, err := client.MoveTab(t.Context(), connect.NewRequest(&v1.MoveTabRequest{WindowId: "window-a"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got %v, want invalid argument", err)
	}
}

func TestReadDetachedTabReturnsTheTabWithItsState(t *testing.T) {
	tab := &v1.Tab{Id: "plex:2", Kind: "plex", Path: "notes/Orbits.md", Title: "Orbits", State: `{"focus":"Kepler"}`}
	state, windowRequests := newDetachingState()
	client := newTabLinksClient(t, state)
	detachInBackground(t, client, windowRequests, tab)
	read, err := client.ReadDetachedTab(t.Context(), connect.NewRequest(&v1.ReadDetachedTabRequest{WindowId: "window-b", TabId: "plex:2"}))
	if err != nil {
		t.Fatal(err)
	}
	got := read.Msg.GetTab()
	if got.GetKind() != "plex" || got.GetPath() != tab.GetPath() || got.GetTitle() != "Orbits" || got.GetState() != tab.GetState() || read.Msg.GetMoveId() == "" {
		t.Errorf("the detached tab is %+v, want %+v", got, tab)
	}

	missing, err := client.ReadDetachedTab(t.Context(), connect.NewRequest(&v1.ReadDetachedTabRequest{WindowId: "window-b", TabId: "plex:9"}))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Msg.GetTab() != nil {
		t.Errorf("ReadDetachedTab returned %+v for a tab with no pending detach", missing.Msg.GetTab())
	}
}

func TestHighlightWindowSendsAnEventOnTheStreamOfThatWindow(t *testing.T) {
	client := newTabLinksClient(t, newState())
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
	client := newTabLinksClient(t, newState())
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
	testsupport.CheckProduced(t, map[v1.TabMoveErrorCode]tablink.MoveError{
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_WINDOW_CLOSED: tablink.MoveErrorWindowClosed,
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_TIMEOUT:       tablink.MoveErrorTimeout,
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_NOT_OWNER:     tablink.MoveErrorNotOwner,
		v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_CANCELED:      tablink.MoveErrorCancelled,
	}, convertMoveErrorToWire)
}

func TestWatchTabLinksSendsTheGroupPathInTheSnapshot(t *testing.T) {
	client := newTabLinksClient(t, newState())
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
