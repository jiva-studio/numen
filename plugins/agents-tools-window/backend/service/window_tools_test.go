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
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

// The invented vault: a note, a bird guide whose text layer runs to page 120, a scan with no text
// version, a book of two chapters and a recorded talk.
const (
	testVault = "marsh"
	notePath  = "notes/Heron.md"
	guidePath = "library/Bird Guide.pdf"
	scanPath  = "library/Old Scan.pdf"
	bookPath  = "library/Marsh Book.epub"
	talkPath  = "recordings/Marsh Talk.mp3"
	pdfBytes  = "%PDF-1.7\n\x00\x01binary body"
	epubBytes = "PK\x03\x04epub body"
	readerID  = "org.example.reader"
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
	vault.File(scanPath, pdfBytes+"scan")
	vault.File(bookPath, epubBytes)
	vault.File(talkPath, "ID3\x00\x03audio")

	texts := hosttest.NewTexts(testVault, readerID)
	texts.File(guidePath, []byte(pdfBytes))
	texts.File(bookPath, []byte(epubBytes))
	putText(t, texts, guidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: "Chapter one. The marsh at dawn. ", Selector: `{"page":1}`},
		{Text: "The grey heron stands in the reeds. ", Selector: `{"page":4,"boxes":[[0,0,1,1]]}`},
		{Text: "Index of birds. ", Selector: `{"page":120}`},
		{Text: "Herons nest in tall trees. ", Selector: `{"page":7}`},
	})
	putText(t, texts, bookPath, "book-text", []byte(epubBytes), []domain.TextSegment{
		{Text: "The marsh wakes slowly. ", Heading: "The Marsh at Dawn", Selector: `{"cfi":"epubcfi(/6/4!/4/2/1:0)"}`},
		{Text: "Mist lies on the water. ", Heading: "The Marsh at Dawn", Selector: `{"cfi":"epubcfi(/6/4!/4/8/1:0)"}`},
		{Text: "In winter the herons stay. ", Heading: "Herons in Winter", Selector: `{"cfi":"epubcfi(/6/6!/4/2/1:0)"}`},
		{Text: "They fish under the ice. ", Heading: "Herons in Winter", Selector: `{"cfi":"epubcfi(/6/6!/4/12/1:0)"}`},
	})

	window := hosttest.NewWindow(vault)
	registry := tools.New()
	manager := service.NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	if err := service.RegisterTools(registry, manager, service.Host{Vault: vault, Window: window, Texts: texts}); err != nil {
		t.Fatal(err)
	}
	return windowTools{window: window, calls: registry}
}

