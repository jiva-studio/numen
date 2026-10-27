package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
)

// createGuidedEpub is a book whose spine is the fourth child of the package element, after a guide,
// and whose first chapter holds a paragraph longer than one segment, set off by three spaces.
func createGuidedEpub(t *testing.T) []byte {
	t.Helper()
	return createZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Guided</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest>
    <item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="c2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <guide><reference type="text" href="c1.xhtml"/></guide>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`),
		"EPUB/c1.xhtml": []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head>
<body><h1>Prologue</h1><p>   ` + strings.Repeat("word ", 20_000) + `</p></body></html>`),
		"EPUB/c2.xhtml": []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Two</title></head>
<body><p>Epilogue of the  guided book</p></body></html>`),
	})
}

// newGuidedBookServer is a book server over a vault holding the guided book as guided.epub.
func newGuidedBookServer(t *testing.T) (*BookServer, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "guided.epub"), createGuidedEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}
	books := NewBookServer(hosttest.New("vault"), func() string { return dir })
	t.Cleanup(books.Close)
	return books, dir
}

func TestGetCfiNamesTheCharacterAtAnOffset(t *testing.T) {
	books, _ := newGuidedBookServer(t)
	book, err := books.GetBook(t.Context(), connect.NewRequest(&epubv1.GetBookRequest{Path: "guided.epub"}))
	if err != nil {
		t.Fatal(err)
	}
	second := book.Msg.GetDocuments()[1].GetOffset()

	got, err := books.GetCfi(t.Context(), connect.NewRequest(&epubv1.GetCfiRequest{Path: "guided.epub", Offset: second + int32(len("Epilogue of the "))}))

	if err != nil || got.Msg.GetCfi() != "epubcfi(/8/4!/4/2/1:17)" {
		t.Errorf("GetCfi = %v (%v), want epubcfi(/8/4!/4/2/1:17)", got, err)
	}
}

func TestFindOffsetReturnsTheOffsetACFINames(t *testing.T) {
	books, _ := newGuidedBookServer(t)
	book, err := books.GetBook(t.Context(), connect.NewRequest(&epubv1.GetBookRequest{Path: "guided.epub"}))
	if err != nil {
		t.Fatal(err)
	}
	second := book.Msg.GetDocuments()[1].GetOffset()

	got, err := books.FindOffset(t.Context(), connect.NewRequest(&epubv1.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/8/4!/4/2/1:17)"}))

	want := second + int32(len("Epilogue of the "))
	if err != nil || got.Msg.GetOffset() != want {
		t.Errorf("FindOffset = %v (%v), want offset %d", got, err, want)
	}
}

func TestGetCfiAndFindOffsetFollowTheFileWhenItChanges(t *testing.T) {
	books, dir := newGuidedBookServer(t)
	first, err := books.GetCfi(t.Context(), connect.NewRequest(&epubv1.GetCfiRequest{Path: "guided.epub", Offset: 0}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "guided.epub"), createPictureEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := books.GetCfi(t.Context(), connect.NewRequest(&epubv1.GetCfiRequest{Path: "guided.epub", Offset: 0}))

	if err != nil || first.Msg.GetCfi() != "epubcfi(/8/2!/4/2/1:0)" || second.Msg.GetCfi() != "epubcfi(/6/2!/4/2/1:0)" {
		t.Errorf("the CFIs of offset 0 are %v and %v (%v), want the guided book's and then the pictured book's", first, second, err)
	}
}

func TestFindOffsetRefusesAStringThatIsNoCFI(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.FindOffset(t.Context(), connect.NewRequest(&epubv1.FindOffsetRequest{Path: "guided.epub", Cfi: "/8/4!/4/2"}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("FindOffset of no CFI: %v, want invalid argument", err)
	}
}

func TestFindOffsetFindsNoDocumentTheSpineDoesNotHold(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.FindOffset(t.Context(), connect.NewRequest(&epubv1.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/6/4!/4/2/1:0)"}))

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("FindOffset of a CFI outside the spine: %v, want not found", err)
	}
}

func TestGetCfiRefusesNoPath(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.GetCfi(t.Context(), connect.NewRequest(&epubv1.GetCfiRequest{}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("GetCfi without a path: %v, want invalid argument", err)
	}
}

func TestGetCfiFindsNothingInABookWithNoText(t *testing.T) {
	dir := t.TempDir()
	raw := createZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata/>
<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="c1"/></spine></package>`),
		"EPUB/c1.xhtml": []byte(`<html><body><img src="cover.png"/></body></html>`),
	})
	if err := os.WriteFile(filepath.Join(dir, "blank.epub"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	books := NewBookServer(hosttest.New("vault"), func() string { return dir })
	t.Cleanup(books.Close)

	_, err := books.GetCfi(t.Context(), connect.NewRequest(&epubv1.GetCfiRequest{Path: "blank.epub"}))

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("GetCfi in a book with no text: %v, want not found", err)
	}
}
