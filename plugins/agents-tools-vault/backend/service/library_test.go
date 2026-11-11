package service_test

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// The invented library: a PDF of a bird guide with a text layer of three pages, a scan with no text
// version, an audio recording with a transcript, and two notes.
const (
	guidePath  = "library/Bird Guide.pdf"
	scanPath   = "library/Old Scan.pdf"
	talkPath   = "recordings/Marsh Talk.mp3"
	pdfBytes   = "%PDF-1.7\n%\xe2\xe3\xcf\xd3binary body"
	talkBytes  = "ID3\x04\x00\xff\xfbaudio"
	readerID   = "org.example.reader"
	scribeID   = "org.example.scribe"
	pageOne    = "Chapter one. The marsh at dawn is quiet. "
	pageTwo    = "The gray heron stands in the reeds and waits for fish. "
	pageThree  = "Herons nest in tall trees near the water. "
	talkSecond = "Welcome to the marsh talk. "
	talkThird  = "Today we speak about the heron. "
)

// placeAnswer is a place a tool reports, with the tool's JSON keys.
type placeAnswer struct {
	Page            int      `json:"page"`
	PageCount       int      `json:"pageCount"`
	PositionSeconds *float64 `json:"positionSeconds"`
	CFI             string   `json:"cfi"`
	Chapter         string   `json:"chapter"`
}

func getPlaceKeys() []string {
	return []string{"page", "pageCount", "positionSeconds", "endSeconds", "cfi", "chapter"}
}

// librarySearchAnswer is the result of library_search with the tool's JSON keys.
type librarySearchAnswer struct {
	Query   string `json:"query"`
	Matches []struct {
		Path   string       `json:"path"`
		Title  string       `json:"title"`
		Place  *placeAnswer `json:"place"`
		Text   string       `json:"text"`
		Start  int          `json:"start"`
		Length int          `json:"length"`
		Line   int          `json:"line"`
		URI    string       `json:"uri"`
	} `json:"matches"`
}

// libraryReadAnswer is the result of library_read with the tool's JSON keys.
type libraryReadAnswer struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Version   string `json:"version"`
	Text      string `json:"text"`
	Start     int    `json:"start"`
	Length    int    `json:"length"`
	Size      int    `json:"size"`
	NextStart *int   `json:"nextStart"`
	Ranges    []struct {
		Start  int          `json:"start"`
		Length int          `json:"length"`
		Place  *placeAnswer `json:"place"`
	} `json:"ranges"`
}

// fileReadAnswer is the result of file_read with the tool's JSON keys.
type fileReadAnswer struct {
	Path        string `json:"path"`
	Text        string `json:"text"`
	Start       int    `json:"start"`
	Length      int    `json:"length"`
	Size        int    `json:"size"`
	Fingerprint string `json:"fingerprint"`
}

func getPlacePage(place *placeAnswer) int {
	if place == nil {
		return 0
	}
	return place.Page
}

func getPlaceSeconds(place *placeAnswer) float64 {
	if place == nil || place.PositionSeconds == nil {
		return -1
	}
	return *place.PositionSeconds
}

type library struct {
	vault    *hosttest.Vault
	texts    *hosttest.Texts
	search   *hosttest.Search
	registry *tools.Registry
}

func newLibrary(t *testing.T) library {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File("Entropy.md", "---\ntitle: Entropy Concept\n---\n# Entropy\nDisorder grows.")
	vault.File("notes/Herons.md", "---\ntitle: Herons of the Marsh\n---\nHerons fish at dusk.")
	vault.File(guidePath, pdfBytes)
	vault.File(scanPath, pdfBytes+"scan")
	vault.File(talkPath, talkBytes)
	vault.File("data.txt", "line 1\nline 2\nline 3\n")

	texts := hosttest.NewTexts(testVault, readerID)
	texts.File(guidePath, []byte(pdfBytes))
	texts.File(talkPath, []byte(talkBytes))
	putText(t, texts, readerID, guidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: pageOne, Selector: `{"page":1,"boxes":[[0,0,1,1]]}`},
		{Text: pageTwo[:20], Selector: `{"page":2}`},
		{Text: pageTwo[20:], Selector: `{"page":2}`},
		{Text: pageThree, Selector: `{"page":3}`},
	})
	putText(t, texts.As(scribeID), scribeID, talkPath, "transcript", []byte(talkBytes), []domain.TextSegment{
		{Text: talkSecond, Selector: `{"t":[0,4.5]}`},
		{Text: talkThird, Selector: `{"t":[4.5,9]}`},
	})

	search := hosttest.NewSearch(testVault)
	search.SetIndexedPassages([]host.Passage{
		{Path: guidePath, Text: "The gray heron stands in the reeds", Location: "page 2", Start: 41, Length: 56, Selector: `{"page":2}`},
		{Path: "notes/Herons.md", Text: "A heron fishes at dusk.", Start: 0, Length: 23, Line: 1},
		{Path: talkPath, Text: "Today we speak about the heron.", Location: "00:04", Start: 27, Length: 32, Selector: `{"t":[4.5,9]}`},
	})

	svc := service.New(vaultsOf(vault)).WithLibrary(libraryOf(texts, search))
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	return library{vault: vault, texts: texts, search: search, registry: registry}
}

