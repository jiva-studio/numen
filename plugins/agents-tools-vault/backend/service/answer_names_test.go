package service_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
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

func TestVaultListReturnsIsCurrentAndPathKeys(t *testing.T) {
	sky, registry := newSkyVault(t)
	garden := hosttest.New("garden-vault")
	garden.File("Beds.md", "# Beds\nRaised beds of oak.")
	sky.Beside(garden)

	var answer struct {
		Vaults []map[string]any `json:"vaults"`
	}
	decodeInto(t, callTool(t, registry, "garden-vault", "vault_list", `{}`), &answer)
	if len(answer.Vaults) != 2 {
		t.Fatalf("got %d vaults, want 2: %v", len(answer.Vaults), answer.Vaults)
	}
	for _, vault := range answer.Vaults {
		requireKeys(t, "a vault of vault_list", vault, []string{"id", "name", "path", "isCurrent"}, []string{"showing", "folder"})
		if want := vault["id"] == "garden-vault"; vault["isCurrent"] != want {
			t.Errorf("vault %v has isCurrent %v, want %v", vault["id"], vault["isCurrent"], want)
		}
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "vault_list", true), "vaults")
	requireKeys(t, "the output schema of a vault_list vault", properties, []string{"isCurrent", "path"}, []string{"showing", "folder"})
}

func TestVaultListDescriptionDoesNotNameTheShowingKey(t *testing.T) {
	_, registry := newSkyVault(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() == "vault_list" && strings.Contains(info.GetDescription(), "showing") {
			t.Errorf("the description of vault_list names showing: %s", info.GetDescription())
		}
	}
}

func TestVaultProblemListReturnsTheChecksKey(t *testing.T) {
	_, registry := newSkyVault(t)
	answer := decodeAnswer(t, callTool(t, registry, skyVault, "vault_problem_list", `{}`))
	requireKeys(t, "the answer of vault_problem_list", answer, []string{"problems", "checks"}, []string{"ran"})
	want := make([]any, 0, len(getDefaultChecks()))
	for _, check := range getDefaultChecks() {
		want = append(want, check)
	}
	if checks := requireList(t, "answer[checks]", answer["checks"]); !slices.Equal(checks, want) {
		t.Errorf("checks = %v, want %v", answer["checks"], want)
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "vault_problem_list", true))
	requireKeys(t, "the output schema of vault_problem_list", properties, []string{"checks"}, []string{"ran"})
}

func TestLibraryReadReturnsTheNextStartKey(t *testing.T) {
	l := newLibrary(t)
	res, err := l.registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_read", Arguments: []byte(`{"path":"` + guidePath + `","page":2}`)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		t.Fatal(res.GetResult().GetText())
	}
	answer := decodeAnswer(t, res.GetResult().GetStructured())
	requireKeys(t, "the answer of library_read", answer, []string{"nextStart"}, []string{"next"})
	if got, want := answer["nextStart"], float64(len(pageOne)+len(pageTwo)); got != want {
		t.Errorf("nextStart = %v, want %v", got, want)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.registry, "library_read", true))
	requireKeys(t, "the output schema of library_read", properties, []string{"nextStart"}, []string{"next"})
}

func TestLibrarySearchReturnsThePathKeyOfAMatch(t *testing.T) {
	l := newLibrary(t)
	res, err := l.registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_search", Arguments: []byte(`{"query":"heron"}`)})
	if err != nil {
		t.Fatal(err)
	}
	var answer struct {
		Matches []map[string]any `json:"matches"`
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Matches) != 3 {
		t.Fatalf("got %d matches, want 3: %s", len(answer.Matches), res.GetResult().GetStructured())
	}
	requireKeys(t, "a match of library_search", answer.Matches[0], []string{"path"}, []string{"source"})
	if answer.Matches[0]["path"] != guidePath {
		t.Errorf("the first match has path %v, want %s", answer.Matches[0]["path"], guidePath)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.registry, "library_search", true), "matches")
	requireKeys(t, "the output schema of a library_search match", properties, []string{"path"}, []string{"source"})
}

