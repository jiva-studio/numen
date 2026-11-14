package hosttest_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func newWindowVault() *hosttest.Window {
	vault := hosttest.New("marsh")
	vault.File("notes/Heron.md", "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: Gray Heron\n---\nHerons fish at dusk.")
	vault.File("library/Birds.pdf", "%PDF-1.7\n\x00\x01binary body")
	return hosttest.NewWindow(vault)
}

func TestTheWindowFocusesANoteOfTheVaultAndRecordsIt(t *testing.T) {
	window := newWindowVault()

	got, err := window.FocusNote(t.Context(), "notes/Heron.md")
	if err != nil {
		t.Fatal(err)
	}
	want := host.FocusedNote{Path: "notes/Heron.md", Title: "Gray Heron", ID: "01H7X1234567890ABCDEFGHJKM"}
	if got != want {
		t.Errorf("focused %+v, want %+v", got, want)
	}
	if focused := window.GetFocusedNotes(); !slices.Equal(focused, []string{"notes/Heron.md"}) {
		t.Errorf("the window recorded %v", focused)
	}
}

func TestWindowReturnsErrNotExistForNonNote(t *testing.T) {
	window := newWindowVault()

	for _, path := range []string{"notes/Gone.md", "library/Birds.pdf"} {
		if _, err := window.FocusNote(t.Context(), path); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("FocusNote of %s got %v, want ErrNotExist", path, err)
		}
	}
	if focused := window.GetFocusedNotes(); len(focused) != 0 {
		t.Errorf("the window recorded %v for notes not in the vault", focused)
	}
}

func TestTheWindowOpensAFileAtThePassagesItIsGivenAndRecordsThem(t *testing.T) {
	window := newWindowVault()
	passages := []host.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}}

	if err := window.FocusPassages(t.Context(), "library/Birds.pdf", passages); err != nil {
		t.Fatal(err)
	}
	want := []hosttest.FocusRequest{{Path: "library/Birds.pdf", Passages: passages}}
	if got := window.GetFocusRequests(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window recorded %+v, want %+v", got, want)
	}
}

func TestTheWindowOpensAFileWithNoPassagesAndANote(t *testing.T) {
	window := newWindowVault()

	if err := window.FocusPassages(t.Context(), "library/Birds.pdf", nil); err != nil {
		t.Errorf("a file with no passages returned %v", err)
	}
	if err := window.FocusPassages(t.Context(), "notes/Heron.md", []host.PassageSpan{{Start: 0, Length: 6}}); err != nil {
		t.Errorf("a note returned %v", err)
	}
	want := []hosttest.FocusRequest{{Path: "library/Birds.pdf"}, {Path: "notes/Heron.md", Passages: []host.PassageSpan{{Start: 0, Length: 6}}}}
	if got := window.GetFocusRequests(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window recorded %+v, want %+v", got, want)
	}
}

func TestWindowOpensNoFileNotInVault(t *testing.T) {
	window := newWindowVault()

	if err := window.FocusPassages(t.Context(), "library/Gone.pdf", nil); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("FocusPassages of a missing file got %v, want ErrNotExist", err)
	}
	if got := window.GetFocusRequests(); len(got) != 0 {
		t.Errorf("the window recorded %+v for a file not in the vault", got)
	}
}

func TestTheWindowServesTheTabsItWasGiven(t *testing.T) {
	window := newWindowVault()

	if tabs, activeID, err := window.ReadOpenTabs(t.Context()); err != nil || len(tabs) != 0 || activeID != "" {
		t.Errorf("a new window has %+v open with %q active (%v)", tabs, activeID, err)
	}
	open := []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Gray Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: `{"page":4}`},
	}
	window.SetOpenTabs(open, "tab-2")

	tabs, activeID, err := window.ReadOpenTabs(t.Context())
	if err != nil || !reflect.DeepEqual(tabs, open) || activeID != "tab-2" {
		t.Errorf("the window has %+v open with %q active (%v), want %+v with tab-2", tabs, activeID, err, open)
	}
}

func TestWindowSetToNoWindowReturnsErrNoWindow(t *testing.T) {
	window := newWindowVault()
	window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: "notes/Heron.md"}}, "tab-1")
	window.SetNoWindow()
	ctx := t.Context()

	if _, err := window.FocusNote(ctx, "notes/Heron.md"); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("FocusNote got %v, want ErrNoWindow", err)
	}
	if err := window.FocusPassages(ctx, "library/Birds.pdf", nil); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("FocusPassages got %v, want ErrNoWindow", err)
	}
	if _, _, err := window.ReadOpenTabs(ctx); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("ReadOpenTabs got %v, want ErrNoWindow", err)
	}
	if len(window.GetFocusedNotes()) != 0 || len(window.GetFocusRequests()) != 0 {
		t.Errorf("a host with no window recorded %v and %+v", window.GetFocusedNotes(), window.GetFocusRequests())
	}
}

func getBirdTabs() []host.OpenTab {
	return []host.OpenTab{
		{ID: "tab-1", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: `{"page":4,"pageCount":80}`},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: `{"page":61,"pageCount":80}`},
	}
}

func TestTheWindowActivatesATabItHasAndRecordsTheID(t *testing.T) {
	window := newWindowVault()
	window.SetOpenTabs(getBirdTabs(), "tab-1")

	tab, activeID, err := window.ActivateTab(t.Context(), "tab-2")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tab, getBirdTabs()[1]) || activeID != "tab-2" {
		t.Errorf("activated %+v with %q active, want %+v with tab-2", tab, activeID, getBirdTabs()[1])
	}
	if _, readID, err := window.ReadOpenTabs(t.Context()); err != nil || readID != "tab-2" {
		t.Errorf("the window reads %q as active (%v), want tab-2", readID, err)
	}
	if _, _, err := window.ActivateTab(t.Context(), "tab-1"); err != nil {
		t.Fatal(err)
	}
	if activated := window.GetActivatedTabs(); !slices.Equal(activated, []string{"tab-2", "tab-1"}) {
		t.Errorf("the window recorded %v, want [tab-2 tab-1]", activated)
	}
}

func TestTheWindowReturnsErrNotExistForATabIDItHasNoTabWith(t *testing.T) {
	window := newWindowVault()
	window.SetOpenTabs(getBirdTabs(), "tab-1")

	if _, _, err := window.ActivateTab(t.Context(), "tab-9"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("ActivateTab got %v, want ErrNotExist", err)
	}
	if activated := window.GetActivatedTabs(); len(activated) != 0 {
		t.Errorf("the window recorded %v for a tab it does not have", activated)
	}
	_, activeID, err := window.ReadOpenTabs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if activeID != "tab-1" {
		t.Errorf("the window reads %q as active, want tab-1", activeID)
	}
}

func TestTheWindowSetToNoWindowActivatesNoTab(t *testing.T) {
	window := newWindowVault()
	window.SetOpenTabs(getBirdTabs(), "tab-1")
	window.SetNoWindow()

	if _, _, err := window.ActivateTab(t.Context(), "tab-2"); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("ActivateTab got %v, want ErrNoWindow", err)
	}
	if activated := window.GetActivatedTabs(); len(activated) != 0 {
		t.Errorf("a host with no window recorded %v", activated)
	}
}
