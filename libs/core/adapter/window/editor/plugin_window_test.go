package editor

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// newBoundWindow is a state showing a vault of one PDF and one note, the window bound to it, and
// what a WatchFocus subscriber of the state receives.
func newBoundWindow(t *testing.T) (*State, pluginport.PluginWindow, <-chan domain.Place) {
	t.Helper()
	state := &State{Places: newPlaceBroadcaster()}
	state.setVault(testsupport.NewVault(t, map[string]string{
		"library/Birds.pdf": "%PDF-1.7 a guide to the birds of the marsh",
		"notes/Heron.md":    "# Grey Heron\nHerons fish at dusk.\n",
	}))
	line, unsubscribe := state.Places.subscribe()
	t.Cleanup(unsubscribe)
	return state, NewPluginWindow(state, nil, vaultreader.Readers{}), line
}

func TestPluginWindowFocusPassagesReachesTheWindowWithTheRanges(t *testing.T) {
	_, window, line := newBoundWindow(t)
	ranges := []domain.ByteRange{{Start: 1500, End: 1580}, {Start: 400, End: 1100}}

	if err := window.FocusPassages(t.Context(), "library/Birds.pdf", ranges); err != nil {
		t.Fatal(err)
	}
	select {
	case place := <-line:
		want := domain.Place{Path: "library/Birds.pdf", Ranges: ranges}
		if !reflect.DeepEqual(place, want) {
			t.Errorf("the window received %+v, want %+v", place, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the window received no place")
	}
}

func TestPluginWindowRefusesAnInvalidPassageAndAMissingFileAndDoesNotMoveTheWindow(t *testing.T) {
	_, window, line := newBoundWindow(t)

	invalid := window.FocusPassages(t.Context(), "library/Birds.pdf", []domain.ByteRange{{Start: -1, End: 4}})
	if !errors.Is(invalid, pluginport.ErrInvalidFocus) {
		t.Errorf("a passage before the text answered %v, want ErrInvalidFocus", invalid)
	}
	_, emptyPath := window.FocusNote(t.Context(), "")
	if !errors.Is(emptyPath, pluginport.ErrInvalidFocus) {
		t.Errorf("a note with no path answered %v, want ErrInvalidFocus", emptyPath)
	}
	missing := window.FocusPassages(t.Context(), "library/Gone.pdf", nil)
	if !errors.Is(missing, fs.ErrNotExist) {
		t.Errorf("a missing file answered %v, want fs.ErrNotExist", missing)
	}
	select {
	case place := <-line:
		t.Errorf("the window was moved to %+v", place)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPluginWindowReadsTheTabsTheWindowLastReported(t *testing.T) {
	state, window, _ := newBoundWindow(t)
	api := &workspaceHandler{state: state}

	if _, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		ActiveId: "tab-2",
		Tabs: []*v1.Tab{
			{Id: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Grey Heron"},
			{Id: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4}`},
		},
	})); err != nil {
		t.Fatal(err)
	}

	got, err := window.ReadOpenTabs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := domain.OpenTabs{
		ActiveID: "tab-2",
		Tabs: []domain.Tab{
			{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Grey Heron"},
			{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4}`},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the window has %+v open, want %+v", got, want)
	}
}

// activateFixture is a bound window whose state serves WatchFocus over HTTP, and the focus requests
// one subscriber of that stream receives.
type activateFixture struct {
	state  *State
	api    *workspaceHandler
	window pluginport.PluginWindow
	focus  <-chan *v1.WatchFocusResponse
}

// newActivateFixture is a bound window with the tabs tab-1 (active) and tab-2 of one PDF and a note
// reported, and a WatchFocus subscriber that is known to be subscribed.
func newActivateFixture(t *testing.T) activateFixture {
	t.Helper()
	state, window, _ := newBoundWindow(t)
	api := &workspaceHandler{state: state}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewWorkspaceServiceHandler(api))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	listening, stop := context.WithCancel(t.Context())
	t.Cleanup(stop)
	stream, err := numenv1connect.NewWorkspaceServiceClient(http.DefaultClient, server.URL).
		WatchFocus(listening, connect.NewRequest(&v1.WatchFocusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	focus := make(chan *v1.WatchFocusResponse, 16)
	go func() {
		defer close(focus)
		for stream.Receive() {
			focus <- stream.Msg()
		}
	}()
	f := activateFixture{state: state, api: api, window: window, focus: focus}
	f.waitSubscribed(t)
	f.reportTabs(t, "tab-1")
	return f
}

// waitSubscribed requests a place until the stream delivers one, so the stream handler is subscribed
// before the test asks for anything else.
func (f activateFixture) waitSubscribed(t *testing.T) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		if err := f.state.GetWindow().FocusPlace(t.Context(), domain.Place{Path: "notes/Heron.md"}); err != nil {
			t.Fatal(err)
		}
		select {
		case msg := <-f.focus:
			if msg.GetPath() == "notes/Heron.md" {
				return
			}
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatal("the focus stream delivered no place")
		}
	}
}

// reportTabs reports the tabs of the window with the active id, as the window does on every change.
func (f activateFixture) reportTabs(t *testing.T, activeID string) {
	t.Helper()
	if _, err := f.api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		ActiveId: activeID,
		Tabs: []*v1.Tab{
			{Id: "tab-1", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4,"pageCount":80}`},
			{Id: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":61,"pageCount":80}`},
			{Id: "tab-3", Kind: "note", Path: "notes/Heron.md", Title: "Grey Heron"},
		},
	})); err != nil {
		t.Fatal(err)
	}
}

