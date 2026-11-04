package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

const pluginID = "md.numen.readers.epub"

// newTextVault is a vault holding one file under name, the texts of that vault with the plugin
// providing, and the plugin's text server.
func newTextVault(t *testing.T, name string, raw []byte) (*hosttest.Texts, *TextServer) {
	t.Helper()
	files := hosttest.New("vault")
	files.File(name, string(raw))
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	server := NewTextServer(files, texts, texts.GetStore(pluginID), hosttest.NewQueue())
	t.Cleanup(server.Close)
	return texts, server
}

// createText starts the text of a book and waits for it to be written.
func createText(t *testing.T, server *TextServer, req host.TextCreation) {
	t.Helper()
	if err := server.CreateText(t.Context(), req); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	server.Wait()
}

// readLive is the live version of a file and its whole text.
func readLive(t *testing.T, texts *hosttest.Texts, path string) domain.VersionText {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range listing.Versions {
		if v.GetID() == listing.Live {
			text, err := host.ReadWholeText(t.Context(), texts, v)
			if err != nil {
				t.Fatal(err)
			}
			return text
		}
	}
	t.Fatalf("%s has no live version among %v", path, listing.Versions)
	return domain.VersionText{}
}

func TestTheTextOfABookIsItsLiveVersion(t *testing.T) {
	raw := createImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)
	book, err := epub.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	createText(t, server, host.TextCreation{Path: "book.epub"})

	text := readLive(t, texts, "book.epub")
	if text.Version().Kind != textKind || !text.Version().IsFromFile() {
		t.Errorf("version %+v, want a book text made from the file", text.Version())
	}
	if text.Version().SourceHash != domain.HashContent(raw) || text.Version().File != domain.HashContent(raw)+".jsonl" {
		t.Errorf("version %+v does not name the file and the text as they are", text.Version())
	}
	if text.Text != book.Text {
		t.Errorf("text %q, want the text the reader sets the book with, %q", text.Text, book.Text)
	}
	sections := text.GetHeadings()
	if len(sections) != len(book.Parts) {
		t.Fatalf("sections %v, want the %d parts the book names", sections, len(book.Parts))
	}
	for i, part := range book.Parts {
		if sections[i].Title != part.Title || sections[i].Offset != part.Offset {
			t.Errorf("section %d is %q at %d, want %q at %d", i, sections[i].Title, sections[i].Offset, part.Title, part.Offset)
		}
	}
	if got := text.Locate(strings.Index(text.Text, "Goodbye")); got != "Second part" {
		t.Errorf("Goodbye is located at %q, want Second part", got)
	}
}

