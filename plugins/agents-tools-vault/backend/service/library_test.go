package service_test

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// The invented library: a PDF of a bird guide with a text layer of three pages, a scan with no text
// version, an audio recording with a transcript, and two notes.
const (
	guidePath  = "library/Bird Guide.pdf"
	scanPath   = "library/Old Scan.pdf"
	talkPath   = "recordings/Marsh Talk.mp3"
	pdfBytes   = "%PDF-1.7\n\x00\x01binary body"
	readerID   = "org.example.reader"
	scribeID   = "org.example.scribe"
	pageOne    = "Chapter one. The marsh at dawn is quiet. "
	pageTwo    = "The grey heron stands in the reeds and waits for fish. "
	pageThree  = "Herons nest in tall trees near the water. "
	talkSecond = "Welcome to the marsh talk. "
	talkThird  = "Today we speak about the heron. "
)

type library struct {
	vault  *hosttest.Vault
	texts  *hosttest.Texts
	search *hosttest.Search
	calls  *tools.Registry
}

func newLibrary(t *testing.T) library {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File("Entropy.md", "---\ntitle: Entropy Concept\n---\n# Entropy\nDisorder grows.")
	vault.File("notes/Herons.md", "---\ntitle: Herons of the Marsh\n---\nHerons fish at dusk.")
	vault.File(guidePath, pdfBytes)
	vault.File(scanPath, pdfBytes+"scan")
	vault.File(talkPath, "ID3\x00\x03audio")
	vault.File("data.txt", "line 1\nline 2\nline 3\n")

	texts := hosttest.NewTexts(testVault, readerID)
	texts.File(guidePath, []byte(pdfBytes))
	texts.File(talkPath, []byte("ID3\x00\x03audio"))
	putText(t, texts, readerID, guidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: pageOne, Selector: `{"page":1,"boxes":[[0,0,1,1]]}`},
		{Text: pageTwo[:20], Selector: `{"page":2}`},
		{Text: pageTwo[20:], Selector: `{"page":2}`},
		{Text: pageThree, Selector: `{"page":3}`},
	})
	putText(t, texts.As(scribeID), scribeID, talkPath, "transcript", []byte("ID3\x00\x03audio"), []domain.TextSegment{
		{Text: talkSecond, Selector: `{"t":[0,4.5]}`},
		{Text: talkThird, Selector: `{"t":[4.5,9]}`},
	})

	search := hosttest.NewSearch(testVault)
	search.SetIndexedPassages([]host.Passage{
		{Path: guidePath, Text: "The grey heron stands in the reeds", Location: "page 2", Start: 41, Length: 56, Selector: `{"page":2}`},
		{Path: "notes/Herons.md", Text: "Herons fish at dusk.", Start: 0, Length: 20, Line: 1},
		{Path: talkPath, Text: "Today we speak about the heron.", Location: "00:04", Start: 27, Length: 32, Selector: `{"t":[4.5,9]}`},
	})

	svc := service.New(vault).WithLibrary(texts, search)
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	return library{vault: vault, texts: texts, search: search, calls: registry}
}

func putText(t *testing.T, texts *hosttest.Texts, pluginID, path, kind string, file []byte, segments []domain.TextSegment) {
	t.Helper()
	name := kind + ".jsonl"
	if err := host.WriteText(t.Context(), texts.GetStore(pluginID), name, segments); err != nil {
		t.Fatal(err)
	}
	if _, err := texts.PutVersion(t.Context(), host.TextPut{
		Path: path, Key: kind, Kind: domain.TextKind(kind), SourceHash: domain.HashContent(file), File: name,
	}); err != nil {
		t.Fatal(err)
	}
}