func libraryOf(texts *hosttest.Texts, search *hosttest.Search) func(domain.VaultID) service.Library {
	return func(id domain.VaultID) service.Library {
		on := texts.With(id)
		return service.Library{Versions: on, Texts: on, Search: search.With(id)}
	}
}

func putText(t *testing.T, texts *hosttest.Texts, pluginID, path, kind string, file []byte, segments []domain.TextSegment) {
	t.Helper()
	name := kind + ".jsonl"
	if err := host.WriteText(t.Context(), texts.GetStore(pluginID), name, segments); err != nil {
		t.Fatal(err)
	}
	if _, err := texts.PutVersion(t.Context(), host.TextPut{
		Path: path, Key: kind, Kind: domain.TextKind(kind), InputHash: domain.HashContent(file), File: name,
	}); err != nil {
		t.Fatal(err)
	}
}

func (l library) call(t *testing.T, name, arguments string, out any) (failure string) {
	t.Helper()
	res, err := l.registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		return res.GetResult().GetText()
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("%s gave error code %v", name, res.GetError())
	}
	if out != nil {
		if err := json.Unmarshal(res.GetResult().GetStructured(), out); err != nil {
			t.Fatalf("%s: %v in %s", name, err, res.GetResult().GetStructured())
		}
	}
	return ""
}

