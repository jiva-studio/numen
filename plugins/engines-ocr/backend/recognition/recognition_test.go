package recognition_test

import (
	"context"
	"errors"
	"image"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

const pluginID = "md.numen.engines.ocr"

// pages returns "page N" for the Nth page, in one block whose two lines are in the top
// and the bottom half of the page, and counts the pages.
type pages struct {
	mu        sync.Mutex
	pageCount int
	err       error
	says      string
}

func (p *pages) GetRecognitionModel() engine.Model {
	return engine.Model{Layout: "layout", Recogniser: "lines", DPI: 72, From: "a test"}
}

func (p *pages) Recognise(_ context.Context, page image.Image) ([]engine.Block, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pageCount++
	if p.err != nil {
		return nil, p.err
	}
	if p.says == "" {
		return nil, nil
	}
	b := page.Bounds()
	first := p.says + " one"
	text := first + " " + p.says + " two"
	return []engine.Block{
		{Label: "doc_title", Text: "Heading", IsHeading: true},
		{Label: "text", Text: text, Boxes: []engine.Box{
			{Rect: image.Rect(0, 0, b.Dx(), b.Dy()/2), Span: engine.ByteSpan{To: len(first)}},
			{Rect: image.Rect(0, b.Dy()/2, b.Dx(), b.Dy()), Span: engine.ByteSpan{From: len(first) + 1, To: len(text)}},
		}},
	}, nil
}

func (p *pages) getPageCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pageCount
}

