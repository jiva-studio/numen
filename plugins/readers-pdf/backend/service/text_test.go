package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

const pluginID = "md.numen.readers.pdf"

// newVault is a vault holding one file under name with the bytes of a fixture, the texts of that
// vault with the plugin providing, and the plugin's text server.
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
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	server := NewTextServer(func() string { return root }, texts, texts.GetStore(pluginID))
	return texts, server, raw
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

func TestTheTextLayerOfAFileIsItsLiveVersion(t *testing.T) {
	texts, server, raw := newVault(t, "books/tiny.pdf", "tiny.pdf")

	if err := server.CreateText(t.Context(), host.TextCreation{Path: "books/tiny.pdf"}); err != nil {
		t.Fatalf("CreateText: %v", err)
	}

	text := readLive(t, texts, "books/tiny.pdf")
	if text.Version.Kind != textKind {
		t.Errorf("kind %q, want %q", text.Version.Kind, textKind)
	}
	if text.Version.SourceHash != domain.HashContent(raw) || !text.Version.IsFromFile() {
		t.Errorf("version %+v is not made from the file as it is", text.Version)
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
	if err := server.CreateText(t.Context(), host.TextCreation{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	text := readLive(t, texts, "tiny.pdf")

	from := strings.Index(text.Text, "Delta")
	boxes := getBoxes(text, domain.ByteSpan{From: from, To: from + len("Delta epsilon")})

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
}

func TestTheOutlineOpensTheSectionsOfTheLayer(t *testing.T) {
	texts, server, raw := newVault(t, "outline.pdf", "outline.pdf")
	if err := server.CreateText(t.Context(), host.TextCreation{Path: "outline.pdf"}); err != nil {
		t.Fatal(err)
	}
	book, err := pdf.Read(raw)
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
	book, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	all := make([]int, len(book.Pages))
	for i := range all {
		all[i] = i
	}
	boxes, err := book.Highlights(raw, all)
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	sections := 0
	for page := range book.Pages {
		for _, part := range getLayerParts(book, page, page+1, boxes) {
			if selector, err := host.ParseSelector(part.Selector); err != nil || selector.Page != page+1 {
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

			if err := server.CreateText(t.Context(), host.TextCreation{Path: name}); err != nil {
				t.Fatalf("CreateText: %v", err)
			}

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

func TestTheReaderMakesTextOfFilesOnly(t *testing.T) {
	_, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")

	err := server.CreateText(t.Context(), host.TextCreation{Path: "tiny.pdf", Source: "md.numen.engines.ocr/page"})

	if !errors.Is(err, host.ErrInvalidVersion) {
		t.Errorf("CreateText of a version = %v, want ErrInvalidVersion", err)
	}
}

func TestTheDocumentServiceLightsRunsOfTheLiveVersion(t *testing.T) {
	texts, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")
	if err := server.CreateText(t.Context(), host.TextCreation{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	documents := NewDocumentServerIn(server.vaultRoot(), texts)
	text := readLive(t, texts, "tiny.pdf")
	beta, zeta := strings.Index(text.Text, "beta"), strings.Index(text.Text, "zeta")

	res, err := documents.GetHighlights(t.Context(), connect.NewRequest(&numenv1.GetHighlightsRequest{
		Path: "tiny.pdf",
		Spans: []*numenv1.Span{
			{From: int32(zeta), To: int32(zeta + len("zeta"))},
			{From: int32(beta), To: int32(zeta + len("zeta"))},
		},
	}))
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	runs := res.Msg.GetRuns()
	if len(runs) != 2 {
		t.Fatalf("%d runs, want the 2 asked about", len(runs))
	}
	if places := runs[0].GetPlaces(); len(places) != 1 || places[0].GetPage() != 1 || len(places[0].GetRects()) != 1 {
		t.Errorf("zeta is placed at %v, want one rectangle on page 1", places)
	}
	places := runs[1].GetPlaces()
	if len(places) != 2 || places[0].GetPage() != 0 || places[1].GetPage() != 1 {
		t.Fatalf("beta to zeta is placed at %v, want pages 0 and 1", places)
	}
	if len(places[0].GetRects()) != 2 || len(places[1].GetRects()) != 3 {
		t.Errorf("beta to zeta covers %d and %d words, want 2 and 3", len(places[0].GetRects()), len(places[1].GetRects()))
	}
}

func TestADocumentWithNoLiveVersionIsLitNowhere(t *testing.T) {
	texts, server, _ := newVault(t, "tiny.pdf", "tiny.pdf")
	documents := NewDocumentServerIn(server.vaultRoot(), texts)

	res, err := documents.GetHighlights(t.Context(), connect.NewRequest(&numenv1.GetHighlightsRequest{
		Path: "tiny.pdf", Spans: []*numenv1.Span{{From: 0, To: 5}},
	}))
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	if runs := res.Msg.GetRuns(); len(runs) != 1 || len(runs[0].GetPlaces()) != 0 {
		t.Errorf("runs %v, want one standing nowhere", runs)
	}
}
