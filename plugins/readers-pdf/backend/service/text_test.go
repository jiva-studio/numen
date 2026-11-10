package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// compilationDirVariable names the folder TestMain makes for the compiled PDFium of the test binary.
const compilationDirVariable = "NUMEN_PDF_TEST_COMPILATION_DIR"

func newTestWorkers(tb testing.TB) *pdf.Workers {
	tb.Helper()
	workers := pdf.NewWorkers()
	compilation, err := wazero.NewCompilationCacheWithDir(os.Getenv(compilationDirVariable))
	if err != nil {
		tb.Fatal(err)
	}
	workers.SetCompilationCache(compilation)
	tb.Cleanup(func() {
		if err := errors.Join(workers.Close(), compilation.Close(context.WithoutCancel(tb.Context()))); err != nil {
			tb.Error(err)
		}
	})
	return workers
}

// TestMain makes the folder the pools of the tests keep the compiled PDFium in.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	dir, err := os.MkdirTemp("", "numen-pdf-compilation-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "make the compilation folder: %v\n", err)
		return 1
	}
	if err := os.Setenv(compilationDirVariable, dir); err != nil {
		fmt.Fprintf(os.Stderr, "set %s: %v\n", compilationDirVariable, err)
		return 1
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintf(os.Stderr, "remove the compilation folder: %v\n", err)
		code = 1
	}
	return code
}

func newVault(t *testing.T, name, fixture string) (*hosttest.Texts, *TextServer, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	return newVaultOf(t, name, raw)
}

func newVaultOf(t *testing.T, name string, raw []byte) (*hosttest.Texts, *TextServer, []byte) {
	t.Helper()
	files := hosttest.OnDisk(t, "vault", map[string]string{name: string(raw)})
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	texts.SetStore(pluginID, files.Store())
	server := newTextServerOver(t, filesOf(files), files, texts, hosttest.NewQueue())
	t.Cleanup(server.Close)
	return texts, server, raw
}

func newTextServerOver(t *testing.T, vaults func(domain.VaultID) vaultFiles, home *hosttest.Vault, texts *hosttest.Texts, queue host.Queue) *TextServer {
	t.Helper()
	return NewTextServer(
		vaults,
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return home.Store().With(id) },
		queue, newTestWorkers(t))
}

func filesOf(vault *hosttest.Vault) func(domain.VaultID) vaultFiles {
	return func(id domain.VaultID) vaultFiles { return vault.With(id) }
}

