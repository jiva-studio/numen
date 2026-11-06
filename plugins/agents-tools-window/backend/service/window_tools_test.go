package service_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
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
	window *hosttest.Window
	calls  *tools.Registry
}

func newWindowTools(t *testing.T) windowTools {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File(notePath, "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: Grey Heron\n---\nHerons fish at dusk.")
	vault.File(guidePath, pdfBytes)
	vault.File(bookPath, "PK\x03\x04epub body")
	vault.File(talkPath, "ID3\x00\x03audio")

	window := hosttest.NewWindow(vault)
	return windowTools{window: window, calls: newRegistry(t, service.Host{Vault: vault, Window: window})}
}

// newRegistry is the window tools registered over the host.
func newRegistry(t *testing.T, on service.Host) *tools.Registry {
	t.Helper()
	registry := tools.New()
	manager := service.NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	if err := service.RegisterTools(registry, manager, on); err != nil {
		t.Fatal(err)
	}
	return registry
}

// call runs a tool on the vault with the call context and returns the structured answer, or the
// text of the tool error.
func (w windowTools) call(t *testing.T, name, arguments string, callContext map[string]string, out any) (failure string) {
	t.Helper()
	res, err := w.calls.InvokeTool(t.Context(), tools.Call{
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
	listed, err := w.calls.ListTools(t.Context())
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

// isOnServer reports whether the tool is on the server. A tool that names no server is on the full
// server alone.
func isOnServer(info *tools.ToolInfo, server tools.Server) bool {
	if len(info.GetServers()) == 0 {
		return server == tools.Full
	}
	return slices.Contains(info.GetServers(), server)
}

// tabOutput is one tab as a tool answers it. Location is the raw JSON of the location key, nil where
// the key is absent.
type tabOutput struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Path     string          `json:"path"`
	Title    string          `json:"title"`
	Location json.RawMessage `json:"location"`
	IsActive bool            `json:"isActive"`
}

type tabsOutput struct {
	Tabs     []tabOutput `json:"tabs"`
	ActiveID string      `json:"activeId"`
}

// requireLocation fails the test unless the location is the JSON object want, or absent where want
// is empty.
func requireLocation(t *testing.T, what string, location json.RawMessage, want string) {
	t.Helper()
	if want == "" {
		if location != nil {
			t.Errorf("%s has the location %s, want none", what, location)
		}
		return
	}
	var got, wanted map[string]any
	if err := json.Unmarshal(location, &got); err != nil {
		t.Errorf("%s has the location %s, want %s", what, location, want)
		return
	}
	if err := json.Unmarshal([]byte(want), &wanted); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("%s has the location %s, want %s", what, location, want)
	}
}

func TestTheToolsThatMoveTheWindowAreOnTheFullServerOnly(t *testing.T) {
	w := newWindowTools(t)
	for _, name := range []string{"note_focus", "source_focus", "window_focus_tab"} {
		info := w.getToolInfo(t, name)
		if !isOnServer(info, tools.Full) || isOnServer(info, tools.Reading) {
			t.Errorf("%s is on %v, want the full server only", name, info.GetServers())
		}
	}
	tabs := w.getToolInfo(t, "window_tabs_get")
	if !isOnServer(tabs, tools.Full) || !isOnServer(tabs, tools.Reading) {
		t.Errorf("window_tabs_get is on %v, want the full and the reading server", tabs.GetServers())
	}
}

func TestNoteFocusDescriptionExplainsWhenToMoveTheWindow(t *testing.T) {
	w := newWindowTools(t)
	description := w.getToolInfo(t, "note_focus").GetDescription()
	for _, want := range []string{"neighbourhood", "asked to be taken to a note", "Talking about one is not asking", "`note_read`"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description of note_focus does not say %q: %s", want, description)
		}
	}
	sourceDescription := w.getToolInfo(t, "source_focus").GetDescription()
	for _, want := range []string{"`library_search`", "start", "length", "first", "Talking about a passage is not asking"} {
		if !strings.Contains(sourceDescription, want) {
			t.Errorf("the description of source_focus does not say %q: %s", want, sourceDescription)
		}
	}
}

func TestNoteFocusAnswersTheFocusedNoteAndMovesTheWindowToIt(t *testing.T) {
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
	if out.Note.Path != notePath || out.Note.Title != "Grey Heron" || out.Note.ID != "01H7X1234567890ABCDEFGHJKM" {
		t.Errorf("note %+v", out.Note)
	}
	if focused := w.window.GetFocusedNotes(); !slices.Equal(focused, []string{notePath}) {
		t.Errorf("the window was asked to focus %v", focused)
	}
}

func TestNoteFocusOfANoteTheVaultDoesNotHoldIsAToolErrorAndMovesNothing(t *testing.T) {
	w := newWindowTools(t)
	if failure := w.call(t, "note_focus", `{"path":"notes/Gone.md"}`, nil, nil); failure == "" {
		t.Error("a note the vault does not hold was focused")
	}
	if focused := w.window.GetFocusedNotes(); len(focused) != 0 {
		t.Errorf("the window was asked to focus %v", focused)
	}
}

func TestSourceFocusPassesThePassagesUnchangedAndAnswersThePathAndHighlightCount(t *testing.T) {
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
		t.Errorf("answered %+v", out)
	}
	want := []hosttest.FocusRequest{{Path: guidePath, Passages: []host.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}}}}
	if got := w.window.GetFocusRequests(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window was asked %+v, want %+v", got, want)
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
		t.Errorf("answered %+v", out)
	}
	got := w.window.GetFocusRequests()
	if len(got) != 1 || got[0].Path != guidePath || len(got[0].Passages) != 0 {
		t.Errorf("the window was asked %+v", got)
	}
}