func putText(t *testing.T, texts *hosttest.Texts, path, kind string, file []byte, segments []domain.TextSegment) {
	t.Helper()
	name := kind + ".jsonl"
	if err := host.WriteText(t.Context(), texts.GetStore(readerID), name, segments); err != nil {
		t.Fatal(err)
	}
	if _, err := texts.PutVersion(t.Context(), host.TextPut{
		Path: path, Key: kind, Kind: domain.TextKind(kind), SourceHash: domain.HashContent(file), File: name,
	}); err != nil {
		t.Fatal(err)
	}
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
	if res.GetResult().GetIsRefusal() {
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

type tabsOutput struct {
	Tabs []struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Path     string `json:"path"`
		Title    string `json:"title"`
		Where    string `json:"where"`
		IsActive bool   `json:"active"`
	} `json:"tabs"`
	ActiveID string `json:"activeId"`
	Summary  string `json:"summary"`
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

func TestNoteFocusDescriptionTellsWhenToMoveThePerson(t *testing.T) {
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
		Focused struct {
			Path  string `json:"path"`
			Title string `json:"title"`
			ID    string `json:"id"`
		} `json:"focused"`
	}
	if failure := w.call(t, "note_focus", `{"path":"notes/Heron.md"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Focused.Path != notePath || out.Focused.Title != "Grey Heron" || out.Focused.ID != "01H7X1234567890ABCDEFGHJKM" {
		t.Errorf("focused %+v", out.Focused)
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

func TestSourceFocusPassesThePassagesAsTheyStandAndAnswersTheSummary(t *testing.T) {
	w := newWindowTools(t)
	var out struct {
		IsShown bool   `json:"shown"`
		Summary string `json:"summary"`
	}
	arguments := `{"path":"library/Bird Guide.pdf","passages":[{"start":1500,"length":80},{"start":400,"length":700}]}`
	if failure := w.call(t, "source_focus", arguments, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if !out.IsShown || out.Summary != "the document is open at that passage, with the words of it lit" {
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
		IsShown bool   `json:"shown"`
		Summary string `json:"summary"`
	}
	if failure := w.call(t, "source_focus", `{"path":"library/Bird Guide.pdf"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if !out.IsShown || out.Summary != "the document is open at its beginning" {
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
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":4}`},
	}, "tab-2")

	if len(out.Tabs) != 2 || out.ActiveID != "tab-2" {
		t.Fatalf("answered %+v", out)
	}
	first, second := out.Tabs[0], out.Tabs[1]
	if first.ID != "tab-1" || first.Kind != "note" || first.Path != notePath || first.Title != "Grey Heron" || first.IsActive || first.Where != "" {
		t.Errorf("the first tab is %+v", first)
	}
	if second.ID != "tab-2" || second.Path != guidePath || !second.IsActive || second.Where != "page 4 of 120" {
		t.Errorf("the second tab is %+v", second)
	}
}

func TestWindowTabsGetReadsThePositionOfEachReaderTab(t *testing.T) {
	for name, tc := range map[string]struct {
		tab  host.OpenTab
		want string
	}{
		"a page of a PDF with a text version":     {host.OpenTab{Path: guidePath, State: `{"page":4}`}, "page 4 of 120"},
		"a page of a PDF with no text version":    {host.OpenTab{Path: scanPath, State: `{"page":4}`}, "page 4"},
		"a time of a recording past an hour":      {host.OpenTab{Path: talkPath, State: `{"t":[3723.9,3730]}`}, "1:02:03"},
		"a time of a recording under an hour":     {host.OpenTab{Path: talkPath, State: `{"t":[125.5,130]}`}, "2:05"},
		"a CFI a segment starts at":               {host.OpenTab{Path: bookPath, State: `{"cfi":"epubcfi(/6/6!/4/2/1:0)"}`}, `chapter "Herons in Winter"`},
		"a CFI between two segments":              {host.OpenTab{Path: bookPath, State: `{"cfi":"epubcfi(/6/4!/4/10/1:17)"}`}, `chapter "The Marsh at Dawn"`},
		"a CFI after the last segment":            {host.OpenTab{Path: bookPath, State: `{"cfi":"epubcfi(/6/6!/4/20/1:3)"}`}, `chapter "Herons in Winter"`},
		"a CFI before every segment":              {host.OpenTab{Path: bookPath, State: `{"cfi":"epubcfi(/6/2!/4/2/1:0)"}`}, ""},
		"a CFI of a book with no text version":    {host.OpenTab{Path: "library/Other.epub", State: `{"cfi":"epubcfi(/6/4!/4/2/1:0)"}`}, ""},
		"a state of keys no reader saves":         {host.OpenTab{Path: guidePath, State: `{"zoom":2}`}, ""},
		"no state":                                {host.OpenTab{Path: guidePath}, ""},
		"a state that is not JSON":                {host.OpenTab{Path: guidePath, State: `page four`}, ""},
		"a recording state with no start":         {host.OpenTab{Path: talkPath, State: `{"t":[]}`}, ""},
		"a page that is no number":                {host.OpenTab{Path: guidePath, State: `{"page":"four"}`}, ""},
		"the page key read whatever the tab kind": {host.OpenTab{Path: guidePath, Kind: "plex", State: `{"page":7}`}, "page 7 of 120"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			tab := tc.tab
			tab.ID = "tab-1"
			out := w.readTabs(t, []host.OpenTab{tab}, "tab-1")
			if len(out.Tabs) != 1 || out.Tabs[0].Where != tc.want {
				t.Errorf("answered %+v, want the tab at %q", out.Tabs, tc.want)
			}
		})
	}
}

func TestWindowTabsGetSummaryOfTheActiveTabIncludesItsPosition(t *testing.T) {
	w := newWindowTools(t)
	out := w.readTabs(t, []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":4}`},
	}, "tab-2")

	want := `the reader:reader-pdf "Bird Guide" at library/Bird Guide.pdf is in front of them, open at page 4 of 120`
	if out.Summary != want {
		t.Errorf("the summary is %q, want %q", out.Summary, want)
	}
}