func TestLibrarySearchAcceptsTheFolderArgument(t *testing.T) {
	l := newLibrary(t)
	var narrowed librarySearchAnswer
	if failure := l.call(t, "library_search", `{"query":"heron","folder":"recordings/"}`, &narrowed); failure != "" {
		t.Fatal(failure)
	}
	if got := getMatchPaths(narrowed); !slices.Equal(got, []string{talkPath}) {
		t.Errorf("the passages under recordings/ are %v, want only %s", got, talkPath)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.registry, "library_search", false))
	requireKeys(t, "the input schema of library_search", properties, []string{"folder"}, []string{"pathPrefix", "path_prefix"})
}

func TestLibrarySearchFolderHasTheDescriptionOfTheNoteListFolder(t *testing.T) {
	_, registry := newGeologyVault(t)
	getDescription := func(tool string) string {
		folder, ok := getSchemaProperties(t, getToolSchema(t, registry, tool, false))["folder"].(map[string]any)
		if !ok {
			t.Fatalf("the input schema of %s has no folder", tool)
		}
		description, isText := folder["description"].(string)
		if !isText {
			t.Fatalf("the folder of %s has no description: %v", tool, folder)
		}
		return description
	}
	if search, list := getDescription("library_search"), getDescription("note_list"); search != list {
		t.Errorf("the folder of library_search is described as %q, want the text of note_list %q", search, list)
	}
}

func TestLibrarySearchRejectsThePathPrefixArgument(t *testing.T) {
	l := newLibrary(t)
	if failure := l.call(t, "library_search", `{"query":"heron","pathPrefix":"recordings/"}`, nil); !strings.Contains(failure, "pathPrefix is not known") {
		t.Errorf("library_search with pathPrefix gave %q, want the argument rejected as not known", failure)
	}
}

func TestLibrarySearchReturnsThePlaceKeyOfAMatch(t *testing.T) {
	l := newLibrary(t)
	answer := decodeAnswer(t, callTool(t, l.registry, testVault, "library_search", `{"query":"heron"}`))
	matches := requireList(t, "answer[matches]", answer["matches"])
	if len(matches) != 3 {
		t.Fatalf("matches = %v, want three", matches)
	}
	for _, m := range matches {
		match, isObject := m.(map[string]any)
		if !isObject {
			t.Fatalf("a match is %v, not an object", m)
		}
		requireKeys(t, "a match of library_search", match, nil, []string{"location"})
		if match["path"] == "notes/Herons.md" {
			requireKeys(t, "the note match of library_search", match, nil, []string{"place"})
			continue
		}
		requireKeys(t, "a file match of library_search", match, []string{"place"}, nil)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.registry, "library_search", true), "matches")
	requireKeys(t, "the output schema of a library_search match", properties, []string{"place"}, []string{"location"})
	requireExactSchemaKeys(t, "the output schema of a library_search place",
		getSchemaProperties(t, getToolSchema(t, l.registry, "library_search", true), "matches", "place"), getPlaceKeys())
}

func requireExactSchemaKeys(t *testing.T, what string, properties map[string]any, want []string) {
	t.Helper()
	got := make([]string, 0, len(properties))
	for key := range properties {
		got = append(got, key)
	}
	slices.Sort(got)
	want = slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("%s has the keys %v, want %v", what, got, want)
	}
}

func newPlacedLibrary(t *testing.T) library {
	t.Helper()
	l := newLibrary(t)
	l.search.SetIndexedPassages([]host.Passage{
		{Path: guidePath, Text: "egret on page five", Location: "Waders", Start: 10, Length: 18, Selector: `{"page":5,"boxes":[[0,0,1,1]]}`},
		{Path: "library/Marsh Book.epub", Text: "egret in the reeds", Location: "The Reed Beds", Start: 20, Length: 18, Selector: `{"cfi":"epubcfi(/6/4!/4/2/1:0)"}`},
		{Path: talkPath, Text: "egret at the start", Location: "00:00", Start: 0, Length: 18, Selector: `{"t":[0,3.5]}`},
		{Path: "library/Pond.pdf", Text: "egret by the pond", Location: "Ponds", Start: 0, Length: 17, Selector: `{"page":"two"}`},
	})
	return l
}

