package textversion_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

const (
	pdfReader   = "org.numen.readers-pdf"
	ocr         = "org.numen.engines-ocr"
	proofreader = "org.numen.tools-proofreading"
	player      = "org.numen.views-media-player"
	scanPath    = "books/scan.pdf"

	kindPage        domain.TextKind = "page-text"
	kindRecognition domain.TextKind = "recognition"
)

// scene is one vault holding one PDF, and the three plugins of the worked example:
// a reader of the PDF's own text layer that runs on its own and declares no menu
// entry, a recogniser, and a proofreader of both kinds of text.
type scene struct {
	vault    domain.Vault
	versions *textversion.Versions
	plugins  *plugins
	listener *listener
	reader   *fakeProvider
	ocr      *fakeProvider
	fixer    *fakeProvider
	files    map[string][]byte
	folders  *memoryFiles
	repo     *memoryVersions
	made     providers
	byID     map[string]*fakeProvider
}

func newScene(t *testing.T) *scene {
	t.Helper()
	s := &scene{
		vault:    domain.Vault{ID: "v1", Name: "v1"},
		listener: &listener{},
		files:    map[string][]byte{scanPath: []byte("%PDF scan, first printing")},
		folders:  &memoryFiles{},
		repo:     &memoryVersions{},
		plugins: &plugins{all: []textport.TextPlugin{
			{ID: ocr, Name: "OCR"},
			{ID: pdfReader, Name: "PDF", Files: domain.FileInput{".pdf"}},
			{ID: proofreader, Name: "Proofreading"},
		}},
	}
	page := func(word string, n int) domain.TextSegment {
		return domain.TextSegment{Text: word + " ", Selector: domain.Selector(`{"page":` + string(rune('1'+n)) + `}`)}
	}
	s.reader = &fakeProvider{pluginID: pdfReader, kind: kindPage, files: s.files,
		make: func(string, domain.VersionText, bool) []domain.TextSegment {
			return []domain.TextSegment{page("layer", 0), page("teh", 1)}
		}}
	s.ocr = &fakeProvider{pluginID: ocr, kind: kindRecognition, files: s.files,
		make: func(string, domain.VersionText, bool) []domain.TextSegment {
			return []domain.TextSegment{page("scanned", 0), page("teh", 1)}
		}}
	s.fixer = &fakeProvider{pluginID: proofreader, kind: kindRecognition, files: s.files,
		make: func(_ string, source domain.VersionText, _ bool) []domain.TextSegment {
			return []domain.TextSegment{{Text: strings.ReplaceAll(source.Text, "teh", "the")}}
		}}
	s.made = providers{pdfReader: s.reader, ocr: s.ocr, proofreader: s.fixer}
	s.byID = map[string]*fakeProvider{pdfReader: s.reader, ocr: s.ocr, proofreader: s.fixer}
	versions, err := textversion.NewVersions(s.repo, s.folders, s.plugins, s.made, fileReaders(s.files), steppingClock())
	if err != nil {
		t.Fatal(err)
	}
	s.versions = versions
	versions.SetListener(s.listener)
	for _, p := range []*fakeProvider{s.reader, s.ocr, s.fixer} {
		p.versions, p.vault, p.folders = versions, s.vault, s.folders
	}
	return s
}

func (s *scene) fileHash() string { return domain.HashContent(s.files[scanPath]) }

// getLive is the live version of the scan and its text.
func (s *scene) getLive(t *testing.T) (domain.TextVersionID, string) {
	t.Helper()
	listing, err := s.versions.ListVersions(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	if listing.Live == "" {
		return "", ""
	}
	text, err := s.versions.GetLiveText(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatalf("read the live text: %v", err)
	}
	return listing.Live, text.Text
}

// requireLive asserts which version is live and what its text is.
func (s *scene) requireLive(t *testing.T, step string, wantID domain.TextVersionID, wantText string) {
	t.Helper()
	id, text := s.getLive(t)
	if id != wantID || text != wantText {
		t.Fatalf("%s: the live version is %q with %q, want %q with %q", step, id, text, wantID, wantText)
	}
}

// runEntry runs the command of the plugin: over the live version where it reads versions, and
// over the file for any other plugin.
func (s *scene) runEntry(t *testing.T, pluginID string) {
	t.Helper()
	var source domain.TextVersionID
	if pluginID == proofreader {
		source, _ = s.getLive(t)
	}
	if err := s.byID[pluginID].create(t.Context(), scanPath, source); err != nil {
		t.Fatalf("the plugin %s: %v", pluginID, err)
	}
}

// requestText requests text for the scan from the standalone plugins.
func (s *scene) requestText(t *testing.T) {
	t.Helper()
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
}

func (s *scene) getReaderVersion() domain.TextVersionID {
	return domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8])
}

func getProofreadingOf(source domain.TextVersionID) domain.TextVersionID {
	return domain.NewTextVersionID(proofreader, "from-"+strings.ReplaceAll(string(source), "/", "-"))
}

