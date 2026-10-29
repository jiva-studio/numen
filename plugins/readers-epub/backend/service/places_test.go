package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
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
func newGuidedBookServer(t *testing.T) (*BookServer, *hosttest.Vault) {
	t.Helper()
	files := hosttest.New("vault")
	files.File("guided.epub", string(createGuidedEpub(t)))
	books := NewBookServer(files)
	t.Cleanup(books.Close)
	return books, files
}

func TestGetCfiNamesTheCharacterAtAnOffset(t *testing.T) {
	books, _ := newGuidedBookServer(t)
	book, err := books.GetBook(t.Context(), api.GetBookRequest{Path: "guided.epub"})
	if err != nil {
		t.Fatal(err)
	}
	second := book.Documents[1].Offset

	got, err := books.GetCfi(t.Context(), api.GetCfiRequest{Path: "guided.epub", Offset: second + int(len("Epilogue of the "))})

	if err != nil || got.Cfi != "epubcfi(/8/4!/4/2/1:17)" {
		t.Errorf("GetCfi = %v (%v), want epubcfi(/8/4!/4/2/1:17)", got, err)
	}
}

func TestFindOffsetReturnsTheOffsetACFINames(t *testing.T) {
	books, _ := newGuidedBookServer(t)
	book, err := books.GetBook(t.Context(), api.GetBookRequest{Path: "guided.epub"})
	if err != nil {
		t.Fatal(err)
	}
	second := book.Documents[1].Offset

	got, err := books.FindOffset(t.Context(), api.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/8/4!/4/2/1:17)"})

	want := second + int(len("Epilogue of the "))
	if err != nil || got.Offset != want {
		t.Errorf("FindOffset = %v (%v), want offset %d", got, err, want)
	}
}

func TestGetCfiAndFindOffsetFollowTheFileWhenItChanges(t *testing.T) {
	books, files := newGuidedBookServer(t)
	first, err := books.GetCfi(t.Context(), api.GetCfiRequest{Path: "guided.epub", Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	files.File("guided.epub", string(createImageEpub(t)))

	second, err := books.GetCfi(t.Context(), api.GetCfiRequest{Path: "guided.epub", Offset: 0})

	if err != nil || first.Cfi != "epubcfi(/8/2!/4/2/1:0)" || second.Cfi != "epubcfi(/6/2!/4/2/1:0)" {
		t.Errorf("the CFIs of offset 0 are %v and %v (%v), want the guided book's and then the pictured book's", first, second, err)
	}
}

func TestFindOffset_ErrorsOnInvalidCFI(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.FindOffset(t.Context(), api.FindOffsetRequest{Path: "guided.epub", Cfi: "/8/4!/4/2"})

	if codeOf(err) != server.CodeInvalidArgument {
		t.Errorf("FindOffset of no CFI: %v, want invalid argument", err)
	}
}

func TestFindOffsetFindsNoDocumentTheSpineDoesNotHold(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.FindOffset(t.Context(), api.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/6/4!/4/2/1:0)"})

	if codeOf(err) != server.CodeNotFound {
		t.Errorf("FindOffset of a CFI outside the spine: %v, want not found", err)
	}
}

func TestGetCfi_ErrorsOnEmptyPath(t *testing.T) {
	books, _ := newGuidedBookServer(t)

	_, err := books.GetCfi(t.Context(), api.GetCfiRequest{})

	if codeOf(err) != server.CodeInvalidArgument {
		t.Errorf("GetCfi without a path: %v, want invalid argument", err)
	}
}

func TestGetCfiFindsNothingInABookWithNoText(t *testing.T) {
	raw := createZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata/>
<manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="c1"/></spine></package>`),
		"EPUB/c1.xhtml": []byte(`<html><body><img src="cover.png"/></body></html>`),
	})
	files := hosttest.New("vault")
	files.File("blank.epub", string(raw))
	books := NewBookServer(files)
	t.Cleanup(books.Close)

	_, err := books.GetCfi(t.Context(), api.GetCfiRequest{Path: "blank.epub"})

	if codeOf(err) != server.CodeNotFound {
		t.Errorf("GetCfi in a book with no text: %v, want not found", err)
	}
}
