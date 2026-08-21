package source

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
)

// A shelf is what a recognition is written to, in memory.
type shelf struct {
	mu    sync.Mutex
	files map[string][]byte
	held  map[string]bool
}

func newShelf() *shelf { return &shelf{files: map[string][]byte{}, held: map[string]bool{}} }

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

func (s *shelf) Claim(_ context.Context, name string) (func() error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held[name] {
		return nil, port.ErrClaimed
	}
	s.held[name] = true
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.held, name)
		return nil
	}, nil
}

// hold takes a name the way another run holds it.
func (s *shelf) hold(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[name] = true
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
	says string
	// heads is what every page says in the region that opens a part of the
	// document. Empty is a page opening none.
	heads string
	// blank is the page, counted from one, that the model reads nothing on.
	blank int
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
	if s.says == "" || s.pages == s.blank {
		return nil, nil
	}
	// No two pages say the same thing, so a coordinate read at the wrong offset
	// names the wrong words.
	said := fmt.Sprintf("%s %d", s.says, s.pages)
	var out []ocr.Block
	if s.heads != "" {
		out = append(out, ocr.Block{Label: "doc_title", Text: fmt.Sprintf("%s %d", s.heads, s.pages), Head: true, Depth: 1})
	}
	return append(out, ocr.Block{
		Label: "text",
		Text:  said,
		Spans: []ocr.Span{{Box: image.Rect(10, 20, 30, 40), Length: len(said)}},
	}), nil
}

func (s *speaker) Close() error { return nil }

// documentPath is where the document sits in the vault under test.
const documentPath = "library/scan.pdf"

// recogniser is a Recognise over one vault holding one document.
//
// The document is the pdf package's own fixture: four pages that a reader can
// open and count, and that the document names none of, which is all this needs
// — what the pages say comes from the model, and the model here is a fake.
func recogniser(t *testing.T, says string) (Recognise, domain.Vault, *store, *shelf, *speaker) {
	t.Helper()
	return reading(t, says, "outline.pdf")
}

