package service_test

import (
	"encoding/json"
	"maps"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

func decodeAnswer(t *testing.T, structured []byte) map[string]any {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal(structured, &answer); err != nil {
		t.Fatalf("unmarshal %s: %v", structured, err)
	}
	return answer
}

func requireKeys(t *testing.T, what string, object map[string]any, want, absent []string) {
	t.Helper()
	for _, key := range want {
		if _, ok := object[key]; !ok {
			t.Errorf("%s has no key %q: %v", what, key, object)
		}
	}
	for _, key := range absent {
		if _, ok := object[key]; ok {
			t.Errorf("%s has the key %q: %v", what, key, object)
		}
	}
}

func requireList(t *testing.T, what string, value any) []any {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is %v, not an array", what, value)
	}
	return list
}

func requireObject(t *testing.T, what string, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is %v, not an object", what, value)
	}
	return object
}

func getToolSchema(t *testing.T, registry *tools.Registry, name string, isOutput bool) map[string]any {
	t.Helper()
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() != name {
			continue
		}
		raw := info.GetInputSchema()
		if isOutput {
			raw = info.GetOutputSchema()
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("schema of %s: %v in %s", name, err, raw)
		}
		return schema
	}
	t.Fatalf("no tool %s is registered", name)
	return nil
}

func getSchemaProperties(t *testing.T, schema map[string]any, path ...string) map[string]any {
	t.Helper()
	node := schema
	for _, name := range path {
		properties, hasProperties := node["properties"].(map[string]any)
		if !hasProperties {
			t.Fatalf("the schema has no properties before %q along %v", name, path)
		}
		next, ok := properties[name].(map[string]any)
		if !ok {
			t.Fatalf("the schema has no property %q along %v", name, path)
		}
		if items, isArray := next["items"].(map[string]any); isArray {
			next = items
		}
		node = next
	}
	properties, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("the schema has no properties at the end of %v", path)
	}
	return properties
}

func visitSchema(node any, onProperty, onDescription func(string)) {
	object, ok := node.(map[string]any)
	if !ok {
		return
	}
	if description, isText := object["description"].(string); isText {
		onDescription(description)
	}
	if properties, isObject := object["properties"].(map[string]any); isObject {
		for name, property := range properties {
			onProperty(name)
			visitSchema(property, onProperty, onDescription)
		}
	}
	for _, key := range []string{"items", "additionalProperties"} {
		visitSchema(object[key], onProperty, onDescription)
	}
	for _, key := range []string{"$defs", "definitions"} {
		if definitions, isObject := object[key].(map[string]any); isObject {
			for _, definition := range definitions {
				visitSchema(definition, onProperty, onDescription)
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if variants, isList := object[key].([]any); isList {
			for _, variant := range variants {
				visitSchema(variant, onProperty, onDescription)
			}
		}
	}
}

func visitEveryToolSchema(t *testing.T, registry *tools.Registry, onProperty, onDescription func(tool, text string)) {
	t.Helper()
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		onDescription(info.GetName(), info.GetDescription())
		for _, raw := range [][]byte{info.GetInputSchema(), info.GetOutputSchema()} {
			if len(raw) == 0 {
				continue
			}
			var schema any
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatalf("schema of %s: %v in %s", info.GetName(), err, raw)
			}
			name := info.GetName()
			visitSchema(schema,
				func(text string) { onProperty(name, text) },
				func(text string) { onDescription(name, text) })
		}
	}
}

