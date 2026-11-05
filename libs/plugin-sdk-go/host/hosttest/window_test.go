package hosttest_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// newWindowVault is a vault of one note and one PDF, and its window.
func newWindowVault() *hosttest.Window {
	vault := hosttest.New("marsh")
	vault.File("notes/Heron.md", "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: Grey Heron\n---\nHerons fish at dusk.")
	vault.File("library/Birds.pdf", "%PDF-1.7\n\x00\x01binary body")
	return hosttest.NewWindow(vault)
}

func TestTheWindowFocusesANoteOfTheVaultAndRecordsIt(t *testing.T) {
	window := newWindowVault()

	got, err := window.FocusNote(t.Context(), "notes/Heron.md")
	if err != nil {
		t.Fatal(err)
	}
	want := host.FocusedNote{Path: "notes/Heron.md", Title: "Grey Heron", ID: "01H7X1234567890ABCDEFGHJKM"}
	if got != want {
		t.Errorf("focused %+v, want %+v", got, want)
	}
	if focused := window.GetFocusedNotes(); !slices.Equal(focused, []string{"notes/Heron.md"}) {
		t.Errorf("the window recorded %v", focused)
	}
}

func TestTheWindowReturnsErrNotExistForAPathThatIsNoNote(t *testing.T) {
	window := newWindowVault()

	for _, path := range []string{"notes/Gone.md", "library/Birds.pdf"} {
		if _, err := window.FocusNote(t.Context(), path); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("FocusNote of %s got %v, want ErrNotExist", path, err)
		}
	}
	if focused := window.GetFocusedNotes(); len(focused) != 0 {
		t.Errorf("the window recorded %v for notes it does not hold", focused)
	}
}

func TestTheWindowOpensAFileAtThePassagesItIsGivenAndRecordsThem(t *testing.T) {
	window := newWindowVault()
	passages := []host.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}}

	summary, err := window.FocusPassages(t.Context(), "library/Birds.pdf", passages)
	if err != nil {
		t.Fatal(err)
	}
	if summary != "the document is open at that passage, with the words of it lit" {
		t.Errorf("the summary is %q", summary)
	}
	want := []hosttest.FocusRequest{{Path: "library/Birds.pdf", Passages: passages}}
	if got := window.GetFocusRequests(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window recorded %+v, want %+v", got, want)
	}
}

func TestTheWindowSummaryNamesABeginningAndANote(t *testing.T) {
	window := newWindowVault()

	beginning, err := window.FocusPassages(t.Context(), "library/Birds.pdf", nil)
	if err != nil || beginning != "the document is open at its beginning" {
		t.Errorf("a file with no passages answered %q (%v)", beginning, err)
	}
	note, err := window.FocusPassages(t.Context(), "notes/Heron.md", []host.PassageSpan{{Start: 0, Length: 6}})
	if err != nil || note != "the note is in front of them, and a note is shown whole" {
		t.Errorf("a note answered %q (%v)", note, err)
	}
}

func TestTheWindowOpensNoFileTheVaultDoesNotHold(t *testing.T) {
	window := newWindowVault()

	if _, err := window.FocusPassages(t.Context(), "library/Gone.pdf", nil); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("FocusPassages of a missing file got %v, want ErrNotExist", err)
	}
	if got := window.GetFocusRequests(); len(got) != 0 {
		t.Errorf("the window recorded %+v for a file it does not hold", got)
	}
}

func TestTheWindowServesTheTabsItWasGiven(t *testing.T) {
	window := newWindowVault()

	if tabs, activeID, err := window.ReadOpenTabs(t.Context()); err != nil || len(tabs) != 0 || activeID != "" {
		t.Errorf("a new window has %+v open with %q active (%v)", tabs, activeID, err)
	}
	open := []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Grey Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4}`},
	}
	window.SetOpenTabs(open, "tab-2")

	tabs, activeID, err := window.ReadOpenTabs(t.Context())
	if err != nil || !slices.Equal(tabs, open) || activeID != "tab-2" {
		t.Errorf("the window has %+v open with %q active (%v), want %+v with tab-2", tabs, activeID, err, open)
	}
}

func TestTheWindowSetToNoWindowAnswersErrNoWindow(t *testing.T) {
	window := newWindowVault()
	window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: "notes/Heron.md"}}, "tab-1")
	window.SetNoWindow()
	ctx := t.Context()

	if _, err := window.FocusNote(ctx, "notes/Heron.md"); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("FocusNote got %v, want ErrNoWindow", err)
	}
	if _, err := window.FocusPassages(ctx, "library/Birds.pdf", nil); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("FocusPassages got %v, want ErrNoWindow", err)
	}
	if _, _, err := window.ReadOpenTabs(ctx); !errors.Is(err, host.ErrNoWindow) {
		t.Errorf("ReadOpenTabs got %v, want ErrNoWindow", err)
	}
	if len(window.GetFocusedNotes()) != 0 || len(window.GetFocusRequests()) != 0 {
		t.Errorf("a host with no window recorded %v and %+v", window.GetFocusedNotes(), window.GetFocusRequests())
	}
}
