package texts

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/segments"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
)

const pluginID = "md.numen.readers.epub"

func filesOf(vault *hosttest.Vault) func(domain.VaultID) Vault {
	return func(id domain.VaultID) Vault { return vault.With(id) }
}

func textsOf(texts *hosttest.Texts) func(domain.VaultID) host.TextVersions {
	return func(id domain.VaultID) host.TextVersions { return texts.With(id) }
}

func storeOf(vault *hosttest.Vault) func(domain.VaultID) host.StoreWriter {
	return func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) }
}

func newTextVault(t *testing.T, name string, raw []byte) (*hosttest.Texts, *Writer) {
	t.Helper()
	files := hosttest.New("vault")
	files.File(name, string(raw))
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	texts.SetStore(pluginID, files.Store())
	server := NewWriter(filesOf(files), textsOf(texts), storeOf(files), hosttest.NewQueue(), noReport)
	t.Cleanup(server.Close)
	return texts, server
}

func createText(t *testing.T, server *Writer, req Request) {
	t.Helper()
	if err := server.CreateText(t.Context(), req); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	server.Wait()
}

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
	raw := testsupport.CreateImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)
	book, err := epub.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	createText(t, server, Request{Path: "book.epub"})

	text := readLive(t, texts, "book.epub")
	if text.Version().Kind != Kind || !text.Version().IsFromFile() {
		t.Errorf("version %+v, want a book text made from the file", text.Version())
	}
	if text.Version().InputHash != domain.HashContent(raw) || text.Version().File != domain.HashContent(raw)+".jsonl" {
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
		"locked.epub":      testsupport.CreateEncryptedEpub(t),
		"nocontainer.epub": testsupport.CreateZipWithoutContainer(t),
		"nopackage.epub":   testsupport.CreateZipEpub(t, nil),
		"emptytext.epub": testsupport.CreateZipEpub(t, map[string][]byte{
			"EPUB/package.opf": []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata/>
<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="c1"/></spine></package>`),
			"EPUB/c1.xhtml": []byte(`<html><body>   </body></html>`),
		}),
	} {
		t.Run(name, func(t *testing.T) {
			texts, server := newTextVault(t, name, raw)

			createText(t, server, Request{Path: name})

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

func TestTheTextOfABookIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	raw := testsupport.CreateImageEpub(t)
	physics := hosttest.New("Physics")
	chemistry := hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.File("book.epub", string(raw))
	texts := hosttest.NewTexts("Physics", pluginID)
	other := texts.With("Chemistry")
	other.File("book.epub", raw)
	other.SetStore(pluginID, chemistry.Store())
	server := NewWriter(filesOf(physics), textsOf(texts), storeOf(physics), hosttest.NewQueue(), noReport)
	t.Cleanup(server.Close)

	createText(t, server, Request{Vault: "Chemistry", Path: "book.epub"})

	if text := readLive(t, other, "book.epub"); text.Version().Kind != Kind {
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
	book, err := epub.Read(testsupport.CreateGuidedEpub(t))
	if err != nil {
		t.Fatal(err)
	}

	cut, err := segments.Cut(book.Text, book.Parts, book.ReadPlaces())
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, segment := range cut {
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

func firstWord(text string) string {
	if words := strings.Fields(text); len(words) > 0 {
		return words[0]
	}
	return ""
}

func TestTheTextOfABookCarriesTheCFIOfItsCharacters(t *testing.T) {
	raw := testsupport.CreateImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)

	createText(t, server, Request{Path: "book.epub"})

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
	host.StoreWriter
	err error
}

func (f *failingStore) Write(_ context.Context, _ string, _ []byte) error {
	return f.err
}

type failingTexts struct {
	host.TextVersions
	err error
}

func (f *failingTexts) PutVersion(_ context.Context, _ host.TextPut) (domain.TextVersion, error) {
	return domain.TextVersion{}, f.err
}

func TestWriterLifecycleAndFailures(t *testing.T) {
	raw := testsupport.CreateImageEpub(t)
	files := hosttest.New("vault")
	files.File("book.epub", string(raw))
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("book.epub", raw)
	texts.SetStore(pluginID, files.Store())
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	srv := NewWriter(filesOf(files), textsOf(texts), storeOf(files), queue, noReport)

	if err := srv.CreateText(t.Context(), Request{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}
	// Duplicate call while running is a no-op
	if err := srv.CreateText(t.Context(), Request{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}
	srv.Close()
	if err := held.Release(); err != nil {
		t.Error(err)
	}
	// Call after Close is a no-op
	if err := srv.CreateText(t.Context(), Request{Path: "book.epub"}); err != nil {
		t.Fatal(err)
	}

	// Failure reporting when file does not exist
	q2 := hosttest.NewQueue()
	srv2 := NewWriter(filesOf(files), textsOf(texts), storeOf(files), q2, noReport)
	t.Cleanup(srv2.Close)
	createText(t, srv2, Request{Path: "missing.epub"})
	tasks := q2.ListTasks()
	if len(tasks) != 1 || tasks[0].FailureReason == "" {
		t.Errorf("expected failed task for missing file, got %+v", tasks)
	}

	// EnqueueWork failure
	q3 := &failingQueue{Queue: hosttest.NewQueue(), enqueueErr: errors.New("queue down")}
	srv3 := NewWriter(filesOf(files), textsOf(texts), storeOf(files), q3, noReport)
	t.Cleanup(srv3.Close)
	createText(t, srv3, Request{Path: "book.epub"})
	if tasks := q3.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "queue down") {
		t.Errorf("expected queue down task failure, got %+v", tasks)
	}

	// ReportTask failure
	q4 := &failingQueue{Queue: hosttest.NewQueue(), reportErr: errors.New("report failed")}
	srv4 := NewWriter(filesOf(files), textsOf(texts), storeOf(files), q4, noReport)
	t.Cleanup(srv4.Close)
	createText(t, srv4, Request{Path: "book.epub"})
	if tasks := q4.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "report failed") {
		t.Errorf("expected report failed task failure, got %+v", tasks)
	}

	// Store Write failure
	q5 := hosttest.NewQueue()
	srv5 := NewWriter(filesOf(files), textsOf(texts), func(domain.VaultID) host.StoreWriter {
		return &failingStore{StoreWriter: files.Store(), err: errors.New("store full")}
	}, q5, noReport)
	t.Cleanup(srv5.Close)
	createText(t, srv5, Request{Path: "book.epub"})
	if tasks := q5.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "store full") {
		t.Errorf("expected store full task failure, got %+v", tasks)
	}

	// PutVersion failure
	q6 := hosttest.NewQueue()
	srv6 := NewWriter(filesOf(files), func(domain.VaultID) host.TextVersions {
		return &failingTexts{TextVersions: texts, err: errors.New("put version failed")}
	}, storeOf(files), q6, noReport)
	t.Cleanup(srv6.Close)
	createText(t, srv6, Request{Path: "book.epub"})
	if tasks := q6.ListTasks(); len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "put version failed") {
		t.Errorf("expected put version failed task failure, got %+v", tasks)
	}

	// Sequential CreateText after completion works (verifies delete(s.running, key))
	q7 := hosttest.NewQueue()
	srv7 := NewWriter(filesOf(files), textsOf(texts), storeOf(files), q7, noReport)
	t.Cleanup(srv7.Close)
	createText(t, srv7, Request{Path: "book.epub"})
	guided := testsupport.CreateGuidedEpub(t)
	files.File("book.epub", string(guided))
	texts.File("book.epub", guided)
	createText(t, srv7, Request{Path: "book.epub"})
	if live := readLive(t, texts, "book.epub"); !strings.Contains(live.Text, "Prologue") {
		t.Errorf("second CreateText did not update live version, got %q", live.Text)
	}
}

func noReport(string, error) {}