// call runs a tool on the vault and returns the structured answer, or the failure text.
func (l library) call(t *testing.T, name, arguments string, out any) (failure string) {
	t.Helper()
	res, err := l.calls.InvokeTool(context.Background(), tools.Call{VaultID: testVault, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsRefusal() {
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
	var out service.LibrarySearchOutput
	if failure := l.call(t, "library_search", `{"query":"heron"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(out.Matches) != 3 || out.Query != "heron" {
		t.Fatalf("matches %+v", out.Matches)
	}
	pdf := out.Matches[0]
	if pdf.Source != guidePath || pdf.Title != "Bird Guide" || pdf.Location != "page 2" || pdf.Text != "The grey heron stands in the reeds" ||
		pdf.Start != 41 || pdf.Length != 56 {
		t.Errorf("the PDF passage is %+v", pdf)
	}
	if want := "numen:" + url.PathEscape(guidePath) + "?start=41&length=56"; pdf.At != want {
		t.Errorf("the address is %q, want %q", pdf.At, want)
	}
	note := out.Matches[1]
	if note.Title != "Herons of the Marsh" || note.Source != "notes/Herons.md" || note.Location != "" {
		t.Errorf("the note passage is %+v, want the title of the note", note)
	}
	if talk := out.Matches[2]; talk.Source != talkPath || talk.Location != "00:04" || talk.Title != "Marsh Talk" {
		t.Errorf("the transcript passage is %+v", talk)
	}
}

func TestLibrarySearchLimitsItToAKindAndAPathPrefix(t *testing.T) {
	l := newLibrary(t)
	var files, notes, narrowed service.LibrarySearchOutput
	l.call(t, "library_search", `{"query":"heron","kinds":["file"]}`, &files)
	l.call(t, "library_search", `{"query":"heron","kinds":["note"]}`, &notes)
	l.call(t, "library_search", `{"query":"heron","path_prefix":"recordings/","limit":1}`, &narrowed)
	if got := sourcesOf(files); !slices.Equal(got, []string{guidePath, talkPath}) {
		t.Errorf("the files that answer are %v", got)
	}
	if got := sourcesOf(notes); !slices.Equal(got, []string{"notes/Herons.md"}) {
		t.Errorf("the notes that answer are %v", got)
	}
	if got := sourcesOf(narrowed); !slices.Equal(got, []string{talkPath}) {
		t.Errorf("one passage under recordings/ is %v", got)
	}
}

func sourcesOf(out service.LibrarySearchOutput) []string {
	var got []string
	for _, m := range out.Matches {
		got = append(got, m.Source)
	}
	return got
}

func TestLibrarySearchRejectsUnknownKindAndBlankQuery(t *testing.T) {
	l := newLibrary(t)
	failure := l.call(t, "library_search", `{"query":"heron","kinds":["book"]}`, nil)
	if !strings.Contains(failure, `"book"`) || !strings.Contains(failure, "note") || !strings.Contains(failure, "file") {
		t.Errorf("failure %q, want the kind named and the two kinds offered", failure)
	}
	var blank service.LibrarySearchOutput
	if failure := l.call(t, "library_search", `{"query":"   "}`, &blank); failure != "" || len(blank.Matches) != 0 {
		t.Errorf("a blank query gave %q and %v", failure, blank.Matches)
	}
}

func TestLibrarySearchWorksTheVaultTheCallNames(t *testing.T) {
	vault := hosttest.New(testVault)
	garden := hosttest.New("garden-vault")
	vault.Beside(garden)
	search := hosttest.NewSearch(testVault, "garden-vault")
	search.SetIndexedPassages([]host.Passage{{Path: "Physics.md", Text: "heron physics", Start: 0, Length: 13}})
	gardenSearch, isFake := search.With("garden-vault").(*hosttest.Search)
	if !isFake {
		t.Fatal("the search of another vault is not a fake")
	}
	gardenSearch.SetIndexedPassages([]host.Passage{{Path: "Beds.md", Text: "heron beds", Start: 0, Length: 10}})
	registry, err := service.New(vault).WithLibrary(hosttest.NewTexts(testVault, readerID), search).Registry()
	if err != nil {
		t.Fatal(err)
	}

	for vaultID, want := range map[string]string{testVault: "Physics.md", "garden-vault": "Beds.md"} {
		res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: "library_search", Arguments: []byte(`{"query":"heron"}`)})
		if err != nil {
			t.Fatal(err)
		}
		var out service.LibrarySearchOutput
		if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil || len(out.Matches) != 1 || out.Matches[0].Source != want {
			t.Errorf("the call naming %s answered %s (%v), want %s alone", vaultID, res.GetResult().GetStructured(), err, want)
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

func TestLibraryReadReturnsPageWithLocationsAndNext(t *testing.T) {
	l := newLibrary(t)
	var out service.LibraryReadOutput
	if failure := l.call(t, "library_read", `{"path":"`+guidePath+`","page":2}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != pageTwo || out.Start != len(pageOne) || out.Length != len(pageTwo) {
		t.Errorf("page 2 is %q at %d for %d", out.Text, out.Start, out.Length)
	}
	if out.Size != len(pageOne)+len(pageTwo)+len(pageThree) || out.Kind != "page-text" {
		t.Errorf("size %d kind %q", out.Size, out.Kind)
	}
	if len(out.Locations) != 1 || out.Locations[0].Page != 2 || out.Locations[0].Start != len(pageOne) || out.Locations[0].Length != len(pageTwo) {
		t.Errorf("locations %+v, want the one page, its two segments joined", out.Locations)
	}
	if out.Next == nil || *out.Next != len(pageOne)+len(pageTwo) {
		t.Fatalf("next %v, want the end of the page", out.Next)
	}

	var rest service.LibraryReadOutput
	l.call(t, "library_read", `{"path":"`+guidePath+`","start":`+itoa(*out.Next)+`}`, &rest)
	if rest.Text != pageThree || rest.Next != nil || len(rest.Locations) != 1 || rest.Locations[0].Page != 3 {
		t.Errorf("the continuation is %+v, want page 3 and no next", rest)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestLibraryReadStopsAtCharacterBoundary(t *testing.T) {
	l := newLibrary(t)
	var out service.LibraryReadOutput
	l.call(t, "library_read", `{"path":"`+guidePath+`","start":3,"length":10}`, &out)
	if out.Text != (pageOne + pageTwo + pageThree)[3:13] || out.Start != 3 || out.Length != 10 || out.Next == nil || *out.Next != 13 {
		t.Errorf("range %q at %d for %d next %v", out.Text, out.Start, out.Length, out.Next)
	}

	cyrillic := hosttest.NewTexts(testVault, readerID)
	cyrillic.File("ru.pdf", []byte(pdfBytes))
	putText(t, cyrillic, readerID, "ru.pdf", "page-text", []byte(pdfBytes), []domain.TextSegment{{Text: "Ачарьи говорят", Selector: `{"page":1}`}})
	l.vault.File("ru.pdf", pdfBytes)
	svc := service.New(l.vault).WithLibrary(cyrillic, l.search)
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_read", Arguments: []byte(`{"path":"ru.pdf","start":0,"length":3}`)})
	if err != nil {
		t.Fatal(err)
	}
	var slice service.LibraryReadOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &slice); err != nil {
		t.Fatal(err)
	}
	if slice.Text != "А" || slice.Length != 2 || slice.Next == nil || *slice.Next != 2 {
		t.Errorf("3 bytes of a two-byte letter text gave %q for %d, want the whole letter before the cut", slice.Text, slice.Length)
	}
}

func TestLibraryReadReturnsTranscriptTimes(t *testing.T) {
	l := newLibrary(t)
	var out service.LibraryReadOutput
	l.call(t, "library_read", `{"path":"`+talkPath+`"}`, &out)
	if out.Text != talkSecond+talkThird || out.Kind != "transcript" || len(out.Locations) != 2 {
		t.Fatalf("transcript %+v", out)
	}
	if first := out.Locations[0]; first.FromSeconds != 0 || first.ToSeconds != 4.5 || first.Start != 0 || first.Length != len(talkSecond) {
		t.Errorf("first location %+v", first)
	}
	if second := out.Locations[1]; second.FromSeconds != 4.5 || second.ToSeconds != 9 || second.Start != len(talkSecond) {
		t.Errorf("second location %+v", second)
	}
}

func TestLibraryReadReportsMissingTextVersion(t *testing.T) {
	l := newLibrary(t)
	failure := l.call(t, "library_read", `{"path":"`+scanPath+`"}`, nil)
	for _, want := range []string{scanPath, "no text version", "reader", "recognition", "transcription"} {
		if !strings.Contains(failure, want) {
			t.Errorf("failure %q does not say %q", failure, want)
		}
	}
}

func TestLibraryReadReadsATextFileThatNeedsNoTextVersion(t *testing.T) {
	l := newLibrary(t)
	var out service.LibraryReadOutput
	if failure := l.call(t, "library_read", `{"path":"data.txt"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != "line 1\nline 2\nline 3\n" || out.Size != 21 || out.Kind != "" || len(out.Locations) != 0 {
		t.Errorf("text file %+v", out)
	}
}

func TestLibraryReadRefusesAPageThatIsNotThereAndAFileThatIsNot(t *testing.T) {
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

func TestFileReadRefusesABinaryAndNamesLibraryRead(t *testing.T) {
	l := newLibrary(t)
	for _, path := range []string{guidePath, talkPath} {
		failure := l.call(t, "file_read", `{"path":"`+path+`"}`, nil)
		if !strings.Contains(failure, "library_read") || !strings.Contains(failure, path) {
			t.Errorf("file_read of %s refused with %q, want the path and library_read named", path, failure)
		}
	}
	var md service.FileReadOutput
	if failure := l.call(t, "file_read", `{"path":"Entropy.md"}`, &md); failure != "" || !strings.Contains(md.Content, "Disorder grows.") {
		t.Errorf("a markdown file gave %q and %+v", failure, md)
	}
	var txt service.FileReadOutput
	if failure := l.call(t, "file_read", `{"path":"data.txt","start":7,"length":6}`, &txt); failure != "" || txt.Content != "line 2" {
		t.Errorf("a byte range of a text file gave %q and %q", failure, txt.Content)
	}
}

func TestTheDescriptionsSayWhichToolReadsWhat(t *testing.T) {
	l := newLibrary(t)
	infos, err := l.calls.ListToolInfos(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*tools.ToolInfo{}
	for _, info := range infos {
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
				t.Errorf("%s says %q, want %q in it", name, info.GetDescription(), want)
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
