package source

import (
	"context"
	"errors"
	"image"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
)

// A shelf is what a recognition is written to, in memory.
type shelf struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newShelf() *shelf { return &shelf{files: map[string][]byte{}} }

func (s *shelf) Open(domain.Vault) (port.DerivedStore, error) { return s, nil }

func (s *shelf) Read(_ context.Context, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, held := s.files[name]
	if !held {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s *shelf) Write(_ context.Context, name string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append([]byte(nil), content...)
	return nil
}

func (s *shelf) Append(_ context.Context, name string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append(s.files[name], content...)
	return nil
}

func (s *shelf) Remove(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, name)
	return nil
}

func (s *shelf) names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for name := range s.files {
		out = append(out, name)
	}
	return out
}

// A speaker reads every page as the same thing, or as nothing, and counts how
// many pages it was shown.
type speaker struct {
	says  string
	pages int
	stop  func(int)
}

func (s *speaker) Recognition() port.Recognition {
	return port.Recognition{Layout: "layout", Recogniser: "reader", DPI: 300, From: "a test"}
}

func (s *speaker) Read(ctx context.Context, _ image.Image) ([]ocr.Block, error) {
	s.pages++
	if s.stop != nil {
		s.stop(s.pages)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.says == "" {
		return nil, nil
	}
	return []ocr.Block{{Label: "text", Text: s.says}}, nil
}

func (s *speaker) Close() error { return nil }

// documentPath is where the document sits in the vault under test.
const documentPath = "library/scan.pdf"

// recogniser is a Recognise over one vault holding one document.
//
// The document is the pdf package's own fixture: four pages that a reader can
// open and count, which is all this needs — what the pages say comes from the
// model, and the model here is a fake.
func recogniser(t *testing.T, says string) (Recognise, domain.Vault, *store, *shelf, *speaker) {
	t.Helper()
	raw, err := os.ReadFile("../../pdf/testdata/outline.pdf")
	if err != nil {
		t.Fatal(err)
	}
	shelved := newLibrary()
	shelved.hold(documentPath, domain.KindBook, raw, 1)
	v := first
	readers := vaults{first.ID: shelved}
	index := newStore()
	shelf := newShelf()
	model := &speaker{says: says}
	return Recognise{
		Readers: readers,
		Sources: index,
		Derived: shelf,
		By:      model,
		Batch:   1,
	}, v, index, shelf, model
}

func TestWhatIsReadIsWrittenDownAndClaimed(t *testing.T) {
	u, v, index, shelf, model := recogniser(t, "What the page says.")

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Pages || res.Pages == 0 {
		t.Fatalf("read %d of %d pages", res.Read, res.Pages)
	}
	if model.pages != res.Pages {
		t.Errorf("the model was shown %d pages and the document has %d", model.pages, res.Pages)
	}

	// The artifact is named by the hash of what was read, and the source now
	// says its text is there.
	src := index.sources[v.ID][documentPath]
	if src.TextPath == "" {
		t.Fatal("the source does not say where its text is")
	}
	raw, err := shelf.Read(t.Context(), src.TextPath)
	if err != nil {
		t.Fatalf("the artifact is not where the source says: %v", err)
	}
	prose, marks := ocr.Read(raw)
	if !strings.Contains(prose, "What the page says.") {
		t.Errorf("the artifact says %q", prose)
	}
	if len(marks) != res.Pages {
		t.Errorf("the artifact names %d pages and the document has %d", len(marks), res.Pages)
	}

	// No recipe, so the source owes its text: cutting it into windows is
	// extraction's, which is the one thing that knows the sizes.
	if src.Recipe != "" {
		t.Errorf("recognition wrote a recipe: %q", src.Recipe)
	}
}

func TestADocumentSayingNothingWritesNothing(t *testing.T) {
	// An empty artifact would stand in for a text layer that worked, and a
	// source naming one has nothing to fall back to.
	u, v, index, shelf, _ := recogniser(t, "")

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Empty {
		t.Error("a document that says nothing was not reported as saying nothing")
	}
	if names := shelf.names(); len(names) != 0 {
		t.Errorf("it wrote %v", names)
	}
	if src, held := index.sources[v.ID][documentPath]; held && src.TextPath != "" {
		t.Errorf("the source was pointed at %q", src.TextPath)
	}
}

func TestADocumentAlreadyReadIsNotReadAgain(t *testing.T) {
	u, v, _, _, model := recogniser(t, "Something.")

	if _, err := u.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	first := model.pages

	if _, err := u.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if model.pages != first {
		t.Errorf("the model was shown %d more pages", model.pages-first)
	}
}

func TestARunThatStoppedIsTakenUpAgain(t *testing.T) {
	// A document is an hour, and a person may stop one. What was read is on
	// disk, and the next run begins where the last one left off rather than at
	// the beginning.
	u, v, _, shelf, model := recogniser(t, "A page.")

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}
	stopped := model.pages
	if stopped < 2 {
		t.Fatalf("it stopped after %d pages, which is too early to tell anything", stopped)
	}

	// Nothing final was written: a half-read document is not an artifact, and
	// a scan meeting one cuts the document itself.
	for _, name := range shelf.names() {
		if strings.HasSuffix(name, ".txt") {
			t.Errorf("a stopped run left %q behind", name)
		}
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed == 0 {
		t.Error("the second run began at the beginning")
	}
	if model.pages >= stopped+res.Pages {
		t.Errorf("the second run read %d pages, and the document has %d", model.pages-stopped, res.Pages)
	}
	if res.Read != res.Pages {
		t.Errorf("read %d of %d pages after taking it up again", res.Read, res.Pages)
	}
}

func TestWhatReadItIsKeptBesideWhatItRead(t *testing.T) {
	u, v, index, shelf, _ := recogniser(t, "Something.")

	if _, err := u.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	hash := strings.TrimSuffix(strings.TrimPrefix(src.TextPath, "ocr/"), ".txt")

	raw, err := shelf.Read(t.Context(), text.Beside("ocr", hash))
	if err != nil {
		t.Fatalf("nothing says what read the document: %v", err)
	}
	for _, want := range []string{"layout", "reader", "300", "a test"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("what was kept does not mention %q: %s", want, raw)
		}
	}
}

func TestAReadingDeletedByHandIsNoticed(t *testing.T) {
	// The store is a folder on the person's disk and they may empty it. A source
	// standing on a reading that is gone answers a search with nothing, and
	// would go on doing so: its recipe is the one in use, so nothing else asks
	// after it.
	u, v, index, shelf, _ := recogniser(t, "Something.")
	if _, err := u.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	stood := index.sources[v.ID][documentPath].TextPath
	if stood == "" {
		t.Fatal("the source does not stand on a reading")
	}

	// Cut it once, so it is a source that owes nothing.
	extract := Extract{Readers: u.Readers, Sources: index, Owing: index, Derived: shelf}
	if _, err := extract.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if index.sources[v.ID][documentPath].Recipe == "" {
		t.Fatal("the source was not cut, so this cannot tell anything")
	}

	if err := shelf.Remove(t.Context(), stood); err != nil {
		t.Fatal(err)
	}
	res, err := extract.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Forgotten != 1 {
		t.Errorf("noticed %d readings gone, want 1", res.Forgotten)
	}
	if got := index.sources[v.ID][documentPath].TextPath; got != "" {
		t.Errorf("the source still stands on %q", got)
	}
}