func createText(t *testing.T, server *TextServer, req TextRequest) {
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

func TestTheTextLayerOfAFileIsItsLiveVersion(t *testing.T) {
	texts, server, raw := newVault(t, "books/tiny.pdf", "tiny.pdf")

	createText(t, server, TextRequest{Path: "books/tiny.pdf"})

	text := readLive(t, texts, "books/tiny.pdf")
	if text.Version().Kind != textKind {
		t.Errorf("kind %q, want %q", text.Version().Kind, textKind)
	}
	if text.Version().InputHash != domain.HashContent(raw) || !text.Version().IsFromFile() {
		t.Errorf("version %+v is not made from the file as it is", text.Version())
	}
	if want := "Alpha beta gamma\nDelta epsilon zeta\n"; text.Text != want {
		t.Errorf("text %q, want %q", text.Text, want)
	}
	if got := text.GetSelector(strings.Index(text.Text, "epsilon")); !strings.HasPrefix(string(got), `{"page":2,`) {
		t.Errorf("epsilon has selector %s, want page 2", got)
	}
}

func TestARunOfTheLayerIsLitOverTheWordsItCovers(t *testing.T) {
	texts, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")
	createText(t, server, TextRequest{Path: "tiny.pdf"})
	text := readLive(t, texts, "tiny.pdf")

	from := strings.Index(text.Text, "Delta")
	boxes := getBoxes(text, pdf.ByteRange{Start: from, End: from + len("Delta epsilon")})

	if len(boxes) != 2 {
		t.Fatalf("%d boxes, want one a word of the two: %+v", len(boxes), boxes)
	}
	for _, box := range boxes {
		if box.Page != 1 {
			t.Errorf("a box on page %d, want 1", box.Page)
		}
		if box.MaxX <= box.MinX || box.MaxY <= box.MinY {
			t.Errorf("box %+v covers nothing", box.Rect)
		}
	}

	customVersion := domain.NewVersionText(domain.TextVersion{}, []domain.TextSegment{
		{Text: "hello", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
	})
	customBoxes := getBoxes(customVersion, pdf.ByteRange{Start: 0, End: 5})
	if len(customBoxes) != 1 || customBoxes[0].MinX != 0.1 || customBoxes[0].MinY != 0.2 || customBoxes[0].MaxX != 0.3 || customBoxes[0].MaxY != 0.4 {
		t.Errorf("customBoxes = %+v, want [0.1, 0.2, 0.3, 0.4]", customBoxes)
	}
}

func TestTheOutlineOpensTheSectionsOfTheLayer(t *testing.T) {
	texts, server, raw := newVault(t, "outline.pdf", "outline.pdf")
	createText(t, server, TextRequest{Path: "outline.pdf"})
	book, err := newTestWorkers(t).Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	sections := readLive(t, texts, "outline.pdf").GetHeadings()

	if len(sections) != len(book.Parts) {
		t.Fatalf("%d sections, want the %d the outline names", len(sections), len(book.Parts))
	}
	for i, part := range book.Parts {
		if sections[i].Title != part.Title || sections[i].Offset != part.Offset {
			t.Errorf("section %d is %q at %d, want %q at %d", i, sections[i].Title, sections[i].Offset, part.Title, part.Offset)
		}
	}
}

func TestPagesOfTheLayerJoinIntoTheWholeText(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "outline.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	workers := newTestWorkers(t)
	book, err := workers.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	all := make([]int, len(book.Pages))
	for i := range all {
		all[i] = i
	}
	boxes, err := book.Highlights(workers, raw, all)
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	sections := 0
	for page := range book.Pages {
		layerParts, err := getLayerParts(book, page, page+1, boxes)
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range layerParts {
			if selector, err := readPageSelector(part.Selector); err != nil || selector.Page != page+1 {
				t.Errorf("a segment of page %d has selector %s", page, part.Selector)
			}
			if part.Heading != "" {
				sections++
			}
			joined.WriteString(part.Text)
		}
	}

	if joined.String() != book.Text {
		t.Errorf("the pages join into %q, want %q", joined.String(), book.Text)
	}
	if sections != len(book.Parts) {
		t.Errorf("the pages open %d sections, want %d", sections, len(book.Parts))
	}
}

func TestAFileWithNoTextLayerGetsNoVersion(t *testing.T) {
	for name, raw := range map[string][]byte{
		"scan.pdf":      nil,
		"corrupted.pdf": []byte("INVALID_PDF_HEADER"),
	} {
		t.Run(name, func(t *testing.T) {
			var texts *hosttest.Texts
			var server *TextServer
			if raw == nil {
				texts, server, _ = newVault(t, name, name)
			} else {
				texts, server, _ = newVaultOf(t, name, raw)
			}

			createText(t, server, TextRequest{Path: name})

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

func TestGetHighlightsPlacesRunsOfTheLiveVersion(t *testing.T) {
	texts, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")
	createText(t, server, TextRequest{Path: "tiny.pdf"})
	documents := newDocumentServer(server, texts)
	text := readLive(t, texts, "tiny.pdf")
	beta, zeta := strings.Index(text.Text, "beta"), strings.Index(text.Text, "zeta")

	res, err := documents.GetHighlights(t.Context(), GetHighlightsRequest{
		Path: "tiny.pdf",
		Ranges: []Range{
			{Start: zeta, End: zeta + len("zeta")},
			{Start: beta, End: zeta + len("zeta")},
		},
	})
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	runs := res.Runs
	if len(runs) != 2 {
		t.Fatalf("%d runs, want the 2 asked about", len(runs))
	}
	if places := runs[0].Places; len(places) != 1 || places[0].Page != 1 || len(places[0].Rects) != 1 {
		t.Errorf("zeta is placed at %v, want one rectangle on page 1", places)
	}
	places := runs[1].Places
	if len(places) != 2 || places[0].Page != 0 || places[1].Page != 1 {
		t.Fatalf("beta to zeta is placed at %v, want pages 0 and 1", places)
	}
	if len(places[0].Rects) != 2 || len(places[1].Rects) != 3 {
		t.Errorf("beta to zeta covers %d and %d words, want 2 and 3", len(places[0].Rects), len(places[1].Rects))
	}
}

func TestGetHighlightsPlacesNothingWithoutLiveVersion(t *testing.T) {
	texts, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")
	documents := newDocumentServer(server, texts)

	res, err := documents.GetHighlights(t.Context(), GetHighlightsRequest{
		Path: "tiny.pdf", Ranges: []Range{{Start: 0, End: 5}},
	})
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	if runs := res.Runs; len(runs) != 1 || len(runs[0].Places) != 0 {
		t.Errorf("runs %v, want one run with no places", runs)
	}
}

func newDocumentServer(server *TextServer, texts *hosttest.Texts) *DocumentServer {
	return NewDocumentServer(func(id domain.VaultID) host.FileReader { return server.vaults(id) }, func(id domain.VaultID) LiveTexts { return texts.With(id) }, server.workers)
}

func TestTheTextLayerIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	physics := hosttest.New("Physics")
	chemistry := hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.File("tiny.pdf", string(raw))
	texts := hosttest.NewTexts("Physics", pluginID)
	other := texts.With("Chemistry")
	other.File("tiny.pdf", raw)
	other.SetStore(pluginID, chemistry.Store())
	server := newTextServerOver(t, filesOf(physics), physics, texts, hosttest.NewQueue())
	t.Cleanup(server.Close)

	createText(t, server, TextRequest{Vault: "Chemistry", Path: "tiny.pdf"})

	if text := readLive(t, other, "tiny.pdf"); text.Version().Kind != textKind {
		t.Errorf("Chemistry's live version is %+v", text.Version())
	}
	inShown, err := texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
}

func TestTextServerLifecycleAndVaultStatus(t *testing.T) {
	if got := getStatus(host.ErrNoVault); got != 403 {
		t.Errorf("getStatus(ErrNoVault) = %d, want 403", got)
	}
	if got := getStatus(host.ErrOutside); got != 403 {
		t.Errorf("getStatus(ErrOutside) = %d, want 403", got)
	}
	if got := getStatus(host.ErrNotExist); got != 404 {
		t.Errorf("getStatus(ErrNotExist) = %d, want 404", got)
	}

	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	files := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(raw)})
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("tiny.pdf", raw)
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	texts.SetStore(pluginID, files.Store())
	srv := newTextServerOver(t, filesOf(files), files, texts, queue)

	if err := srv.CreateText(t.Context(), TextRequest{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	// Duplicate call while running is a no-op
	if err := srv.CreateText(t.Context(), TextRequest{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	// Close cancels running task and unlocks
	srv.Close()
	if err := held.Release(); err != nil {
		t.Error(err)
	}

	// CreateText after Close is a no-op
	if err := srv.CreateText(t.Context(), TextRequest{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
}