func TestLibrarySearchReadsThePlaceFromTheSelectorAndLocation(t *testing.T) {
	l := newPlacedLibrary(t)
	var out librarySearchAnswer
	if failure := l.call(t, "library_search", `{"query":"egret"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	places := map[string]*placeAnswer{}
	for _, m := range out.Matches {
		places[m.Path] = m.Place
	}
	if len(places) != 4 {
		t.Fatalf("matches = %+v, want four", out.Matches)
	}
	if pdf := places[guidePath]; getPlacePage(pdf) != 5 || pdf.Chapter != "" || pdf.CFI != "" || pdf.PositionSeconds != nil {
		t.Errorf("the place of the PDF passage is %+v, want page 5 and no chapter", pdf)
	}
	if book := places["library/Marsh Book.epub"]; book == nil || book.CFI != "epubcfi(/6/4!/4/2/1:0)" || book.Chapter != "The Reed Beds" || book.Page != 0 {
		t.Errorf("the place of the EPUB passage is %+v, want its CFI in The Reed Beds", book)
	}
	if talk := places[talkPath]; getPlaceSeconds(talk) != 0 || talk.Chapter != "" {
		t.Errorf("the place of the recording passage is %+v, want positionSeconds 0 and no chapter", talk)
	}
	if pond := places["library/Pond.pdf"]; pond != nil {
		t.Errorf("the place of the passage with a selector that does not decode and no cfi is %+v, want none", pond)
	}
}

func TestLibrarySearchReturnsPositionSecondsZero(t *testing.T) {
	l := newPlacedLibrary(t)
	answer := decodeAnswer(t, callTool(t, l.registry, testVault, "library_search", `{"query":"egret"}`))
	for _, m := range requireList(t, "answer[matches]", answer["matches"]) {
		match, isObject := m.(map[string]any)
		if !isObject || match["path"] != talkPath {
			continue
		}
		place, isPlace := match["place"].(map[string]any)
		if !isPlace {
			t.Fatalf("the recording match is %v, want a place", match)
		}
		if seconds, ok := place["positionSeconds"]; !ok || seconds != float64(0) {
			t.Errorf("the place of the recording match is %v, want positionSeconds 0", place)
		}
		return
	}
	t.Errorf("library_search returned no match of %s: %v", talkPath, answer)
}

func TestLibrarySearchReturnsTheLineCountedFromOne(t *testing.T) {
	l := newLibrary(t)
	l.search.SetIndexedPassages([]host.Passage{
		{Path: "notes/Herons.md", Text: "Herons fish at dusk in the marsh.", Start: 0, Length: 33, Line: 0},
		{Path: "Entropy.md", Text: "Disorder grows in the marsh.", Start: 10, Length: 28, Line: 1},
	})
	answer := decodeAnswer(t, callTool(t, l.registry, testVault, "library_search", `{"query":"marsh"}`))
	lines := map[any]any{}
	for _, m := range requireList(t, "answer[matches]", answer["matches"]) {
		if match, isObject := m.(map[string]any); isObject {
			lines[match["path"]] = match["line"]
		}
	}
	if lines["notes/Herons.md"] != float64(1) {
		t.Errorf("the passage on the first line of the body has the line %v, want 1: %v", lines["notes/Herons.md"], answer)
	}
	if lines["Entropy.md"] != float64(2) {
		t.Errorf("the passage on the second line of the body has the line %v, want 2: %v", lines["Entropy.md"], answer)
	}
}

func TestLibraryReadReturnsRangesWithAPlace(t *testing.T) {
	l := newLibrary(t)
	answer := decodeAnswer(t, callTool(t, l.registry, testVault, "library_read", `{"path":"`+talkPath+`"}`))
	requireKeys(t, "the answer of library_read", answer, []string{"ranges"}, []string{"locations"})
	ranges := requireList(t, "answer[ranges]", answer["ranges"])
	if len(ranges) != 2 {
		t.Fatalf("ranges = %v, want two", ranges)
	}
	first, isObject := ranges[0].(map[string]any)
	if !isObject {
		t.Fatalf("a range is %v, not an object", ranges[0])
	}
	requireExactSchemaKeys(t, "the first range of library_read", first, []string{"start", "length", "place"})
	place, isPlace := first["place"].(map[string]any)
	if !isPlace {
		t.Fatalf("the first range is %v, want a place", first)
	}
	if seconds, ok := place["positionSeconds"]; !ok || seconds != float64(0) || place["endSeconds"] != 4.5 || len(place) != 2 {
		t.Errorf("the place of the first range is %v, want positionSeconds 0 and endSeconds 4.5 alone", place)
	}

	schema := getToolSchema(t, l.registry, "library_read", true)
	requireKeys(t, "the output schema of library_read", getSchemaProperties(t, schema), []string{"ranges"}, []string{"locations"})
	requireExactSchemaKeys(t, "the output schema of a library_read range", getSchemaProperties(t, schema, "ranges"), []string{"start", "length", "place"})
	requireExactSchemaKeys(t, "the output schema of a library_read place", getSchemaProperties(t, schema, "ranges", "place"), getPlaceKeys())
}

func TestLibraryReadReturnsThePageAndTheCFIOfARange(t *testing.T) {
	l := newLibrary(t)
	const bookPath = "library/Marsh Book.epub"
	file := []byte("PK\x03\x04epub body")
	l.vault.File(bookPath, string(file))
	l.texts.File(bookPath, file)
	putText(t, l.texts, readerID, bookPath, "book-text", file, []domain.TextSegment{
		{Text: "The reed beds at dawn. ", Selector: `{"cfi":"epubcfi(/6/4!/4/2/1:0)"}`},
		{Text: "No place here. "},
	})
	var book libraryReadAnswer
	if failure := l.call(t, "library_read", `{"path":"`+bookPath+`"}`, &book); failure != "" {
		t.Fatal(failure)
	}
	if len(book.Ranges) != 1 || book.Ranges[0].Place == nil || book.Ranges[0].Place.CFI != "epubcfi(/6/4!/4/2/1:0)" {
		t.Errorf("the ranges of the book are %+v, want one with the CFI and none for the segment with no place", book.Ranges)
	}

	var guide libraryReadAnswer
	if failure := l.call(t, "library_read", `{"path":"`+guidePath+`","page":2}`, &guide); failure != "" {
		t.Fatal(failure)
	}
	if len(guide.Ranges) != 1 || getPlacePage(guide.Ranges[0].Place) != 2 || guide.Ranges[0].Length != len(pageTwo) {
		t.Errorf("the ranges of page 2 are %+v, want one range of page 2 over its two segments", guide.Ranges)
	}
}

func TestLibraryReadOfAFileWithNoTextVersionNamesNoTool(t *testing.T) {
	l := newLibrary(t)
	failure := l.call(t, "library_read", `{"path":"`+scanPath+`"}`, nil)
	if !strings.Contains(failure, "no text version") {
		t.Fatalf("library_read of a scan with no text version gave %q, want no text version", failure)
	}
	for _, name := range getForeignToolNames() {
		if strings.Contains(failure, name) {
			t.Errorf("library_read of a scan with no text version gave %q, which names %s", failure, name)
		}
	}
	if !strings.Contains(failure, "recognition") || !strings.Contains(failure, "transcription") {
		t.Errorf("library_read of a scan with no text version gave %q, want a recognition or a transcription named", failure)
	}
}

func TestVaultToolTextsNameNoToolOfAnotherPlugin(t *testing.T) {
	_, withVaultList := newGeologyVault(t)
	for _, registry := range []*tools.Registry{newLibrary(t).registry, withVaultList} {
		visitEveryToolSchema(t, registry, func(string, string) {}, func(tool, text string) {
			for _, name := range getForeignToolNames() {
				if strings.Contains(text, name) {
					t.Errorf("a description of %s or its schema names %s: %q", tool, name, text)
				}
			}
		})
	}
	for _, tool := range listVaultTools(t).Tools {
		for _, sentence := range tool.Instructions {
			for _, name := range getForeignToolNames() {
				if strings.Contains(sentence, name) {
					t.Errorf("an instruction of %s names %s: %q", tool.Name, name, sentence)
				}
			}
		}
	}
}

func TestLibraryReadDescriptionNamesRecognitionAndTranscription(t *testing.T) {
	l := newLibrary(t)
	listing, err := l.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() != "library_read" {
			continue
		}
		description := info.GetDescription()
		if strings.Contains(description, "names the tool") {
			t.Errorf("the description of library_read is %q, want no error that names a tool", description)
		}
		if !strings.Contains(description, "recognized") || !strings.Contains(description, "transcribed") {
			t.Errorf("the description of library_read is %q, want the file recognized or transcribed first", description)
		}
	}
}

func getForeignToolNames() []string {
	return []string{
		"recognition_create", "transcript_create", "url_import",
		"note_focus", "source_focus", "window_tab_list", "window_tab_activate", "note_edit_preview",
		"card_add", "card_read", "card_edit", "card_remove", "card_current_get", "card_deck_create",
	}
}

func TestNoteReadAndFileReadReturnTheTextKey(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	note := decodeAnswer(t, callTool(t, registry, testVault, "note_read", `{"path":"Heat.md"}`))
	requireKeys(t, "the answer of note_read", note, []string{"text", "body"}, []string{"content"})
	if want := "# Heat\nHeat flows from hot to cold.\nReferences [[Entropy]]."; note["text"] != want {
		t.Errorf("note_read returned the text %q, want the whole file %q", note["text"], want)
	}
	file := decodeAnswer(t, callTool(t, registry, testVault, "file_read", `{"path":"data.txt"}`))
	requireKeys(t, "the answer of file_read", file, []string{"text"}, []string{"content"})

	requireKeys(t, "the output schema of note_read", getSchemaProperties(t, getToolSchema(t, registry, "note_read", true)),
		[]string{"text", "body"}, []string{"content"})
	requireKeys(t, "the output schema of file_read", getSchemaProperties(t, getToolSchema(t, registry, "file_read", true)),
		[]string{"text"}, []string{"content"})
}

func TestNoteWriteAndFileCreateTakeTheTextArgument(t *testing.T) {
	v, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	callTool(t, registry, testVault, "note_write", `{"path":"Orbit.md","text":"# Orbit\nOrbits curve."}`)
	requireContent(t, v, "Orbit.md", "# Orbit\nOrbits curve.")
	callTool(t, registry, testVault, "file_create", `{"path":"orbits.txt","text":"perihelion"}`)
	requireContent(t, v, "orbits.txt", "perihelion")

	for _, name := range []string{"note_write", "file_create"} {
		requireKeys(t, "the input schema of "+name, getSchemaProperties(t, getToolSchema(t, registry, name, false)),
			[]string{"text"}, []string{"content"})
	}
}

func TestNoteWriteAndFileCreateRejectTheContentArgument(t *testing.T) {
	v, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for name, arguments := range map[string]string{
		"note_write":  `{"path":"Comet.md","content":"# Comet\nA tail of dust."}`,
		"file_create": `{"path":"comet.txt","text":"a tail of dust","content":"a tail of dust"}`,
	} {
		text := getRejectionText(t, registry, testVault, name, arguments)
		if !strings.Contains(text, "content is not known") {
			t.Errorf("%s with the argument content gave the error %q, want content rejected as not known", name, text)
		}
	}
	requireAbsent(t, v, "Comet.md")
	requireAbsent(t, v, "comet.txt")
}

func TestNoteHeadingListReturnsTheStartAndTheLineCountedFromOne(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	answer := decodeAnswer(t, callTool(t, registry, testVault, "note_heading_list", `{"path":"Entropy.md"}`))
	headings := requireList(t, "answer[headings]", answer["headings"])
	if len(headings) != 2 {
		t.Fatalf("headings = %v, want two", headings)
	}
	body := "# Entropy\nEntropy is thermodynamic disorder.\n## Second Law\nEntropy increases.\nSee also [[Heat]]."
	for i, want := range []struct {
		line  float64
		start int
	}{{1, 0}, {3, strings.Index(body, "## Second Law")}} {
		heading, isObject := headings[i].(map[string]any)
		if !isObject {
			t.Fatalf("a heading is %v, not an object", headings[i])
		}
		requireKeys(t, "a heading of note_heading_list", heading, []string{"start", "line"}, []string{"offset"})
		if heading["line"] != want.line || heading["start"] != float64(want.start) {
			t.Errorf("heading %d is %v, want line %v and start %d", i, heading, want.line, want.start)
		}
	}

	requireKeys(t, "the output schema of a heading", getSchemaProperties(t, getToolSchema(t, registry, "note_heading_list", true), "headings"),
		[]string{"start", "line"}, []string{"offset"})
}

func TestNoteSearchReturnsTheLineCountedInTheBody(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"matches"`
	}
	for query, want := range map[string]int{"thermodynamic": 2, "Second Law": 3} {
		decodeInto(t, callTool(t, registry, testVault, "note_search", `{"query":"`+query+`"}`), &out)
		found := false
		for _, m := range out.Matches {
			if m.Path == "Entropy.md" {
				found = true
				if m.Line != want {
					t.Errorf("the match of %q in Entropy.md has the line %d, want %d of the body", query, m.Line, want)
				}
			}
		}
		if !found {
			t.Errorf("note_search of %q did not find Entropy.md: %+v", query, out.Matches)
		}
	}
}