// scene is one vault holding a scan and a PDF with a text layer, and the plugin over it.
type scene struct {
	vault      *hosttest.Vault
	store      host.Store
	texts      *hosttest.Texts
	queue      *hosttest.Queue
	recogniser *pages
	run        *recognition.Recognition
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("../document/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func newScene(t *testing.T) *scene {
	t.Helper()
	s := &scene{
		vault:      hosttest.New("Physics"),
		texts:      hosttest.NewTexts("Physics", pluginID),
		queue:      hosttest.NewQueue(),
		recogniser: &pages{says: "page"},
	}
	s.store = s.vault.Store()
	for path, name := range map[string]string{"books/scan.pdf": "scan.pdf", "books/layer.pdf": "tiny.pdf"} {
		raw := readFixture(t, name)
		s.vault.File(path, raw)
		s.texts.File(path, []byte(raw))
	}
	run, err := recognition.New(
		recognition.Host{Vault: s.vault, Store: s.store, Texts: s.texts, Queue: s.queue},
		func(context.Context) (recognition.Recogniser, error) { return s.recogniser, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	s.run = run
	return s
}

func (s *scene) readLive(t *testing.T, path string, provider host.TextProvider) domain.VersionText {
	t.Helper()
	s.texts.Provide(pluginID, provider)
	listing, err := s.texts.ListVersions(t.Context(), path)
	if err != nil || listing.Live == "" {
		t.Fatalf("%s has no live version: %+v (%v)", path, listing, err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			text, err := host.ReadWholeText(t.Context(), s.texts, version)
			if err != nil {
				t.Fatal(err)
			}
			return text
		}
	}
	t.Fatalf("the live version %s is not listed", listing.Live)
	return domain.VersionText{}
}

func TestRecognisePDFWritesPositionedText(t *testing.T) {
	s := newScene(t)

	res, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"})
	if err != nil || res.Pages != 2 || res.IsSkipped || res.IsEmpty {
		t.Fatalf("recognised %+v (%v), want two pages", res, err)
	}
	text := s.readLive(t, "books/scan.pdf", recognition.NewProvider(s.run, s.store))
	if text.Version.Format != domain.TextFormatPositioned || text.Version.Key != recognition.VersionKey {
		t.Fatalf("the version is %+v", text.Version)
	}
	if !strings.Contains(text.Text, "page two\n\nHeading") {
		t.Errorf("the text reads %q, want the pages in order with their blocks apart", text.Text)
	}
	offset := strings.LastIndex(text.Text, "page two")
	boxes := text.GetBoxes(domain.ByteSpan{From: offset, To: offset + len("page two")})
	if len(boxes) != 1 || boxes[0].Page != 1 || boxes[0].MinY != 0.5 || boxes[0].MaxY != 1 {
		t.Errorf("boxes of the last line: %+v, want the bottom half of the second page", boxes)
	}
	if got := text.Locate(offset); got != "Heading, page 2 of the file" {
		t.Errorf("location of the last line: %q", got)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a finished recognition still shows %+v", tasks)
	}
}

func TestRecogniseSkipsTextLayerUnlessUserStarted(t *testing.T) {
	s := newScene(t)

	res, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/layer.pdf"})
	if err != nil || !res.IsSkipped || s.recogniser.getPageCount() != 0 {
		t.Fatalf("a PDF with a layer, unasked: %+v (%v), %d pages read", res, err, s.recogniser.getPageCount())
	}
	res, err = s.run.Recognise(t.Context(), recognition.Request{Path: "books/layer.pdf", IsUserStarted: true})
	if err != nil || res.IsSkipped || res.Pages == 0 || s.recogniser.getPageCount() != res.Pages {
		t.Fatalf("a PDF with a layer, asked: %+v (%v), %d pages read", res, err, s.recogniser.getPageCount())
	}
}

func TestRecogniseSkipsUnchangedFile(t *testing.T) {
	s := newScene(t)
	for _, isUserStarted := range []bool{false, false, true} {
		if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf", IsUserStarted: isUserStarted}); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.recogniser.getPageCount(); got != 2 {
		t.Fatalf("%d pages were read, want the two of one run", got)
	}

	s.vault.File("books/scan.pdf", readFixture(t, "scan.pdf")+"\n% a second printing\n")
	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}
	if got := s.recogniser.getPageCount(); got != 4 {
		t.Fatalf("%d pages were read, want the replaced file read again", got)
	}
}

func TestRecogniseEmptyScanKeepsNothing(t *testing.T) {
	s := newScene(t)
	s.recogniser.says = ""

	res, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"})

	if err != nil || !res.IsEmpty {
		t.Fatalf("recognised %+v (%v), want nothing", res, err)
	}
	if listing, _ := s.texts.ListVersions(t.Context(), "books/scan.pdf"); len(listing.Versions) != 0 {
		t.Errorf("a scan that says nothing has versions %+v", listing.Versions)
	}
	if files, _ := s.store.List(t.Context()); len(files) != 0 {
		t.Errorf("a scan that says nothing keeps %+v", files)
	}
}

func TestRecogniseFailureReportsTaskError(t *testing.T) {
	s := newScene(t)
	s.recogniser.err = errors.New("the model ran out of memory")

	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf", IsUserStarted: true}); err == nil {
		t.Fatal("a recogniser that failed recognised the file")
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].Error, "out of memory") || !tasks[0].IsUserStarted {
		t.Fatalf("tasks %+v, want the failure pageCount", tasks)
	}

	s.recogniser.err = nil
	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %+v after the file was recognised", tasks)
	}
}

func TestRecogniseMissingFileFails(t *testing.T) {
	s := newScene(t)
	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/gone.pdf"}); !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("a file that is not there answered %v", err)
	}
}

func TestNewRequiresHostAndRecogniser(t *testing.T) {
	v := hosttest.New("Physics")
	open := func(context.Context) (recognition.Recogniser, error) { return &pages{}, nil }
	full := recognition.Host{Vault: v, Store: v.Store(), Texts: hosttest.NewTexts("Physics", pluginID), Queue: hosttest.NewQueue()}
	for name, h := range map[string]recognition.Host{
		"no vault": {Store: full.Store, Texts: full.Texts, Queue: full.Queue},
		"no store": {Vault: full.Vault, Texts: full.Texts, Queue: full.Queue},
		"no texts": {Vault: full.Vault, Store: full.Store, Queue: full.Queue},
		"no queue": {Vault: full.Vault, Store: full.Store, Texts: full.Texts},
	} {
		if _, err := recognition.New(h, open); err == nil {
			t.Errorf("%s: a recognition was made", name)
		}
	}
	if _, err := recognition.New(full, nil); err == nil {
		t.Error("a recognition was made with nothing to recognise with")
	}
}