func TestAFileWithNoReadableBookGetsNoVersion(t *testing.T) {
	for name, raw := range map[string][]byte{
		"corrupted.epub":   []byte("INVALID_EPUB_ZIP"),
		"locked.epub":      createEncryptedEpub(t),
		"nocontainer.epub": createZipWithoutContainer(t),
		"nopackage.epub":   createZipEpub(t, nil),
		"emptytext.epub": createZipEpub(t, map[string][]byte{
			"EPUB/package.opf": []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata/>
<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="c1"/></spine></package>`),
			"EPUB/c1.xhtml": []byte(`<html><body>   </body></html>`),
		}),
	} {
		t.Run(name, func(t *testing.T) {
			texts, server := newTextVault(t, name, raw)

			createText(t, server, host.TextCreation{Path: name})

			listing, err := texts.ListVersions(t.Context(), name)
			if err != nil {
				t.Fatal(err)
			}
			if len(listing.Versions) != 0 {
				t.Errorf("versions %v, want none", listing.Versions)
			}
		})
	}
}

func createZipWithoutContainer(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("mimetype")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("application/epub+zip")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLongTextIsSplitIntoPartsOnWholeCharacters(t *testing.T) {
	text := strings.Repeat("ж", partBytes) + "end"
	sections := []epub.Part{{Title: "One", Offset: 0}, {Title: "Inner", Offset: 0}, {Title: "Two", Offset: 2 * partBytes}}

	parts, err := getSegments(text, sections, epub.Places{})
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	var named []string
	for _, part := range parts {
		if len(part.Text) > partBytes {
			t.Errorf("a part of %d bytes, want at most %d", len(part.Text), partBytes)
		}
		if part.Text != "" && !utf8.RuneStart(part.Text[0]) {
			t.Errorf("a part begins inside a character")
		}
		if part.Heading != "" {
			named = append(named, part.Heading)
		}
		joined.WriteString(part.Text)
	}
	if joined.String() != text {
		t.Error("the parts do not join into the text")
	}
	if strings.Join(named, ",") != "One,Inner,Two" {
		t.Errorf("sections %v, want One, Inner and Two", named)
	}
	version := domain.NewVersionText(domain.TextVersion{Kind: textKind}, parts)
	if got := version.GetHeadingsAt(0); strings.Join(got, ",") != "One,Inner" {
		t.Errorf("names at 0 are %v, want One and Inner", got)
	}
	if got := version.GetHeadings(); got[2].Offset != 2*partBytes {
		t.Errorf("Two begins at %d, want %d", got[2].Offset, 2*partBytes)
	}
}

func TestTheTextOfABookIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	raw := createImageEpub(t)
	physics := hosttest.New("Physics")
	chemistry := hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.File("book.epub", string(raw))
	texts := hosttest.NewTexts("Physics", pluginID)
	other, isTexts := texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of the vault are not hosttest texts")
	}
	other.File("book.epub", raw)
	other.SetStore(pluginID, chemistry.Store())
	server := NewTextServer(physics, texts, physics.Store(), hosttest.NewQueue())
	t.Cleanup(server.Close)

	createText(t, server, host.TextCreation{Vault: "Chemistry", Path: "book.epub"})

	if text := readLive(t, other, "book.epub"); text.Version().Kind != textKind {
		t.Errorf("Chemistry's live version is %+v", text.Version())
	}
	inShown, err := texts.ListVersions(t.Context(), "book.epub")
	if err != nil {
		t.Fatal(err)
	}
	if len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
}

func TestEachSegmentIsSelectedByTheCFIOfItsFirstCharacter(t *testing.T) {
	book, err := epub.Read(createGuidedEpub(t))
	if err != nil {
		t.Fatal(err)
	}

	segments, err := getSegments(book.Text, book.Parts, book.ReadPlaces())
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, segment := range segments {
		selector, err := readCFISelector(segment.Selector)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, segment.Heading+"|"+firstWord(segment.Text)+"|"+selector.CFI)
	}
	want := []string{
		"|Prologue|epubcfi(/8/2!/4/2/1:0)",
		"|word|epubcfi(/8/2!/4/4/1:3)",
		"|ord|epubcfi(/8/2!/4/4/1:65539)",
		"|Epilogue|epubcfi(/8/4!/4/2/1:0)",
	}
	if strings.Join(got, ";") != strings.Join(want, ";") {
		t.Errorf("segments %q, want %q", got, want)
	}
}

// firstWord is the first word of a text, and empty for one with none.
func firstWord(text string) string {
	if words := strings.Fields(text); len(words) > 0 {
		return words[0]
	}
	return ""
}

func TestTheTextOfABookCarriesTheCFIOfItsCharacters(t *testing.T) {
	raw := createImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)

	createText(t, server, host.TextCreation{Path: "book.epub"})

	text := readLive(t, texts, "book.epub")
	selector, err := readCFISelector(text.GetSelector(strings.Index(text.Text, "Goodbye")))
	if err != nil || selector.CFI != "epubcfi(/6/4!/4/4/1:0)" {
		t.Errorf("the selector of Goodbye is %+v (%v), want epubcfi(/6/4!/4/4/1:0)", selector, err)
	}
}

type failingQueue struct {
	*hosttest.Queue
	enqueueErr error
	reportErr  error
}

func (f *failingQueue) EnqueueWork(ctx context.Context, w host.Work) (host.Turn, error) {
	if f.enqueueErr != nil {
		return nil, f.enqueueErr
	}
	return f.Queue.EnqueueWork(ctx, w)
}

func (f *failingQueue) ReportTask(ctx context.Context, task host.Task) error {
	if f.reportErr != nil && task.FailureReason == "" {
		return f.reportErr
	}
	return f.Queue.ReportTask(ctx, task)
}

type failingStore struct {
	host.Store
	err error
}

func (f *failingStore) Write(ctx context.Context, path string, data []byte) error {
	return f.err
}

type failingTexts struct {
	host.Texts
	err error
}

func (f *failingTexts) PutVersion(ctx context.Context, req host.TextPut) (domain.TextVersion, error) {
	return domain.TextVersion{}, f.err
}

func TestTextServerLifecycleAndFailures(t *testing.T) {
	raw := createImageEpub(t)
	files := hosttest.New("vault")
	files.File("book.epub", string(raw))
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("book.epub", raw)
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	srv := NewTextServer(files, texts, texts.GetStore(pluginID), queue)

	if err := srv.CreateText(t.Context(), host.TextCreation{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}
	// Duplicate call while running is a no-op
	if err := srv.CreateText(t.Context(), host.TextCreation{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}
	srv.Close()
	_ = held.Release()
	// Call after Close is a no-op
	if err := srv.CreateText(t.Context(), host.TextCreation{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}

	// Failure reporting when file does not exist
	q2 := hosttest.NewQueue()
	srv2 := NewTextServer(files, texts, texts.GetStore(pluginID), q2)
	t.Cleanup(srv2.Close)
	createText(t, srv2, host.TextCreation{Path: "missing.epub"})
	tasks := q2.ListTasks()
	if len(tasks) != 1 || tasks[0].FailureReason == "" {
		t.Errorf("expected failed task for missing file, got %+v", tasks)
	}

	// EnqueueWork failure
	q3 := &failingQueue{Queue: hosttest.NewQueue(), enqueueErr: errors.New("queue down")}
	srv3 := NewTextServer(files, texts, texts.GetStore(pluginID), q3)
	t.Cleanup(srv3.Close)
	createText(t, srv3, host.TextCreation{Path: "book.epub"})
	if tasks := q3.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "queue down") {
		t.Errorf("expected queue down task failure, got %+v", tasks)
	}

	// ReportTask failure
	q4 := &failingQueue{Queue: hosttest.NewQueue(), reportErr: errors.New("report failed")}
	srv4 := NewTextServer(files, texts, texts.GetStore(pluginID), q4)
	t.Cleanup(srv4.Close)
	createText(t, srv4, host.TextCreation{Path: "book.epub"})
	if tasks := q4.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "report failed") {
		t.Errorf("expected report failed task failure, got %+v", tasks)
	}

	// Store Write failure
	q5 := hosttest.NewQueue()
	srv5 := NewTextServer(files, texts, &failingStore{Store: texts.GetStore(pluginID), err: errors.New("store full")}, q5)
	t.Cleanup(srv5.Close)
	createText(t, srv5, host.TextCreation{Path: "book.epub"})
	if tasks := q5.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "store full") {
		t.Errorf("expected store full task failure, got %+v", tasks)
	}

	// PutVersion failure
	q6 := hosttest.NewQueue()
	srv6 := NewTextServer(files, &failingTexts{Texts: texts, err: errors.New("put version failed")}, texts.GetStore(pluginID), q6)
	t.Cleanup(srv6.Close)
	createText(t, srv6, host.TextCreation{Path: "book.epub"})
	if tasks := q6.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "put version failed") {
		t.Errorf("expected put version failed task failure, got %+v", tasks)
	}

	// Sequential CreateText after completion works (verifies delete(s.running, key))
	q7 := hosttest.NewQueue()
	srv7 := NewTextServer(files, texts, texts.GetStore(pluginID), q7)
	t.Cleanup(srv7.Close)
	createText(t, srv7, host.TextCreation{Path: "book.epub"})
	files.File("book.epub", string(createGuidedEpub(t)))
	texts.File("book.epub", createGuidedEpub(t))
	createText(t, srv7, host.TextCreation{Path: "book.epub"})
	if live := readLive(t, texts, "book.epub"); !strings.Contains(live.Text, "Prologue") {
		t.Errorf("second CreateText did not update live version, got %q", live.Text)
	}
}
