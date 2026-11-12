package workspace

import (
	"context"
	"math"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// memoryTabs keeps the open tabs of each vault in memory.
type memoryTabs map[domain.VaultID]window.OpenTabs

func (m memoryTabs) ReadOpenTabs(_ context.Context, v domain.Vault) (window.OpenTabs, error) {
	return m[v.ID], nil
}

func (m memoryTabs) WriteOpenTabs(_ context.Context, v domain.Vault, open window.OpenTabs) error {
	m[v.ID] = open
	return nil
}

// showing is a window state that shows a vault.
type showing struct {
	*editor.State
	vault domain.Vault
}

func (s showing) GetCurrentVault() domain.Vault { return s.vault }

func newHandlerOver(t *testing.T, window showing, store memoryTabs) *Handler {
	t.Helper()
	links := tablinks.New(LinkGrace, scheduleNothing, newLinkEvents())
	return NewHandler(window, window.Notices, window, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links)
}

// The window reports the tabs that are currently open, and the handler returns
// the latest report.

func TestWriteOpenTabs_StoresLatestTabs(t *testing.T) {
	state := newState()
	api := newHandlerOver(t, showing{State: state}, memoryTabs{})

	if open := state.Tabs.GetOpenTabs(); len(open.Tabs) != 0 || open.ActiveID != "" {
		t.Fatalf("a window that has said nothing has %+v open", open)
	}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		ActiveId: "two",
		Tabs: []*v1.Tab{
			{Id: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{Id: "two", Kind: "reader:player", Path: "Talk.bin", Title: "Talk", State: `{"t":[12]}`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := window.OpenTabs{
		ActiveID: "two",
		Tabs: []window.Tab{
			{ID: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{ID: "two", Kind: "reader:player", Path: "Talk.bin", Title: "Talk", State: `{"t":[12]}`},
		},
	}
	if got := state.Tabs.GetOpenTabs(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window has %+v open", got)
	}
}

func TestReadOpenTabsReturnsTabsSavedForVault(t *testing.T) {
	store := memoryTabs{}
	current := showing{State: newState(), vault: domain.Vault{ID: "v", Path: t.TempDir()}}
	api := newHandlerOver(t, current, store)

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		ActiveId: "one",
		Tabs:     []*v1.Tab{{Id: "one", Kind: "reader:viewer", Path: "a.bin", Title: "a", State: `{"page":3}`}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	// A core started again reads the store, not the tabs held in memory.
	restarted := newHandlerOver(t, showing{State: newState(), vault: current.vault}, store)
	got, err := restarted.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.ReadOpenTabsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetActiveId() != "one" || len(got.Msg.GetTabs()) != 1 || got.Msg.GetTabs()[0].GetState() != `{"page":3}` {
		t.Errorf("read %+v", got.Msg)
	}
}

func TestReadOpenTabsReturnsNoneWithoutAVault(t *testing.T) {
	api := newHandlerOver(t, showing{State: newState()}, memoryTabs{})
	got, err := api.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.ReadOpenTabsRequest{}))
	if err != nil || len(got.Msg.GetTabs()) != 0 {
		t.Errorf("read %+v, %v", got.Msg, err)
	}
}

func TestAttendingTellsWhoeverIsListening(t *testing.T) {
	var received []window.OpenTabs
	state := newState()
	state.NotifyOpenTabs = func(open window.OpenTabs) { received = append(received, open) }
	api := newHandlerOver(t, showing{State: state}, memoryTabs{})

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		ActiveId: "one",
		Tabs:     []*v1.Tab{{Id: "one", Kind: "note", Path: "Entropy.md", Title: "Entropy"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(received) != 1 {
		t.Fatalf("what the window has open was said %d times", len(received))
	}
	front, target := received[0].GetActiveTab()
	if !target || front.Path != "Entropy.md" {
		t.Errorf("the tab in front is %+v", front)
	}
}

func TestFrontedIsTheTabThePersonIsLookingAt(t *testing.T) {
	for name, c := range map[string]struct {
		open window.OpenTabs
		want string
	}{
		"a tab in front": {
			open: window.OpenTabs{
				ActiveID: "two",
				Tabs:     []window.Tab{{ID: "one", Path: "Entropy.md"}, {ID: "two", Path: "Talk.mp3"}},
			},
			want: "Talk.mp3",
		},
		"nothing open":     {open: window.OpenTabs{}},
		"a tab it lets go": {open: window.OpenTabs{ActiveID: "three", Tabs: []window.Tab{{ID: "one"}}}},
		"a tab with no identity": {
			open: window.OpenTabs{Tabs: []window.Tab{{Path: "Entropy.md"}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			front, target := c.open.GetActiveTab()
			if target != (c.want != "") {
				t.Fatalf("a tab in front is %v", target)
			}
			if front.Path != c.want {
				t.Errorf("the tab in front holds %q", front.Path)
			}
		})
	}
}

func TestDetachTabInvokesState(t *testing.T) {
	var got window.DetachedTab
	state := newState()
	api, links := newHandler(t, state, scheduleNothing)
	state.DetachTab = func(_ context.Context, tab window.DetachedTab) error {
		got = tab
		// The new window reads and accepts the tab, so DetachTab responds.
		_, moveID, _ := links.Transfers.ReadDetached("window-b", tab.ID)
		links.Transfers.Accept("window-b", tab.ID, moveID)
		return nil
	}

	_, err := api.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{
		Tab:    &v1.Tab{Id: "plex:7a2e", Kind: "plex", Path: "physics/Ontology.md"},
		X:      150,
		Y:      250,
		Width:  640,
		Height: 480,
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := window.DetachedTab{ID: "plex:7a2e", X: 150, Y: 250, Width: 640, Height: 480}
	if got != want {
		t.Errorf("detached tab: got %+v, want %+v", got, want)
	}
}

func TestDetachTabRequiresATabID(t *testing.T) {
	state := newState()
	state.DetachTab = func(context.Context, window.DetachedTab) error { return nil }
	api, _ := newHandler(t, state, scheduleNothing)

	_, err := api.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{Tab: &v1.Tab{Kind: "plex"}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got code %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestDetachTabFailsWhenUnimplemented(t *testing.T) {
	api, _ := newHandler(t, newState(), scheduleNothing)

	_, err := api.DetachTab(t.Context(), connect.NewRequest(&v1.DetachTabRequest{Tab: &v1.Tab{Id: "plex:1", Kind: "plex"}}))
	if err == nil {
		t.Fatal("expected error when DetachTab is nil, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("got code %v, want %v", connect.CodeOf(err), connect.CodeUnavailable)
	}
}

// The window receives the ranges as 32-bit offsets, so a range that does not fit is left out rather
// than wrapped to another place.
func TestFocusSentToWindowOmitsRangeBeyond32Bits(t *testing.T) {
	got := convertByteRangesToWire([]domain.ByteRange{
		{Start: 400, End: 480},
		{Start: math.MaxInt32 - 5, End: math.MaxInt32 + 5},
		{Start: math.MaxInt32 + 1, End: math.MaxInt32 + 9},
		{Start: math.MaxInt32 - 10, End: math.MaxInt32},
		{Start: 1500, End: 1580},
	})

	want := [][2]int32{{400, 480}, {math.MaxInt32 - 10, math.MaxInt32}, {1500, 1580}}
	ranges := make([][2]int32, 0, len(got))
	for _, one := range got {
		ranges = append(ranges, [2]int32{one.GetStart(), one.GetEnd()})
	}
	if !reflect.DeepEqual(ranges, want) {
		t.Errorf("the window is sent the ranges %v, want %v", ranges, want)
	}
}
