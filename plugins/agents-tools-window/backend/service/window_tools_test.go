package service_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/preview"
)

// The invented vault: a note, a bird guide, a book and a recorded talk.
const (
	testVault = "marsh"
	notePath  = "notes/Heron.md"
	guidePath = "library/Bird Guide.pdf"
	bookPath  = "library/Marsh Book.epub"
	talkPath  = "recordings/Marsh Talk.mp3"
	pdfBytes  = "%PDF-1.7\n\x00\x01binary body"
)

type windowTools struct {
	window   *hosttest.Window
	registry *tools.Registry
}

func newWindowTools(t *testing.T) windowTools {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File(notePath, "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: Gray Heron\n---\nHerons fish at dusk.")
	vault.File(guidePath, pdfBytes)
	vault.File(bookPath, "PK\x03\x04epub body")
	vault.File(talkPath, "ID3\x00\x03audio")

	window := hosttest.NewWindow(vault)
	return windowTools{window: window, registry: newRegistry(t, service.Host{Vault: vault, Window: window})}
}

func newRegistry(t *testing.T, on service.Host) *tools.Registry {
	t.Helper()
	registry := tools.New()
	manager := preview.NewStore(newFakeClock(time.Unix(1_700_000_000, 0)))
	if err := service.RegisterTools(registry, manager, on); err != nil {
		t.Fatal(err)
	}
	return registry
}

func (w windowTools) call(t *testing.T, name, arguments string, callContext map[string]string, out any) (failure string) {
	t.Helper()
	res, err := w.registry.InvokeTool(t.Context(), tools.Call{
		VaultID: testVault, Name: name, Arguments: []byte(arguments), Context: callContext,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("%s gave error code %v", name, res.GetError())
	}
	if res.GetResult().GetIsError() {
		if text := res.GetResult().GetText(); text != "" {
			return text
		}
		return "a tool error with no text"
	}
	if out != nil {
		if err := json.Unmarshal(res.GetResult().GetStructured(), out); err != nil {
			t.Fatalf("%s: %v in %s", name, err, res.GetResult().GetStructured())
		}
	}
	return ""
}

func (w windowTools) getToolInfo(t *testing.T, name string) *tools.ToolInfo {
	t.Helper()
	listed, err := w.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listed.Tools {
		if info.GetName() == name {
			return info
		}
	}
	t.Fatalf("no tool %s is registered", name)
	return nil
}

// tabOutput is one tab as returned by a tool. Place is the raw JSON of the place key, nil where
// the key is absent.
type tabOutput struct {
	TabID    string          `json:"tabId"`
	Kind     string          `json:"kind"`
	Path     string          `json:"path"`
	Title    string          `json:"title"`
	Place    json.RawMessage `json:"place"`
	IsActive bool            `json:"isActive"`
}

type tabsOutput struct {
	Tabs        []tabOutput `json:"tabs"`
	ActiveTabID string      `json:"activeTabId"`
}

func requirePlace(t *testing.T, what string, place json.RawMessage, want string) {
	t.Helper()
	if want == "" {
		if place != nil {
			t.Errorf("%s has the place %s, want none", what, place)
		}
		return
	}
	var got, wanted map[string]any
	if err := json.Unmarshal(place, &got); err != nil {
		t.Errorf("%s has the place %s, want %s", what, place, want)
		return
	}
	if err := json.Unmarshal([]byte(want), &wanted); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("%s has the place %s, want %s", what, place, want)
	}
}

func TestNoteFocusDescriptionExplainsWhenToMoveTheWindow(t *testing.T) {
	w := newWindowTools(t)
	description := w.getToolInfo(t, "note_focus").GetDescription()
	for _, want := range []string{"neighborhood", "asked to be taken to a note", "Talking about one is not asking", "read the note for its content"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description of note_focus does not contain %q: %s", want, description)
		}
	}
	sourceDescription := w.getToolInfo(t, "source_focus").GetDescription()
	for _, want := range []string{"a search of the library returns", "start", "length", "first", "Talking about a passage is not asking"} {
		if !strings.Contains(sourceDescription, want) {
			t.Errorf("the description of source_focus does not contain %q: %s", want, sourceDescription)
		}
	}
}