func TestNoteSearchReturnsLineOneForAMatchOnTheFirstBodyLine(t *testing.T) {
	v, svc := setupTestVault()
	v.File("Comet.md", "---\ntitle: Comet\n---\nIcy nucleus of dust.\nA tail.")
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"matches"`
	}
	decodeInto(t, callTool(t, registry, testVault, "note_search", `{"query":"icy nucleus"}`), &out)
	for _, m := range out.Matches {
		if m.Path == "Comet.md" {
			if m.Line != 1 {
				t.Errorf("the match on the first body line of Comet.md has the line %d, want 1", m.Line)
			}
			return
		}
	}
	t.Errorf("note_search did not find Comet.md: %+v", out.Matches)
}

func TestVaultGetFindsTheVaultByItsNameFolderOrIdentifier(t *testing.T) {
	for setup, isListed := range map[string]bool{"with a vault list": true, "without a vault list": false} {
		t.Run(setup, func(t *testing.T) {
			geology, _, withList := newTwoVaults(t)
			registry := withList
			if !isListed {
				unlisted, err := service.New(vaultsOf(geology)).Registry()
				if err != nil {
					t.Fatal(err)
				}
				registry = unlisted
			}
			for how, address := range getGardenAddresses() {
				answer := decodeAnswer(t, callTool(t, registry, geologyVault, "vault_get", fmt.Sprintf(`{"vault":%q}`, address)))
				if answer["id"] != gardenID || answer["name"] != gardenName || answer["path"] != gardenFolder {
					t.Errorf("vault_get by its %s %q returned %v, want the garden vault", how, address, answer)
				}
			}
			own := decodeAnswer(t, callTool(t, registry, geologyVault, "vault_get", `{}`))
			if own["id"] != geologyVault {
				t.Errorf("vault_get with no vault returned %v, want the vault of the call", own)
			}
		})
	}
}

func TestVaultGetAcceptsTheVaultArgumentAndRejectsTheID(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	requireKeys(t, "the input schema of vault_get", getSchemaProperties(t, getToolSchema(t, registry, "vault_get", false)),
		[]string{"vault"}, []string{"id"})
	text := getRejectionText(t, registry, geologyVault, "vault_get", `{"id":"`+gardenID+`"}`)
	if !strings.Contains(text, "id is not known") {
		t.Errorf("vault_get with the argument id gave the error %q, want id rejected as not known", text)
	}
}

func TestVaultProblemListNamesTheChecksInItsDescriptionOnly(t *testing.T) {
	_, registry := newSkyVault(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	checks := []string{"parse", "frontmatter", "ambiguous", "dangling"}
	for _, info := range listing.Tools {
		if info.GetName() != "vault_problem_list" {
			continue
		}
		for _, check := range checks {
			if !regexp.MustCompile(`\b` + check + `\b`).MatchString(info.GetDescription()) {
				t.Errorf("the description of vault_problem_list does not name the check %s: %q", check, info.GetDescription())
			}
		}
	}
	field, ok := getSchemaProperties(t, getToolSchema(t, registry, "vault_problem_list", false))["checks"].(map[string]any)
	if !ok {
		t.Fatal("the input schema of vault_problem_list has no checks")
	}
	description, isText := field["description"].(string)
	if !isText {
		t.Fatalf("the checks of vault_problem_list have no description: %v", field)
	}
	for _, check := range checks {
		if regexp.MustCompile(`\b` + check + `\b`).MatchString(description) {
			t.Errorf("the checks of vault_problem_list are described as %q, which names the check %s", description, check)
		}
	}
	for _, tool := range listVaultTools(t).Tools {
		if tool.Name != "vault_problem_list" {
			continue
		}
		for _, sentence := range tool.Instructions {
			for _, check := range checks[:3] {
				if regexp.MustCompile(`\b` + check + `\b`).MatchString(sentence) {
					t.Errorf("an instruction of vault_problem_list names the check %s: %q", check, sentence)
				}
			}
		}
	}
}

func TestFileCreateReturnsThePathAndSizeWithoutSuccess(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	answer := decodeAnswer(t, callTool(t, registry, testVault, "file_create", `{"path":"orbits.txt","text":"perihelion"}`))
	requireKeys(t, "the answer of file_create", answer, []string{"path", "size"}, []string{"success"})
	if answer["path"] != "orbits.txt" || answer["size"] != float64(len("perihelion")) {
		t.Errorf("file_create returned %v, want orbits.txt of 10 bytes", answer)
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "file_create", true))
	requireKeys(t, "the output schema of file_create", properties, []string{"path", "size"}, []string{"success"})
}

func TestVaultToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	markers := getNarrativeMarkers()
	_, withVaultList := newGeologyVault(t)
	for _, registry := range []*tools.Registry{newLibrary(t).registry, withVaultList} {
		visitEveryToolSchema(t, registry, func(string, string) {}, func(tool, description string) {
			for _, marker := range markers {
				if marker.MatchString(description) {
					t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
				}
			}
		})
	}
}

func TestVaultInstructionsNameTheKeysInCamelCase(t *testing.T) {
	instructions := joinToolInstructions(listVaultTools(t))
	for _, want := range []string{"`uri`", "kinds", "isCurrent"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not name %q:\n%s", want, instructions)
		}
	}
	for _, old := range []string{"showing", "path_prefix", "pathPrefix", "next_offset", "next_start", "is_current", "note_count"} {
		if strings.Contains(instructions, old) {
			t.Errorf("the instructions name %q:\n%s", old, instructions)
		}
	}
}

func TestVaultToolInstructionsContainNoNarrativeMarker(t *testing.T) {
	listing := listVaultTools(t)
	if joinToolInstructions(listing) == "" {
		t.Fatal("no tool lists instructions")
	}
	for _, tool := range listing.Tools {
		for _, sentence := range tool.Instructions {
			for _, marker := range getNarrativeMarkers() {
				if marker.MatchString(sentence) {
					t.Errorf("an instruction of %s matches %s: %q", tool.Name, marker, sentence)
				}
			}
		}
	}
}

// lowerCamelCase is the form of every property name of a tool schema.
var lowerCamelCase = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestVaultToolSchemasNameEveryPropertyInLowerCamelCase(t *testing.T) {
	_, withVaultList := newGeologyVault(t)
	for _, registry := range []*tools.Registry{newLibrary(t).registry, withVaultList} {
		visitEveryToolSchema(t, registry, func(tool, name string) {
			if !lowerCamelCase.MatchString(name) {
				t.Errorf("the schema of %s names the property %q, which is not lowerCamelCase", tool, name)
			}
		}, func(string, string) {})
	}
}