// receiveTabID is the next focus request that names a tab id, or nil when none arrives within the
// wait. Keepalives and places are skipped.
func (f activateFixture) receiveTabID(t *testing.T, wait time.Duration) *v1.WatchFocusResponse {
	t.Helper()
	deadline := time.After(wait)
	for {
		select {
		case msg, isOpen := <-f.focus:
			if !isOpen {
				t.Fatal("the focus stream ended")
			}
			if msg.GetTabId() != "" {
				return msg
			}
		case <-deadline:
			return nil
		}
	}
}

// activationResult is what one ActivateTab call returned.
type activationResult struct {
	open domain.OpenTabs
	err  error
}

// startActivation runs ActivateTab of the id in the background.
func (f activateFixture) startActivation(ctx context.Context, id string) <-chan activationResult {
	results := make(chan activationResult, 1)
	go func() {
		open, err := f.window.ActivateTab(ctx, id)
		results <- activationResult{open: open, err: err}
	}()
	return results
}

func TestPluginWindowActivateTabWaitsForTheReportOfTheActiveTab(t *testing.T) {
	f := newActivateFixture(t)
	results := f.startActivation(t.Context(), "tab-2")

	msg := f.receiveTabID(t, 5*time.Second)
	if msg == nil {
		t.Fatal("the focus stream received no tab id")
	}
	if msg.GetTabId() != "tab-2" || msg.GetPath() != "" || len(msg.GetRanges()) != 0 {
		t.Errorf("the focus stream received %v, want the tab id tab-2 alone", msg)
	}
	select {
	case got := <-results:
		t.Fatalf("ActivateTab returned %+v before the window reported the tab active", got)
	case <-time.After(100 * time.Millisecond):
	}

	f.reportTabs(t, "tab-2")
	select {
	case got := <-results:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.open.ActiveID != "tab-2" || len(got.open.Tabs) != 3 {
			t.Errorf("ActivateTab returned %+v, want the reported tabs with tab-2 active", got.open)
		}
	case <-time.After(time.Second):
		t.Fatal("ActivateTab did not return within a second of the report that made the tab active")
	}
}

func TestPluginWindowActivateTabOfAnUnknownIDIsErrNotExist(t *testing.T) {
	f := newActivateFixture(t)

	select {
	case got := <-f.startActivation(t.Context(), "tab-9"):
		if !errors.Is(got.err, fs.ErrNotExist) {
			t.Errorf("ActivateTab of tab-9 returned %v, want fs.ErrNotExist", got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("ActivateTab of an id no tab has did not return at once")
	}
	if msg := f.receiveTabID(t, 100*time.Millisecond); msg != nil {
		t.Errorf("the focus stream received %v", msg)
	}
}

func TestPluginWindowActivateTabOfTheActiveTabPublishesIt(t *testing.T) {
	f := newActivateFixture(t)

	select {
	case got := <-f.startActivation(t.Context(), "tab-1"):
		if got.err != nil || got.open.ActiveID != "tab-1" || len(got.open.Tabs) != 3 {
			t.Errorf("ActivateTab of the active tab returned %+v (%v), want the reported tabs with tab-1 active", got.open, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("ActivateTab of the active tab did not return at once")
	}
	msg := f.receiveTabID(t, 5*time.Second)
	if msg == nil {
		t.Fatal("the focus stream received no tab id")
	}
	if msg.GetTabId() != "tab-1" || msg.GetPath() != "" || len(msg.GetRanges()) != 0 {
		t.Errorf("the focus stream received %v, want the tab id tab-1 alone", msg)
	}
}

func TestPluginWindowActivateTabReturnsTheLastReportAfterItsBound(t *testing.T) {
	f := newActivateFixture(t)
	f.state.TabActivationBound = 50 * time.Millisecond
	results := f.startActivation(t.Context(), "tab-2")
	if msg := f.receiveTabID(t, 5*time.Second); msg == nil {
		t.Fatal("the focus stream received no tab id")
	}

	select {
	case got := <-results:
		if got.err != nil || got.open.ActiveID != "tab-1" || len(got.open.Tabs) != 3 {
			t.Errorf("ActivateTab returned %+v (%v), want the last reported tabs with tab-1 active", got.open, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("ActivateTab did not return within a second when its bound is 50 ms")
	}
}

func TestPluginWindowActivateTabReturnsWhenTheContextEnds(t *testing.T) {
	f := newActivateFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	results := f.startActivation(ctx, "tab-2")
	if msg := f.receiveTabID(t, 5*time.Second); msg == nil {
		t.Fatal("the focus stream received no tab id")
	}
	cancel()

	select {
	case <-results:
	case <-time.After(time.Second):
		t.Fatal("ActivateTab did not return within a second of the end of its context")
	}
}