func TestLibrarySearchFindsPDFPassageWithPageAndAddress(t *testing.T) {
	l := newLibrary(t)
	var out librarySearchAnswer
	if failure := l.call(t, "library_search", `{"query":"heron"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Matches) != 3 || out.Query != "heron" {
		t.Fatalf("matches %+v", out.Matches)
	}
	pdf := out.Matches[0]
	if pdf.Path != guidePath || pdf.Title != "Bird Guide" || pdf.Text != "The gray heron stands in the reeds" ||
		pdf.Start != 41 || pdf.Length != 56 {
		t.Errorf("the PDF passage is %+v", pdf)
	}
	if getPlacePage(pdf.Place) != 2 || pdf.Place.Chapter != "" {
		t.Errorf("the place of the PDF passage is %+v, want page 2 and no chapter", pdf.Place)
	}
	if want := "numen:" + url.PathEscape(guidePath) + "?start=41&length=56"; pdf.URI != want {
		t.Errorf("the address is %q, want %q", pdf.URI, want)
	}
	note := out.Matches[1]
	if note.Title != "Herons of the Marsh" || note.Path != "notes/Herons.md" || note.Place != nil {
		t.Errorf("the note passage is %+v, want the title of the note and no place", note)
	}
	talk := out.Matches[2]
	if talk.Path != talkPath || talk.Title != "Marsh Talk" {
		t.Errorf("the transcript passage is %+v", talk)
	}
	if getPlaceSeconds(talk.Place) != 4.5 || talk.Place.Chapter != "" {
		t.Errorf("the place of the transcript passage is %+v, want 4.5 seconds and no chapter", talk.Place)
	}
}

func TestLibrarySearchMatchHasURIKeyAndNoAtKey(t *testing.T) {
	l := newLibrary(t)
	res, err := l.registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_search", Arguments: []byte(`{"query":"heron"}`)})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}
	matches, isList := out["matches"].([]any)
	if !isList || len(matches) != 3 {
		t.Fatalf("matches %v, want three passages", out["matches"])
	}
	for _, m := range matches {
		match, isObject := m.(map[string]any)
		if !isObject {
			t.Fatalf("a match is %v, want an object", m)
		}
		address, isText := match["uri"].(string)
		if !isText || !strings.HasPrefix(address, "numen:") {
			t.Errorf("the match %v has uri %v, want a numen: address", match, match["uri"])
		}
		if _, hasAt := match["at"]; hasAt {
			t.Errorf("the match %v still has the key at", match)
		}
	}
}

func TestLibrarySearchSchemaNamesTheAddressURI(t *testing.T) {
	l := newLibrary(t)
	infos, err := l.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var schema []byte
	for _, info := range infos.Tools {
		if info.GetName() == "library_search" {
			schema = info.GetOutputSchema()
		}
	}
	var output struct {
		Properties struct {
			Matches struct {
				Items struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"items"`
			} `json:"matches"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &output); err != nil {
		t.Fatalf("%v in %s", err, schema)
	}
	properties := output.Properties.Matches.Items.Properties
	if _, hasURI := properties["uri"]; !hasURI {
		t.Errorf("the schema of a match does not name uri: %s", schema)
	}
	if _, hasAt := properties["at"]; hasAt {
		t.Errorf("the schema of a match still names at: %s", schema)
	}
}

func TestLibrarySearchLimitsItToAKindAndAFolder(t *testing.T) {
	l := newLibrary(t)
	var files, notes, narrowed librarySearchAnswer
	l.call(t, "library_search", `{"query":"heron","kinds":["file"]}`, &files)
	l.call(t, "library_search", `{"query":"heron","kinds":["note"]}`, &notes)
	l.call(t, "library_search", `{"query":"heron","folder":"recordings/","limit":1}`, &narrowed)
	if got := getMatchPaths(files); !slices.Equal(got, []string{guidePath, talkPath}) {
		t.Errorf("the files that match are %v", got)
	}
	if got := getMatchPaths(notes); !slices.Equal(got, []string{"notes/Herons.md"}) {
		t.Errorf("the notes that match are %v", got)
	}
	if got := getMatchPaths(narrowed); !slices.Equal(got, []string{talkPath}) {
		t.Errorf("one passage under recordings/ is %v", got)
	}
}

func getMatchPaths(out librarySearchAnswer) []string {
	var got []string
	for _, m := range out.Matches {
		got = append(got, m.Path)
	}
	return got
}

func TestLibrarySearchRejectsUnknownKindAndBlankQuery(t *testing.T) {
	l := newLibrary(t)
	failure := l.call(t, "library_search", `{"query":"heron","kinds":["book"]}`, nil)
	if !strings.Contains(failure, `"book"`) || !strings.Contains(failure, "note") || !strings.Contains(failure, "file") {
		t.Errorf("failure %q, want the kind named and the two kinds offered", failure)
	}
	var blank librarySearchAnswer
	if failure := l.call(t, "library_search", `{"query":"   "}`, &blank); failure != "" || len(blank.Matches) != 0 {
		t.Errorf("a blank query gave %q and %v", failure, blank.Matches)
	}
}

func TestLibrarySearchUsesTheNamedVault(t *testing.T) {
	vault := hosttest.New(testVault)
	garden := hosttest.New("garden-vault")
	vault.Beside(garden)
	search := hosttest.NewSearch(testVault, "garden-vault")
	search.SetIndexedPassages([]host.Passage{{Path: "Physics.md", Text: "heron physics", Start: 0, Length: 13}})
	search.With("garden-vault").SetIndexedPassages([]host.Passage{{Path: "Beds.md", Text: "heron beds", Start: 0, Length: 10}})
	registry, err := service.New(vaultsOf(vault)).WithLibrary(libraryOf(hosttest.NewTexts(testVault, readerID), search)).Registry()
	if err != nil {
		t.Fatal(err)
	}

	for vaultID, want := range map[string]string{testVault: "Physics.md", "garden-vault": "Beds.md"} {
		res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: "library_search", Arguments: []byte(`{"query":"heron"}`)})
		if err != nil {
			t.Fatal(err)
		}
		var out librarySearchAnswer
		if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil || len(out.Matches) != 1 || out.Matches[0].Path != want {
			t.Errorf("the call naming %s returned %s (%v), want %s alone", vaultID, res.GetResult().GetStructured(), err, want)
		}
	}
}

func TestLibrarySearchReportsMissingIndexPermission(t *testing.T) {
	l := newLibrary(t)
	l.search.Deny(host.PermissionIndexRead)
	failure := l.call(t, "library_search", `{"query":"heron"}`, nil)
	if !strings.Contains(failure, "index:read") {
		t.Errorf("failure %q, want the permission named", failure)
	}
}

func TestLibraryReadStopsAtCharacterBoundary(t *testing.T) {
	l := newLibrary(t)
	cyrillic := hosttest.NewTexts(testVault, readerID)
	cyrillic.File("ru.pdf", []byte(pdfBytes))
	putText(t, cyrillic, readerID, "ru.pdf", "page-text", []byte(pdfBytes), []domain.TextSegment{{Text: "Ачарьи говорят", Selector: `{"page":1}`}})
	l.vault.File("ru.pdf", pdfBytes)
	svc := service.New(vaultsOf(l.vault)).WithLibrary(libraryOf(cyrillic, l.search))
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_read", Arguments: []byte(`{"path":"ru.pdf","start":0,"length":3}`)})
	if err != nil {
		t.Fatal(err)
	}
	var slice libraryReadAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &slice); err != nil {
		t.Fatal(err)
	}
	if slice.Text != "А" || slice.Length != 2 || slice.NextStart == nil || *slice.NextStart != 2 {
		t.Errorf("3 bytes of a two-byte letter text gave %q for %d, want the whole letter before the cut", slice.Text, slice.Length)
	}
}

func TestLibraryReadReturnsTranscriptTimes(t *testing.T) {
	l := newLibrary(t)
	var out libraryReadAnswer
	l.call(t, "library_read", `{"path":"`+talkPath+`"}`, &out)
	if out.Text != talkSecond+talkThird || out.Kind != "transcript" || len(out.Ranges) != 2 {
		t.Fatalf("transcript %+v", out)
	}
	if first := out.Ranges[0]; getPlaceSeconds(first.Place) != 0 || first.Start != 0 || first.Length != len(talkSecond) {
		t.Errorf("first range %+v, want the place at 0 seconds", first)
	}
	if second := out.Ranges[1]; getPlaceSeconds(second.Place) != 4.5 || second.Start != len(talkSecond) || second.Length != len(talkThird) {
		t.Errorf("second range %+v, want the place at 4.5 seconds", second)
	}
}

func TestLibraryReadReadsATextFileThatNeedsNoTextVersion(t *testing.T) {
	l := newLibrary(t)
	var out libraryReadAnswer
	if failure := l.call(t, "library_read", `{"path":"data.txt"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != "line 1\nline 2\nline 3\n" || out.Size != 21 || out.Kind != "" || len(out.Ranges) != 0 {
		t.Errorf("text file %+v", out)
	}
}

func TestLibraryReadRejectsMissingPageAndMissingFile(t *testing.T) {
	l := newLibrary(t)
	if failure := l.call(t, "library_read", `{"path":"`+guidePath+`","page":9}`, nil); !strings.Contains(failure, "page 9") || !strings.Contains(failure, "1 to 3") {
		t.Errorf("failure %q, want the page and the pages there are", failure)
	}
	if failure := l.call(t, "library_read", `{"path":"`+guidePath+`","start":99999}`, nil); !strings.Contains(failure, "99999") {
		t.Errorf("failure %q, want the offset named", failure)
	}
	if failure := l.call(t, "library_read", `{"path":"library/Missing.pdf"}`, nil); !strings.Contains(failure, "Missing.pdf") {
		t.Errorf("failure %q, want the path named", failure)
	}
	if failure := l.call(t, "library_read", `{"path":" "}`, nil); !strings.Contains(failure, "path") {
		t.Errorf("failure %q", failure)
	}
}

func TestFileReadRejectsABinaryAndNamesLibraryRead(t *testing.T) {
	l := newLibrary(t)
	for _, path := range []string{guidePath, talkPath} {
		failure := l.call(t, "file_read", `{"path":"`+path+`"}`, nil)
		if !strings.Contains(failure, "library_read") || !strings.Contains(failure, path) {
			t.Errorf("file_read of %s rejected with %q, want the path and library_read named", path, failure)
		}
	}
	var md fileReadAnswer
	if failure := l.call(t, "file_read", `{"path":"Entropy.md"}`, &md); failure != "" || !strings.Contains(md.Text, "Disorder grows.") {
		t.Errorf("a markdown file gave %q and %+v", failure, md)
	}
	var txt fileReadAnswer
	if failure := l.call(t, "file_read", `{"path":"data.txt","start":7,"length":6}`, &txt); failure != "" || txt.Text != "line 2" {
		t.Errorf("a byte range of a text file gave %q and %q", failure, txt.Text)
	}
}

func TestTheDescriptionsSayWhichToolReadsWhat(t *testing.T) {
	l := newLibrary(t)
	infos, err := l.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*tools.ToolInfo{}
	for _, info := range infos.Tools {
		byName[info.GetName()] = info
	}
	for name, mentions := range map[string][]string{
		"note_search":    {"library_search"},
		"library_search": {"note_search", "PDF", "transcript"},
		"file_read":      {"library_read"},
		"library_read":   {"file_read", "text version"},
	} {
		info := byName[name]
		if info == nil {
			t.Fatalf("%s is not listed", name)
		}
		for _, want := range mentions {
			if !strings.Contains(info.GetDescription(), want) {
				t.Errorf("the description of %s is %q, want %q in it", name, info.GetDescription(), want)
			}
		}
	}
	for _, name := range []string{"library_search", "library_read"} {
		servers := byName[name].GetServers()
		if !slices.Contains(servers, tools.Full) || !slices.Contains(servers, tools.Reading) || len(servers) != 2 {
			t.Errorf("%s is offered on %v, want the full and the reading server", name, servers)
		}
	}
}

func TestLibraryReadRejectsAPageWithAStartAndANegativeNumber(t *testing.T) {
	l := newLibrary(t)
	if failure := l.call(t, "library_read", `{"path":"`+guidePath+`","page":2,"start":5}`, nil); !strings.Contains(failure, "page") || !strings.Contains(failure, "start") {
		t.Errorf("a page with a start gave %q, want both named", failure)
	}
	for _, field := range []string{"page", "start", "length"} {
		failure := l.call(t, "library_read", `{"path":"`+guidePath+`","`+field+`":-1}`, nil)
		if !strings.Contains(failure, field) || !strings.Contains(failure, "-1") {
			t.Errorf("a negative %s gave %q, want the field and the number named", field, failure)
		}
	}
}

func TestARangeOverTheBinaryTailOfATextFileIsNotText(t *testing.T) {
	l := newLibrary(t)
	text := strings.Repeat("plain text line\n", 1000)
	l.vault.File("export.dat", text+"\xff\xfe binary tail")
	var head fileReadAnswer
	if failure := l.call(t, "file_read", `{"path":"export.dat"}`, &head); failure != "" || head.Text != text {
		t.Errorf("file_read of the text before the binary tail gave %q and %d bytes, want the 16000 bytes of text", failure, len(head.Text))
	}
	if failure := l.call(t, "file_read", `{"path":"export.dat","start":16000}`, nil); !strings.Contains(failure, "library_read") {
		t.Errorf("file_read of the binary tail gave %q, want an error", failure)
	}
	if failure := l.call(t, "library_read", `{"path":"export.dat","start":16000}`, nil); !strings.Contains(failure, "no text version") {
		t.Errorf("library_read of the binary tail gave %q, want no text version", failure)
	}
}

func TestLibrarySearchFailsWhenTheTitlesOfTheNotesCannotBeRead(t *testing.T) {
	l := newLibrary(t)
	l.vault.Hide()
	failure := l.call(t, "library_search", `{"query":"heron"}`, nil)
	if !strings.Contains(failure, host.ErrNoVault.Error()) {
		t.Errorf("library_search with the notes unreadable gave %q, want a failure containing %q", failure, host.ErrNoVault.Error())
	}
}

func TestLibraryReadFailsOnASegmentWithInvalidSelector(t *testing.T) {
	l := newLibrary(t)
	const pondPath = "library/Pond Survey.pdf"
	file := []byte(pdfBytes + "pond")
	l.vault.File(pondPath, string(file))
	l.texts.File(pondPath, file)
	putText(t, l.texts, readerID, pondPath, "page-text", file, []domain.TextSegment{
		{Text: "Frogs lay eggs in spring. ", Selector: `{"page":1}`},
		{Text: "Newts hide under stones. ", Selector: `{"page":"two"}`},
	})
	failure := l.call(t, "library_read", `{"path":"`+pondPath+`"}`, nil)
	if failure == "" || !strings.Contains(failure, pondPath) {
		t.Errorf("library_read of a text with a selector that does not decode gave %q, want a failure that names %s", failure, pondPath)
	}
}

func TestLibraryReadLengthStatesThePlainTextFileLimit(t *testing.T) {
	l := newLibrary(t)
	length, ok := getSchemaProperties(t, getToolSchema(t, l.registry, "library_read", false))["length"].(map[string]any)
	if !ok {
		t.Fatal("library_read has no length property")
	}
	description, ok := length["description"].(string)
	if !ok {
		t.Fatalf("the length of library_read has no description: %v", length)
	}
	for _, want := range []string{"20000", "50000", "16000"} {
		if !strings.Contains(description, want) {
			t.Errorf("the length of library_read does not state %s: %q", want, description)
		}
	}
}