func TestSourceFocusReportsAHostErrorAsAToolError(t *testing.T) {
	w := newWindowTools(t)
	if failure := w.call(t, "source_focus", `{"path":"library/Gone.pdf"}`, nil, nil); failure == "" {
		t.Error("a file the vault does not hold was reported as shown")
	}
	w.window.SetNoWindow()
	if failure := w.call(t, "source_focus", `{"path":"library/Bird Guide.pdf"}`, nil, nil); failure == "" {
		t.Error("a host with no window reported the file as shown")
	}
}

// readTabs sets the window's tabs and returns what window_tabs_get answers.
func (w windowTools) readTabs(t *testing.T, tabs []host.OpenTab, activeID string) tabsOutput {
	t.Helper()
	w.window.SetOpenTabs(tabs, activeID)
	var out tabsOutput
	if failure := w.call(t, "window_tabs_get", `{}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	return out
}

func TestWindowTabsGetAnswersTheTabsOfTheWindow(t *testing.T) {
	w := newWindowTools(t)
	out := w.readTabs(t, []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 4, PageCount: 120}},
	}, "tab-2")

	if len(out.Tabs) != 2 || out.ActiveID != "tab-2" {
		t.Fatalf("answered %+v", out)
	}
	first, second := out.Tabs[0], out.Tabs[1]
	if first.ID != "tab-1" || first.Kind != "note" || first.Path != notePath || first.Title != "Grey Heron" || first.IsActive {
		t.Errorf("the first tab is %+v", first)
	}
	requireLocation(t, "the note tab", first.Location, "")
	if second.ID != "tab-2" || second.Path != guidePath || !second.IsActive {
		t.Errorf("the second tab is %+v", second)
	}
	requireLocation(t, "the PDF tab", second.Location, `{"page":4,"pageCount":120}`)
}

// seconds is a time of a recording, from its start.
func seconds(n float64) *float64 { return &n }

func TestWindowTabsGetReportsTheLocationFromThePlaceTheReaderReported(t *testing.T) {
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
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: seconds(66.5)}}, `{"positionSeconds":66.5}`,
		},
		"a recording at its beginning": {
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: seconds(0)}}, `{"positionSeconds":0}`,
		},
		"a recording past an hour": {
			host.OpenTab{Kind: "recording", Path: talkPath, Place: &host.TabPlace{PositionSeconds: seconds(3723.9)}}, `{"positionSeconds":3723.9}`,
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
				t.Fatalf("answered %+v, want one tab", out.Tabs)
			}
			requireLocation(t, "the tab", out.Tabs[0].Location, tc.want)
		})
	}
}

// The location comes from the place the reader reported alone, so window_tabs_get has no saved state
// to fail on and answers no error for any tab.
func TestWindowTabsGetAnswersNoErrorForAnyTab(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	var out struct {
		Tabs []map[string]json.RawMessage `json:"tabs"`
	}
	if failure := w.call(t, "window_tabs_get", `{}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != len(getGuideTabs()) {
		t.Fatalf("answered %d tabs, want %d", len(out.Tabs), len(getGuideTabs()))
	}
	for _, tab := range out.Tabs {
		if message, hasError := tab["error"]; hasError {
			t.Errorf("the tab %s is answered with the error %s", tab["id"], message)
		}
	}
	if _, hasError := reflect.TypeFor[service.TabInfo]().FieldByName("Error"); hasError {
		t.Error("service.TabInfo has the field Error; a tab whose reader reported no place has no location and no error")
	}
}

func TestWindowToolsHostHasNoTextVersions(t *testing.T) {
	if field, hasTexts := reflect.TypeFor[service.Host]().FieldByName("Texts"); hasTexts {
		t.Errorf("service.Host has the field %s of type %s; window_tabs_get reads the location from the place the reader reported alone", field.Name, field.Type)
	}
}

func TestWindowTabsGetFallsBackToTheFocusOfTheCallWithoutAWindow(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"}}, "tab-1")
	w.window.SetNoWindow()

	var out tabsOutput
	if failure := w.call(t, "window_tabs_get", `{}`, map[string]string{"focus": "notes/Inferred.md"}, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != 1 || out.Tabs[0].Path != "notes/Inferred.md" || out.ActiveID != "front-tab" || out.Tabs[0].Title != "Inferred" {
		t.Fatalf("answered %+v, want the focus of the call as the one tab", out)
	}
	requireLocation(t, "the tab of the focus", out.Tabs[0].Location, "")

	var none tabsOutput
	if failure := w.call(t, "window_tabs_get", `{}`, nil, &none); failure != "" {
		t.Fatal(failure)
	}
	if len(none.Tabs) != 0 {
		t.Errorf("answered %+v, want nothing open", none)
	}
}

func TestWindowTabsGetPrefersTheTabsOfTheWindowToTheFocusOfTheCall(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"}}, "tab-1")

	var out tabsOutput
	if failure := w.call(t, "window_tabs_get", `{}`, map[string]string{"focus": "notes/Inferred.md"}, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Tabs) != 1 || out.Tabs[0].Path != notePath || out.ActiveID != "tab-1" {
		t.Errorf("answered %+v, want the tab of the window", out)
	}
}