// reading is a Recognise over a vault holding the fixture named.
func reading(t *testing.T, says, fixture string) (Recognise, domain.Vault, *store, *shelf, *speaker) {
	t.Helper()
	raw, err := os.ReadFile("../../pdf/testdata/" + fixture)
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
	if src.TextFrom == "" {
		t.Fatal("the source does not say which producer made its text")
	}
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
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
	if src, held := index.sources[v.ID][documentPath]; held && src.TextFrom != "" {
		t.Errorf("the source was pointed at %q", src.TextFrom)
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
	hash := src.Hash

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
	stood := index.sources[v.ID][documentPath]
	if stood.TextFrom == "" {
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

	if err := shelf.Remove(t.Context(), text.Artifact(stood.TextFrom, stood.Hash)); err != nil {
		t.Fatal(err)
	}
	res, err := extract.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Forgotten != 1 {
		t.Errorf("noticed %d readings gone, want 1", res.Forgotten)
	}
	if got := index.sources[v.ID][documentPath].TextFrom; got != "" {
		t.Errorf("the source still stands on %q", got)
	}
}

// document is the bytes under test and the hash a reading of them is kept
// under.
func document(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../pdf/testdata/outline.pdf")
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint(raw)
}

// reads checks that every coordinate names the words it was read from, in the
// prose the artifact holds.
//
// The offsets rise, because the prose of every page written before this one
// stands in front of it.
func reads(t *testing.T, prose string, boxes []placed.Box, says string) {
	t.Helper()
	at := -1
	for i, box := range boxes {
		if box.Start < 0 || box.Start+box.Length > len(prose) {
			t.Errorf("box %d reaches %d of %d bytes", i, box.Start+box.Length, len(prose))
			continue
		}
		if box.Start <= at {
			t.Errorf("box %d begins at %d, and the one before it at %d", i, box.Start, at)
		}
		at = box.Start
		if got := prose[box.Start : box.Start+box.Length]; !strings.HasPrefix(got, says) {
			t.Errorf("box %d reads %q, and no page says that", i, got)
		}
	}
}

func TestADocumentAnotherRunHoldsIsLeftAlone(t *testing.T) {
	// The name is the hash of the document's bytes, and it is appended to for an
	// hour. One run holds it for all of that.
	u, v, index, shelf, model := recogniser(t, "A page.")
	shelf.hold(text.Partial("ocr", document(t)))

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Busy {
		t.Error("a document another run is reading was read again")
	}
	if model.pages != 0 {
		t.Errorf("the model was shown %d pages", model.pages)
	}
	if names := shelf.names(); len(names) != 0 {
		t.Errorf("it wrote %v", names)
	}
	if _, held := index.sources[v.ID][documentPath]; held {
		t.Error("the source was recorded")
	}
}

func TestEveryCoordinateNamesTheWordsItWasReadFrom(t *testing.T) {
	// Every page is its own batch here, so the offsets only come out right if
	// the prose written before each of them is carried forward.
	const says = "A page."
	u, v, index, shelf, _ := recogniser(t, says)

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	packed, err := shelf.Read(t.Context(), text.Boxes(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatalf("the coordinates are not beside the artifact: %v", err)
	}

	prose, _ := ocr.Read(raw)
	boxes := placed.Unpack(packed)
	if len(boxes) != res.Pages {
		t.Fatalf("%d coordinates over %d pages", len(boxes), res.Pages)
	}
	for i, box := range boxes {
		if box.Page != i {
			t.Errorf("coordinate %d is on page %d", i, box.Page)
		}
	}
	reads(t, prose, boxes, says)
}

func TestCoordinatesAheadOfTheCountAreDropped(t *testing.T) {
	// The coordinates of a batch are written before its pages are, so a run that
	// died between the two left coordinates the count does not claim.
	const says = "A page."
	u, v, index, shelf, model := recogniser(t, says)

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}
	stray := placed.Pack([]placed.Box{{Page: 2, Start: 9000, Length: 7}})
	if err := shelf.Append(t.Context(), text.Boxes("ocr", document(t)), stray); err != nil {
		t.Fatal(err)
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	packed, err := shelf.Read(t.Context(), text.Boxes(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}

	prose, _ := ocr.Read(raw)
	boxes := placed.Unpack(packed)
	if len(boxes) != res.Pages {
		t.Fatalf("%d coordinates over %d pages", len(boxes), res.Pages)
	}
	reads(t, prose, boxes, says)
}

func TestTheSourceIsCutAsItsPagesAreRead(t *testing.T) {
	// A book is an hour. The pages already read answer questions while the rest
	// of it is still being read.
	u, v, _, _, _ := recogniser(t, "A page.")

	cuts := 0
	u.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if cuts < res.Pages {
		t.Errorf("the source was cut %d times over %d pages", cuts, res.Pages)
	}
}

func TestABatchThatDidNotLandWholeIsReadAgain(t *testing.T) {
	// A count stands after the pages it claims. An append cut short leaves
	// pages no count claims, and they are read again.
	const says = "A page."
	u, v, index, shelf, model := recogniser(t, says)

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}

	// The tail of an append that never finished.
	partial := text.Partial("ocr", document(t))
	torn, err := shelf.Read(t.Context(), partial)
	if err != nil {
		t.Fatal(err)
	}
	if err := shelf.Write(t.Context(), partial, append(torn, "A pa"...)); err != nil {
		t.Fatal(err)
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	prose, marks := ocr.Read(raw)
	if len(marks) != res.Pages {
		t.Errorf("the artifact names %d pages and the document has %d", len(marks), res.Pages)
	}
	if strings.Contains(prose, resumeMark) {
		t.Errorf("a count is in the prose: %q", prose)
	}

	packed, err := shelf.Read(t.Context(), text.Boxes(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	reads(t, prose, placed.Unpack(packed), says)
}

func TestAPartialCarryingNoCountIsReadFromTheBeginning(t *testing.T) {
	// The count and the prose in front of it are one fact. A file carrying no
	// count has been read to nowhere.
	const says = "A page."
	u, v, index, shelf, model := recogniser(t, says)

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}

	partial := text.Partial("ocr", document(t))
	held, err := shelf.Read(t.Context(), partial)
	if err != nil {
		t.Fatal(err)
	}
	var without []string
	for _, line := range strings.SplitAfter(string(held), "\n") {
		if !strings.HasPrefix(line, resumeMark) {
			without = append(without, line)
		}
	}
	if err := shelf.Write(t.Context(), partial, []byte(strings.Join(without, ""))); err != nil {
		t.Fatal(err)
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 0 {
		t.Errorf("it took up a file carrying no count at page %d", res.Resumed)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	prose, marks := ocr.Read(raw)
	if len(marks) != res.Pages {
		t.Errorf("the artifact names %d pages and the document has %d", len(marks), res.Pages)
	}
	packed, err := shelf.Read(t.Context(), text.Boxes(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	reads(t, prose, placed.Unpack(packed), says)
}

func TestCoordinatesThatDidNotLandWholeAreNotReadAsRecords(t *testing.T) {
	// An append cut short inside a record leaves bytes that are not one, and
	// every record appended after them is read at a shifted offset.
	const says = "A page."
	u, v, index, shelf, model := recogniser(t, says)

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}
	name := text.Boxes("ocr", document(t))
	if err := shelf.Append(t.Context(), name, make([]byte, 10)); err != nil {
		t.Fatal(err)
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	packed, err := shelf.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	prose, _ := ocr.Read(raw)
	boxes := placed.Unpack(packed)
	if len(boxes) != res.Pages {
		t.Fatalf("%d coordinates over %d pages", len(boxes), res.Pages)
	}
	reads(t, prose, boxes, says)
}

// says is the prose the artifact holds.
func says(t *testing.T, v domain.Vault, index *store, written *shelf) string {
	t.Helper()
	prose, _ := ocr.Read(artifact(t, v, index, written))
	return prose
}

// artifact is what recognition wrote for the document under test.
func artifact(t *testing.T, v domain.Vault, index *store, written *shelf) []byte {
	t.Helper()
	src := index.sources[v.ID][documentPath]
	raw, err := written.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatalf("the artifact is not where the source says: %v", err)
	}
	return raw
}

func TestAReadingNamesItsParts(t *testing.T) {
	// The layout model names the headings and a sidecar keeps them, so a
	// recognised document divides into parts exactly as a book with an outline
	// does. What the scan made of the words does not matter: the part carries the
	// run of prose the heading is.
	u, v, index, shelf, model := recogniser(t, "A page.")
	model.heads = "IAYADEVA GOSVAMI"

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	parts, err := shelf.Read(t.Context(), text.Parts(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatalf("the parts are not beside the artifact: %v", err)
	}

	doc := text.Recognised(raw, parts, nil, nil)
	if len(doc.Places) != res.Pages {
		t.Fatalf("the reading names %d parts over %d pages", len(doc.Places), res.Pages)
	}
	for i, place := range doc.Places {
		want := fmt.Sprintf("IAYADEVA GOSVAMI %d", i+1)
		if place.Title != want {
			t.Errorf("part %d is called %q, want %q", i, place.Title, want)
		}
		if got := doc.Text[place.Offset : place.Offset+len(want)]; got != want {
			t.Errorf("part %d begins at %d, which reads %q", i, place.Offset, got)
		}
	}
}

func TestPartsAheadOfTheCountAreDropped(t *testing.T) {
	// The parts of a batch are written before its pages are, so a run that died
	// between the two left parts the count does not claim. One kept would name a
	// part of the prose that is not there.
	u, v, index, shelf, model := recogniser(t, "A page.")
	model.heads = "A part"

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(pages int) {
		if pages == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}
	stray := ocr.Pack([]ocr.Part{{Start: 9000, Length: 7, Depth: 1}})
	if err := shelf.Append(t.Context(), text.Parts("ocr", document(t)), stray); err != nil {
		t.Fatal(err)
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	src := index.sources[v.ID][documentPath]
	raw, err := shelf.Read(t.Context(), text.Artifact(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}
	parts, err := shelf.Read(t.Context(), text.Parts(src.TextFrom, src.Hash))
	if err != nil {
		t.Fatal(err)
	}

	doc := text.Recognised(raw, parts, nil, nil)
	if len(doc.Places) != res.Pages {
		t.Fatalf("the reading names %d parts over %d pages", len(doc.Places), res.Pages)
	}
	// A part carries a run of the prose, so the one thing every part has to be
	// is where its heading stands.
	for i, place := range doc.Places {
		if !strings.HasPrefix(place.Title, "A part ") {
			t.Errorf("part %d is called %q", i, place.Title)
		}
		at := place.Offset
		if at < 0 || at+len(place.Title) > len(doc.Text) {
			t.Fatalf("part %d begins at %d, and the prose is %d long", i, at, len(doc.Text))
		}
		if got := doc.Text[at : at+len(place.Title)]; got != place.Title {
			t.Errorf("part %d begins at %d, which reads %q", i, at, got)
		}
	}
}

func TestTheArtifactMarksOnePageForEachPageOfTheDocument(t *testing.T) {
	// A page is called where it stands among the marks, so the marks and the
	// pages have to be the same run of things in the same order. A page
	// nothing was read on is still marked.
	u, v, index, shelf, model := recogniser(t, "A page.")
	model.blank = 2

	res, err := u.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}

	prose, marks := ocr.Read(artifact(t, v, index, shelf))
	if len(marks) != res.Pages {
		t.Fatalf("the artifact marks %d pages and the document has %d", len(marks), res.Pages)
	}
	at := -1
	for i, mark := range marks {
		if mark.Offset <= at && i > 0 {
			t.Errorf("page %d begins at %d, and the page before it at %d", i, mark.Offset, at)
		}
		at = mark.Offset
		if mark.Offset < 0 || mark.Offset > len(prose) {
			t.Errorf("page %d begins at %d, and the prose is %d long", i, mark.Offset, len(prose))
		}
	}
}
