package editor

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

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