func TestThePDFExampleShowsTheLiveVersionAtEveryStep(t *testing.T) {
	s := newScene(t)
	a := s.getReaderVersion()
	b := getProofreadingOf(a)
	c := domain.NewTextVersionID(ocr, "file-"+s.fileHash()[:8])
	d := getProofreadingOf(c)

	// A: the file has no version, so the standalone reader is queried for one.
	s.requestText(t)
	s.requireLive(t, "A from the file", a, "layer teh ")
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("putting A reported %v, want one change of %s", got, scanPath)
	}

	// B: proofread A.
	s.runEntry(t, proofreader)
	s.requireLive(t, "B from A", b, "layer the ")

	// C: recognise the file.
	s.runEntry(t, ocr)
	s.requireLive(t, "C from the file", c, "scanned teh ")

	// D: proofread C, which is what the file shows now.
	s.runEntry(t, proofreader)
	s.requireLive(t, "D from C", d, "scanned the ")
}

func TestADisabledPluginsVersionStaysLiveAndReadable(t *testing.T) {
	s := newScene(t)
	s.runEntry(t, ocr)
	c := domain.NewTextVersionID(ocr, "file-"+s.fileHash()[:8])
	s.plugins.setEnabled(ocr, false)

	s.requireLive(t, "OCR disabled", c, "scanned teh ")
	text, err := s.versions.GetVersionText(t.Context(), s.vault, scanPath, c)
	if err != nil || text.Text != "scanned teh " {
		t.Fatalf("the version of a disabled plugin reads %q (%v)", text.Text, err)
	}
	if got := s.reader.getCreated(); got != 0 {
		t.Fatalf("a file with a disabled plugin's live version asked the reader %d times", got)
	}
	s.requestText(t)
	if got := s.reader.getCreated(); got != 0 {
		t.Fatalf("RequestText asked the reader %d times for a file that has a live version", got)
	}
}

func TestAVersionWhoseFileChangedIsNotLive(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.runEntry(t, proofreader)
	b := getProofreadingOf(s.getReaderVersion())
	s.folders.write(proofreader+"/"+strings.TrimPrefix(string(b), proofreader+"/")+".jsonl", []byte(`{"text":"tampered"}`+"\n"))

	s.requireLive(t, "after B's file changed", s.getReaderVersion(), "layer teh ")
	if _, err := s.versions.GetVersionText(t.Context(), s.vault, scanPath, b); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("reading a changed version file: got %v, want ErrNoText", err)
	}
}