func TestWindowTabsGetSummaryOfARecordingNamesTheTimeAfterASinglePreposition(t *testing.T) {
	w := newWindowTools(t)
	out := w.readTabs(t, []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"},
		{ID: "tab-2", Kind: "recording", Path: talkPath, Title: "Marsh Talk", State: `{"t":[66,66]}`},
	}, "tab-2")

	want := `the recording "Marsh Talk" at recordings/Marsh Talk.mp3 is in front of them, open at 1:06`
	if out.Summary != want {
		t.Errorf("the summary is %q, want %q", out.Summary, want)
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
	if len(out.Tabs) != 1 || out.Tabs[0].Path != "notes/Inferred.md" || out.ActiveID != "front-tab" || !strings.Contains(out.Summary, "Inferred") {
		t.Errorf("answered %+v, want the focus of the call as the one tab", out)
	}

	var none tabsOutput
	if failure := w.call(t, "window_tabs_get", `{}`, nil, &none); failure != "" {
		t.Fatal(failure)
	}
	if len(none.Tabs) != 0 || none.Summary != "the window has nothing open" {
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

// focusTab sets the window's tabs, runs window_focus_tab and returns the failure text, if any.
func (w windowTools) focusTab(t *testing.T, arguments string) string {
	t.Helper()
	w.window.SetOpenTabs([]host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Grey Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":4}`},
		{ID: "tab-3", Kind: "plex", Title: "Neighbourhood"},
	}, "tab-1")
	return w.call(t, "window_focus_tab", arguments, nil, nil)
}

func TestWindowFocusTabMovesTheWindowToTheNoteOfATab(t *testing.T) {
	for name, arguments := range map[string]string{
		"by id":    `{"tabId":"tab-1"}`,
		"by path":  `{"path":"notes/Heron.md"}`,
		"by title": `{"title":"Grey Heron"}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			if failure := w.focusTab(t, arguments); failure != "" {
				t.Fatal(failure)
			}
			if focused := w.window.GetFocusedNotes(); !slices.Equal(focused, []string{notePath}) || len(w.window.GetFocusRequests()) != 0 {
				t.Errorf("the window was asked to focus notes %v and files %+v", focused, w.window.GetFocusRequests())
			}
		})
	}
}

func TestWindowFocusTabOpensTheFileOfATabWithNoPassages(t *testing.T) {
	for name, arguments := range map[string]string{
		"by id":    `{"tabId":"tab-2"}`,
		"by title": `{"title":"Bird Guide"}`,
		"by path":  `{"path":"library/Bird Guide.pdf"}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			if failure := w.focusTab(t, arguments); failure != "" {
				t.Fatal(failure)
			}
			want := []hosttest.FocusRequest{{Path: guidePath}}
			got := w.window.GetFocusRequests()
			if len(got) != 1 || got[0].Path != want[0].Path || len(got[0].Passages) != 0 || len(w.window.GetFocusedNotes()) != 0 {
				t.Errorf("the window was asked %+v and notes %v, want %+v", got, w.window.GetFocusedNotes(), want)
			}
		})
	}
}

func TestWindowFocusTabOpensAPathWithoutATab(t *testing.T) {
	w := newWindowTools(t)
	if failure := w.focusTab(t, `{"path":"library/Old Scan.pdf"}`); failure != "" {
		t.Fatal(failure)
	}
	got := w.window.GetFocusRequests()
	if len(got) != 1 || got[0].Path != scanPath {
		t.Errorf("the window was asked %+v, want the scan", got)
	}
}

func TestWindowFocusTabRefusesATabWithNoPathAndNothingThatMatches(t *testing.T) {
	for name, arguments := range map[string]string{
		"a tab with no path":        `{"tabId":"tab-3"}`,
		"a title no tab has":        `{"title":"Nothing Here"}`,
		"an id no tab has":          `{"tabId":"tab-9"}`,
		"no tab, no path, no title": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			if failure := w.focusTab(t, arguments); failure == "" {
				t.Error("the tab was focused")
			}
			if len(w.window.GetFocusedNotes()) != 0 || len(w.window.GetFocusRequests()) != 0 {
				t.Errorf("the window was moved to notes %v and files %+v", w.window.GetFocusedNotes(), w.window.GetFocusRequests())
			}
		})
	}
}