func TestNoteFocusReturnsTheFocusedNoteAndMovesTheWindowToIt(t *testing.T) {
	w := newWindowTools(t)
	var out struct {
		Note struct {
			Path  string `json:"path"`
			Title string `json:"title"`
			ID    string `json:"id"`
		} `json:"note"`
	}
	if failure := w.call(t, "note_focus", `{"path":"notes/Heron.md"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Note.Path != notePath || out.Note.Title != "Gray Heron" || out.Note.ID != "01H7X1234567890ABCDEFGHJKM" {
		t.Errorf("note %+v", out.Note)
	}
	if focused := w.window.GetFocusedNotes(); !slices.Equal(focused, []string{notePath}) {
		t.Errorf("the window received a call to focus %v", focused)
	}
}

func TestNoteFocusOfNoteNotInVaultIsAToolErrorAndMovesNothing(t *testing.T) {
	w := newWindowTools(t)
	if failure := w.call(t, "note_focus", `{"path":"notes/Gone.md"}`, nil, nil); failure == "" {
		t.Error("a note not in the vault was focused")
	}
	if focused := w.window.GetFocusedNotes(); len(focused) != 0 {
		t.Errorf("the window received a call to focus %v", focused)
	}
}

func TestSourceFocusPassesPassagesAndReturnsPathAndCount(t *testing.T) {
	w := newWindowTools(t)
	var out struct {
		Path           string `json:"path"`
		HighlightCount *int   `json:"highlightCount"`
	}
	arguments := `{"path":"library/Bird Guide.pdf","passages":[{"start":1500,"length":80},{"start":400,"length":700}]}`
	if failure := w.call(t, "source_focus", arguments, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Path != guidePath || out.HighlightCount == nil || *out.HighlightCount != 2 {
		t.Errorf("returned %+v", out)
	}
	want := []hosttest.FocusRequest{{Path: guidePath, Passages: []host.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}}}}
	if got := w.window.GetFocusRequests(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window received %+v, want %+v", got, want)
	}
}

func TestSourceFocusWithNoPassagesOpensTheFileAtItsBeginning(t *testing.T) {
	w := newWindowTools(t)
	var out struct {
		Path           string `json:"path"`
		HighlightCount *int   `json:"highlightCount"`
	}
	if failure := w.call(t, "source_focus", `{"path":"library/Bird Guide.pdf"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Path != guidePath || out.HighlightCount == nil || *out.HighlightCount != 0 {
		t.Errorf("returned %+v", out)
	}
	got := w.window.GetFocusRequests()
	if len(got) != 1 || got[0].Path != guidePath || len(got[0].Passages) != 0 {
		t.Errorf("the window received %+v", got)
	}
}

func TestSourceFocusReportsAHostErrorAsAToolError(t *testing.T) {
	w := newWindowTools(t)
	if failure := w.call(t, "source_focus", `{"path":"library/Gone.pdf"}`, nil, nil); failure == "" {
		t.Error("a file not in the vault was reported as shown")
	}
	w.window.SetNoWindow()
	if failure := w.call(t, "source_focus", `{"path":"library/Bird Guide.pdf"}`, nil, nil); failure == "" {
		t.Error("a host with no window reported the file as shown")
	}
}

func (w windowTools) readTabs(t *testing.T, tabs []host.OpenTab, activeID string) tabsOutput {
	t.Helper()
	w.window.SetOpenTabs(tabs, activeID)
	var out tabsOutput
	if failure := w.call(t, "window_tab_list", `{}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	return out
}

func TestWindowTabListReturnsTheTabsOfTheWindow(t *testing.T) {
	w := newWindowTools(t)
	out := w.readTabs(t, []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Gray Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 4, PageCount: 120}},
	}, "tab-2")

	if len(out.Tabs) != 2 || out.ActiveTabID != "tab-2" {
		t.Fatalf("returned %+v", out)
	}
	first, second := out.Tabs[0], out.Tabs[1]
	if first.TabID != "tab-1" || first.Kind != "note" || first.Path != notePath || first.Title != "Gray Heron" || first.IsActive {
		t.Errorf("the first tab is %+v", first)
	}
	requirePlace(t, "the note tab", first.Place, "")
	if second.TabID != "tab-2" || second.Path != guidePath || !second.IsActive {
		t.Errorf("the second tab is %+v", second)
	}
	requirePlace(t, "the PDF tab", second.Place, `{"page":4,"pageCount":120}`)
}

func TestWindowTabListReturnsThePlaceTheReaderReported(t *testing.T) {
	for name, tc := range map[string]struct {
		tab  host.OpenTab
		want string
	}{
		"a page of a PDF and its page count": {
			host.OpenTab{Kind: "reader:reader-pdf", Path: guidePath, Place: &host.TabPlace{Page: 4, PageCount: 120}}, `{"page":4,"pageCount":120}`,
		},
		"a page of a PDF with no page count": {
			host.OpenTab{Kind: "reader:reader-pdf", Path: guidePath, Place: &host.TabPlace{Page: 4}}, `{"page":4}`,
		},
		"a recording at a time": {
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: new(66.5)}}, `{"positionSeconds":66.5}`,
		},
		"a recording at its beginning": {
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: new(float64(0))}}, `{"positionSeconds":0}`,
		},
		"a recording past an hour": {
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: new(3723.9)}}, `{"positionSeconds":3723.9}`,
		},
		"a place of an EPUB and its chapter": {
			host.OpenTab{Kind: "reader:reader-epub", Path: bookPath, Place: &host.TabPlace{CFI: "epubcfi(/6/4!/4/2/1:0)", Chapter: "The Reed Beds"}},
			`{"cfi":"epubcfi(/6/4!/4/2/1:0)","chapter":"The Reed Beds"}`,
		},
		"a place of an EPUB with no chapter": {
			host.OpenTab{Kind: "reader:reader-epub", Path: bookPath, Place: &host.TabPlace{CFI: "epubcfi(/6/6!/4/12/1:0)"}}, `{"cfi":"epubcfi(/6/6!/4/12/1:0)"}`,
		},
		"the place of a tab of any kind": {
			host.OpenTab{Kind: "plex", Path: guidePath, Place: &host.TabPlace{Page: 7}}, `{"page":7}`,
		},
		"a note tab whose kind reports no place": {host.OpenTab{Kind: "note", Path: notePath}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			tab := tc.tab
			tab.ID = "tab-1"
			out := w.readTabs(t, []host.OpenTab{tab}, "tab-1")
			if len(out.Tabs) != 1 {
				t.Fatalf("returned %+v, want one tab", out.Tabs)
			}
			requirePlace(t, "the tab", out.Tabs[0].Place, tc.want)
		})
	}
}

// The place comes from the place the reader reported alone, so window_tab_list has no saved state
// to fail on and returns no error for any tab.
func TestWindowTabListReturnsNoErrorForAnyTab(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	var out struct {
		Tabs []map[string]json.RawMessage `json:"tabs"`
	}
	if failure := w.call(t, "window_tab_list", `{}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != len(getGuideTabs()) {
		t.Fatalf("returned %d tabs, want %d", len(out.Tabs), len(getGuideTabs()))
	}
	for _, tab := range out.Tabs {
		if message, hasError := tab["error"]; hasError {
			t.Errorf("the tab %s returns the error %s", tab["tabId"], message)
		}
	}
	if _, hasError := reflect.TypeFor[api.TabInfo]().FieldByName("Error"); hasError {
		t.Error("api.TabInfo has the field Error; a tab whose reader reported no place has no place and no error")
	}
}

func TestWindowToolsHostHasNoTextVersions(t *testing.T) {
	if field, hasTexts := reflect.TypeFor[service.Host]().FieldByName("Texts"); hasTexts {
		t.Errorf("service.Host has the field %s of type %s; window_tab_list reads the place from the place the reader reported alone", field.Name, field.Type)
	}
}

func TestWindowTabListUsesTheCallFocusWithoutAWindow(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: notePath, Title: "Gray Heron"}}, "tab-1")
	w.window.SetNoWindow()

	var out tabsOutput
	if failure := w.call(t, "window_tab_list", `{}`, map[string]string{"focus": "notes/Inferred.md"}, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != 1 || out.Tabs[0].Path != "notes/Inferred.md" || out.ActiveTabID != "front-tab" || out.Tabs[0].Title != "Inferred" {
		t.Fatalf("returned %+v, want the focus of the call as the one tab", out)
	}
	requirePlace(t, "the tab of the focus", out.Tabs[0].Place, "")

	var none tabsOutput
	if failure := w.call(t, "window_tab_list", `{}`, nil, &none); failure != "" {
		t.Fatal(failure)
	}
	if len(none.Tabs) != 0 {
		t.Errorf("returned %+v, want nothing open", none)
	}
}

func TestWindowTabListPrefersWindowTabsToTheCallFocus(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: notePath, Title: "Gray Heron"}}, "tab-1")

	var out tabsOutput
	if failure := w.call(t, "window_tab_list", `{}`, map[string]string{"focus": "notes/Inferred.md"}, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != 1 || out.Tabs[0].Path != notePath || out.ActiveTabID != "tab-1" {
		t.Errorf("returned %+v, want the tab of the window", out)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