func TestCreateEditMakesANewLiveVersionInTheEditingPluginsFolder(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	a := s.getReaderVersion()
	s.listener.take()

	edit, err := s.versions.CreateEdit(t.Context(), s.vault, textversion.TextEdit{
		PluginID: player, Path: scanPath, Source: a,
		Segments: []domain.TextSegment{{Text: "layer the ", Selector: `{"page":1}`}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if edit.PluginID != player || edit.Source != a || edit.Kind != kindPage || !s.folders.has(player+"/"+edit.File) {
		t.Fatalf("the edit is %+v, want a %s version of %s in the player's folder", edit, kindPage, a)
	}
	s.requireLive(t, "after the edit", edit.GetID(), "layer the ")
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("the edit reported %v, want one change of %s", got, scanPath)
	}

	again, err := s.versions.CreateEdit(t.Context(), s.vault, textversion.TextEdit{
		PluginID: player, Path: scanPath, Source: edit.GetID(), Segments: []domain.TextSegment{{Text: "the layer"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.GetID() == edit.GetID() || again.Source != edit.GetID() {
		t.Fatalf("a second edit is %+v, want a new version made from %s", again, edit.GetID())
	}
	s.requireLive(t, "after the second edit", again.GetID(), "the layer")
}

func TestCreateEditRejectsASourceTheFileDoesNotHave(t *testing.T) {
	s := newScene(t)
	_, err := s.versions.CreateEdit(t.Context(), s.vault, textversion.TextEdit{PluginID: player, Path: scanPath, Source: "x/y"})
	if !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("got %v, want ErrNoText", err)
	}
}

func TestReplacingTheFileMakesItsVersionsStale(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.runEntry(t, proofreader)
	first := s.fileHash()

	s.files[scanPath] = []byte("%PDF scan, second printing")
	if id, _ := s.getLive(t); id != "" {
		t.Fatalf("%s is live after the file was replaced", id)
	}
	if _, err := s.versions.GetLiveText(t.Context(), s.vault, scanPath, s.fileHash()); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("the live text of a replaced file: got %v, want ErrNoText", err)
	}

	s.requestText(t)
	if got := s.reader.getCreated(); got != 2 {
		t.Fatalf("the reader made %d texts, want 2", got)
	}
	s.requireLive(t, "after the replacement", s.getReaderVersion(), "layer teh ")

	listing, err := s.versions.ListVersions(t.Context(), s.vault, scanPath, first)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(listing.Live), proofreader+"/") {
		t.Fatalf("for the first bytes the live version is %q, want the proofreading", listing.Live)
	}
}

func TestRequestTextAsksOncePerContentAndNotForALiveFile(t *testing.T) {
	s := newScene(t)
	for range 3 {
		s.requestText(t)
	}
	if got := s.reader.getCreated(); got != 1 {
		t.Fatalf("the reader was asked %d times, want 1", got)
	}
	if got := s.ocr.getCreated(); got != 0 {
		t.Fatalf("the recogniser, which does not run on its own, was asked %d times", got)
	}
}

func TestDeletePluginVersionsRemovesTheRecordsAndReportsTheFiles(t *testing.T) {
	s := newScene(t)
	s.runEntry(t, ocr)
	s.listener.take()

	if err := s.versions.DeletePluginVersions(t.Context(), s.vault, ocr); err != nil {
		t.Fatal(err)
	}
	listing, err := s.versions.ListVersions(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 || listing.Live != "" {
		t.Fatalf("after the plugin's data was deleted the scan has %+v", listing)
	}
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("deleting the plugin's versions reported %v, want one change of %s", got, scanPath)
	}
}

func TestRecordVersionRejectsInvalidVersions(t *testing.T) {
	s := newScene(t)
	ctx := t.Context()
	if _, err := s.folders.WriteTextFile(ctx, ocr, "k.jsonl", []domain.TextSegment{{Text: "x"}}); err != nil {
		t.Fatal(err)
	}
	s.folders.write(ocr+"/vtt.jsonl", []byte("WEBVTT\n"))
	valid := textversion.VersionPut{Path: scanPath, Key: "k", Kind: kindRecognition, File: "k.jsonl"}
	with := func(change func(*textversion.VersionPut)) textversion.VersionPut {
		params := valid
		change(&params)
		return params
	}
	cases := map[string]struct {
		pluginID string
		params   textversion.VersionPut
		want     error
	}{
		"a plugin that makes no text": {"org.numen.other", valid, textport.ErrNotTextPlugin},
		"no key":                      {ocr, with(func(p *textversion.VersionPut) { p.Key = "" }), textport.ErrInvalidVersion},
		"no kind":                     {ocr, with(func(p *textversion.VersionPut) { p.Kind = "" }), textport.ErrInvalidVersion},
		"no file":                     {ocr, with(func(p *textversion.VersionPut) { p.File = "" }), textport.ErrInvalidVersion},
		"a file that is not there":    {ocr, with(func(p *textversion.VersionPut) { p.File = "gone.jsonl" }), textport.ErrInvalidVersion},
		"a file that is not segments": {ocr, with(func(p *textversion.VersionPut) { p.File = "vtt.jsonl" }), textport.ErrInvalidVersion},
		"a source that is not there":  {ocr, with(func(p *textversion.VersionPut) { p.Source = "x/y" }), textport.ErrInvalidVersion},
		"itself as its source":        {ocr, with(func(p *textversion.VersionPut) { p.Source = domain.NewTextVersionID(ocr, "k") }), textport.ErrInvalidVersion},
	}
	for name, c := range cases {
		if _, err := s.versions.PutVersion(ctx, s.vault, c.pluginID, c.params); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
	if got := s.listener.take(); len(got) != 0 {
		t.Fatalf("a refused put reported %v", got)
	}
}

func TestRecordVersionStoresTheFileHashAndCreateTime(t *testing.T) {
	s := newScene(t)
	ctx := t.Context()
	params := textversion.VersionPut{Path: scanPath, Key: "k", Kind: kindRecognition, SourceHash: s.fileHash(), File: "k.jsonl"}
	firstHash, err := s.folders.WriteTextFile(ctx, ocr, params.File, []domain.TextSegment{{Text: "one"}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.versions.PutVersion(ctx, s.vault, ocr, params)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := s.folders.WriteTextFile(ctx, ocr, params.File, []domain.TextSegment{{Text: "one"}, {Text: "two"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.versions.PutVersion(ctx, s.vault, ocr, params)
	if err != nil {
		t.Fatal(err)
	}
	if first.Hash != firstHash || second.Hash != secondHash || !second.CreateTime.Equal(first.CreateTime) {
		t.Fatalf("versions %+v then %+v, want the file's hash each time and the first time", first, second)
	}
}

func TestGetTextPageReadsTheVersionFileInPages(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	page, err := s.versions.GetTextPage(t.Context(), s.vault, scanPath, s.getReaderVersion(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Segments) != 2 || page.NextCursor != "" || page.Segments[1].Selector != `{"page":2}` {
		t.Fatalf("page %+v, want both segments with their selectors and no next page", page)
	}
	if _, err := s.versions.GetTextPage(t.Context(), s.vault, scanPath, s.getReaderVersion(), "x"); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("a cursor that is no number: got %v, want ErrNoText", err)
	}
}