func getNarrativeMarkers() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bperson\b`),
		regexp.MustCompile(`(?i)\bin front of\b`),
		regexp.MustCompile(`(?i)\bstand(s|ing)?\b`),
		regexp.MustCompile(`(?i)\bcut by\b`),
		regexp.MustCompile(`(?i)\blights?\b`),
		regexp.MustCompile(`(?i)\bis called\b`),
		regexp.MustCompile(`(?i)\byou did not see\b`),
		regexp.MustCompile(`(?i)\bhand this\b`),
		regexp.MustCompile(`(?i)\bsay back\b`),
		regexp.MustCompile(`(?i)\bfiles it under\b`),
		regexp.MustCompile(`(?i)\bspeak\b`),
		regexp.MustCompile(`(?i)\bhold(s|ing)?\b`),
		regexp.MustCompile(`(?i)\breach(es|ing)?\b`),
		regexp.MustCompile(`(?i)\bcarr(y|ies|ied|ying)\b`),
		regexp.MustCompile(`(?i)\brefus(e|es|ed|ing)\b`),
		regexp.MustCompile(`(?i)\blies inside\b`),
		regexp.MustCompile(`(?i)\bgoes to\b`),
		regexp.MustCompile(`(?i)\blit\b`),
		regexp.MustCompile(`(?i)\btak(e|es|ing)\b[^.;]*\boff\b`),
	}
}

func (w windowTools) getAnswer(t *testing.T, name, arguments string) map[string]any {
	t.Helper()
	res, err := w.registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified || res.GetResult().GetIsError() {
		t.Fatalf("%s %s failed: %v %s", name, arguments, res.GetError(), res.GetResult().GetText())
	}
	return decodeAnswer(t, res.GetResult().GetStructured())
}

func TestWindowTabListReturnsPlaceTabIDAndIsActive(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: notePath, Title: "Gray Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 4, PageCount: 120}},
	}, "tab-2")
	answer := w.getAnswer(t, "window_tab_list", `{}`)
	requireKeys(t, "the answer of window_tab_list", answer, []string{"tabs", "activeTabId"}, []string{"summary", "activeId"})
	if answer["activeTabId"] != "tab-2" {
		t.Errorf("activeTabId = %v, want tab-2", answer["activeTabId"])
	}
	list := requireList(t, "answer[tabs]", answer["tabs"])
	if len(list) != 2 {
		t.Fatalf("tabs = %v, want two", answer["tabs"])
	}
	active := requireObject(t, "list[1]", list[1])
	requireKeys(t, "the active tab", active, []string{"tabId", "place", "isActive"}, []string{"id", "location", "where", "active"})
	place := requireObject(t, "the place of the active tab", active["place"])
	if active["tabId"] != "tab-2" || place["page"] != float64(4) || place["pageCount"] != float64(120) || len(place) != 2 || active["isActive"] != true {
		t.Errorf("the active tab is %v, want tab-2 at place {page 4, pageCount 120} and isActive true", active)
	}
	inactive := requireObject(t, "list[0]", list[0])
	requireKeys(t, "the inactive note tab", inactive, []string{"tabId"}, []string{"id", "where", "active", "location", "place"})
	if inactive["isActive"] == true {
		t.Errorf("the inactive tab is %v, want isActive not true", inactive)
	}

	schema := getToolSchema(t, w.registry, "window_tab_list", true)
	requireKeys(t, "the output schema of window_tab_list", getSchemaProperties(t, schema), []string{"tabs", "activeTabId"}, []string{"summary", "activeId"})
	requireKeys(t, "the output schema of a tab", getSchemaProperties(t, schema, "tabs"),
		[]string{"tabId", "place", "isActive"}, []string{"id", "location", "where", "active"})
	requireExactKeys(t, "the output schema of a tab place", getSchemaProperties(t, schema, "tabs", "place"), getPlaceKeys())
}

func getPlaceKeys() []string {
	return []string{"page", "pageCount", "positionSeconds", "cfi", "chapter"}
}

func requireExactKeys(t *testing.T, what string, object map[string]any, want []string) {
	t.Helper()
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	slices.Sort(got)
	want = slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("%s has the keys %v, want %v", what, got, want)
	}
}

func TestWindowTabListReturnsThePlaceOfAPDFARecordingAtItsStartAndAnEPUB(t *testing.T) {
	w := newWindowTools(t)
	atStart := 0.0
	w.window.SetOpenTabs([]host.OpenTab{
		{ID: "tab-1", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 4, PageCount: 120}},
		{ID: "tab-2", Kind: "recording", Path: talkPath, Title: "Marsh Talk", Place: &host.TabPlace{PositionSeconds: &atStart}},
		{ID: "tab-3", Kind: "reader:reader-epub", Path: bookPath, Title: "Marsh Book", Place: &host.TabPlace{CFI: "epubcfi(/6/4!/4/2/1:0)", Chapter: "The Reed Beds"}},
	}, "tab-1")
	list := requireList(t, "answer[tabs]", w.getAnswer(t, "window_tab_list", `{}`)["tabs"])
	if len(list) != 3 {
		t.Fatalf("tabs = %v, want three", list)
	}
	for i, want := range []map[string]any{
		{"page": float64(4), "pageCount": float64(120)},
		{"positionSeconds": float64(0)},
		{"cfi": "epubcfi(/6/4!/4/2/1:0)", "chapter": "The Reed Beds"},
	} {
		tab := requireObject(t, "a tab", list[i])
		place := requireObject(t, "the place of a tab", tab["place"])
		if !maps.Equal(place, want) {
			t.Errorf("the place of %v is %v, want %v", tab["tabId"], place, want)
		}
	}
}

func TestWindowToolTextsNameNoToolOfAnotherPlugin(t *testing.T) {
	w := newWindowTools(t)
	foreign := getForeignToolNames()
	visitEveryToolSchema(t, w.registry, func(string, string) {}, func(tool, text string) {
		for _, name := range foreign {
			if strings.Contains(text, name) {
				t.Errorf("a description of %s or its schema names %s: %q", tool, name, text)
			}
		}
	})
	handler, err := service.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "window-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	listing, err := tools.NewClient(hostcall.Endpoint{BaseURL: web.URL, Token: "window-token"}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listing.Tools {
		for _, sentence := range tool.Instructions {
			for _, name := range foreign {
				if strings.Contains(sentence, name) {
					t.Errorf("an instruction of %s names %s: %q", tool.GetName(), name, sentence)
				}
			}
		}
	}
}

func getForeignToolNames() []string {
	return []string{
		"note_read", "note_write", "note_heading_list", "note_search", "note_list", "note_title_list",
		"note_resolve", "note_rename", "note_move", "note_remove", "link_list", "link_add", "link_update",
		"link_remove", "file_read", "file_create", "library_read", "library_search", "vault_get", "vault_list",
		"vault_problem_list", "vault_add", "vault_rename", "vault_remove", "vault_open",
		"recognition_create", "transcript_create", "url_import",
		"card_add", "card_read", "card_edit", "card_remove", "card_current_get", "card_deck_create",
	}
}

func TestNoteFocusReturnsTheNoteKey(t *testing.T) {
	w := newWindowTools(t)
	answer := w.getAnswer(t, "note_focus", `{"path":"notes/Heron.md"}`)
	requireKeys(t, "the answer of note_focus", answer, []string{"note"}, []string{"focused"})
	note := requireObject(t, "answer[note]", answer["note"])
	if note["path"] != notePath || note["title"] != "Gray Heron" {
		t.Errorf("note = %v, want %s titled Gray Heron", answer["note"], notePath)
	}

	schema := getToolSchema(t, w.registry, "note_focus", true)
	requireKeys(t, "the output schema of note_focus", getSchemaProperties(t, schema), []string{"note"}, []string{"focused"})
}

func TestSourceFocusReturnsThePathAndTheHighlightCount(t *testing.T) {
	w := newWindowTools(t)
	answer := w.getAnswer(t, "source_focus", `{"path":"library/Bird Guide.pdf","passages":[{"start":1500,"length":80},{"start":400,"length":700}]}`)
	requireKeys(t, "the answer of source_focus", answer, []string{"path", "highlightCount"}, []string{"shown", "summary", "kind"})
	if answer["path"] != guidePath || answer["highlightCount"] != float64(2) {
		t.Errorf("source_focus returned %v, want path %s and highlightCount 2", answer, guidePath)
	}

	noPassages := w.getAnswer(t, "source_focus", `{"path":"library/Bird Guide.pdf"}`)
	requireKeys(t, "the answer of source_focus with no passages", noPassages, []string{"path", "highlightCount"}, []string{"shown", "summary"})
	if noPassages["highlightCount"] != float64(0) {
		t.Errorf("source_focus with no passages returned highlightCount %v, want 0", noPassages["highlightCount"])
	}

	schema := getToolSchema(t, w.registry, "source_focus", true)
	requireKeys(t, "the output schema of source_focus", getSchemaProperties(t, schema), []string{"path", "highlightCount"}, []string{"shown", "summary", "kind"})
}

// The window opens a note whole and highlights none of the passages given for it.
func TestSourceFocusOfANoteReturnsAHighlightCountOfZero(t *testing.T) {
	w := newWindowTools(t)
	answer := w.getAnswer(t, "source_focus", `{"path":"notes/Heron.md","passages":[{"start":4,"length":10},{"start":20,"length":5}]}`)
	requireKeys(t, "the answer of source_focus for a note", answer, []string{"path", "highlightCount"}, nil)
	if answer["path"] != notePath || answer["highlightCount"] != float64(0) {
		t.Errorf("source_focus of a note returned %v, want path %s and highlightCount 0", answer, notePath)
	}
}

func TestWindowTabActivateReturnsTheTabOnly(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs([]host.OpenTab{{ID: "tab-1", Kind: "note", Path: notePath, Title: "Gray Heron"}}, "tab-1")
	answer := w.getAnswer(t, "window_tab_activate", `{"tabId":"tab-1"}`)
	requireKeys(t, "the answer of window_tab_activate", answer, []string{"tab"}, []string{"success", "message"})
	tab := requireObject(t, "answer[tab]", answer["tab"])
	requireKeys(t, "the tab of window_tab_activate", tab, []string{"tabId"}, []string{"id"})
	if tab["tabId"] != "tab-1" || tab["path"] != notePath {
		t.Errorf("tab = %v, want tab-1 at %s", answer["tab"], notePath)
	}

	schema := getToolSchema(t, w.registry, "window_tab_activate", true)
	requireKeys(t, "the output schema of window_tab_activate", getSchemaProperties(t, schema), []string{"tab"}, []string{"success", "message"})
	requireKeys(t, "the output schema of the tab of window_tab_activate", getSchemaProperties(t, schema, "tab"),
		[]string{"tabId", "place"}, []string{"id", "location"})
}

func TestNoteEditPreviewReturnsTheDiffOnly(t *testing.T) {
	w := newWindowTools(t)
	answer := w.getAnswer(t, "note_edit_preview", `{"path":"notes/Heron.md","proposedText":"Herons fish at dawn."}`)
	requireKeys(t, "the answer of note_edit_preview", answer, []string{"path", "diff", "additions", "deletions"}, []string{"previewId", "status", "summary"})

	schema := getToolSchema(t, w.registry, "note_edit_preview", true)
	requireKeys(t, "the output schema of note_edit_preview", getSchemaProperties(t, schema), []string{"diff", "additions", "deletions"}, []string{"previewId", "status", "summary"})
}

func TestWindowToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	w := newWindowTools(t)
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, w.registry, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}

func TestWindowToolInstructionsContainNoNarrativeMarker(t *testing.T) {
	// The window tools are read from the handler the host starts, where the plugin mounts its registry.
	handler, err := service.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "window-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	listing, err := tools.NewClient(hostcall.Endpoint{BaseURL: web.URL, Token: "window-token"}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	hasSentence := false
	for _, tool := range listing.Tools {
		for _, sentence := range tool.Instructions {
			hasSentence = true
			for _, marker := range getNarrativeMarkers() {
				if marker.MatchString(sentence) {
					t.Errorf("an instruction of %s matches %s: %q", tool.GetName(), marker, sentence)
				}
			}
		}
	}
	if !hasSentence {
		t.Fatal("no window tool lists instructions")
	}
}

// lowerCamelCase is the form of every property name of a tool schema.
var lowerCamelCase = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestWindowToolSchemasNameEveryPropertyInLowerCamelCase(t *testing.T) {
	w := newWindowTools(t)
	visitEveryToolSchema(t, w.registry, func(tool, name string) {
		if !lowerCamelCase.MatchString(name) {
			t.Errorf("the schema of %s names the property %q, which is not lowerCamelCase", tool, name)
		}
	}, func(string, string) {})
}
