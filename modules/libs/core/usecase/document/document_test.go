package document_test

import (
	"bytes"
	"errors"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

const scanPath = "library/scan.pdf"

func TestGetDocumentAnswersTheShapeOfEachPage(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{scanPath: "the bytes of a scan"})
	held := &shelf{scan: scan{leaves: []leaf{{wide: 600, high: 800}, {unmeasured: true}, {wide: 300, high: 400}}}}

	got, err := document.NewGetDocument(readers, held).Execute(t.Context(), v, scanPath)
	if err != nil {
		t.Fatal(err)
	}

	if got.Fingerprint.Path != scanPath || got.Fingerprint.Size != int64(len("the bytes of a scan")) {
		t.Errorf("the document was read from %+v", got.Fingerprint)
	}
	want := []document.PageSize{{Width: 600, Height: 800}, {}, {Width: 300, Height: 400}}
	if len(got.Pages) != len(want) {
		t.Fatalf("the document has %d pages, want %d", len(got.Pages), len(want))
	}
	for i := range want {
		if got.Pages[i] != want[i] {
			t.Errorf("page %d is %+v, want %+v", i, got.Pages[i], want[i])
		}
	}
}

func TestGetDocumentRefusesAFileTheVaultDoesNotHold(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, nil)

	_, err := document.NewGetDocument(readers, &shelf{}).Execute(t.Context(), v, scanPath)

	if err == nil {
		t.Fatal("a file the vault does not hold was read")
	}
}

func TestGetPageDrawsAJPEGOnceHoweverOftenItIsAsked(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{scanPath: "the bytes of a scan"})
	held := &shelf{scan: scan{leaves: []leaf{{wide: 600, high: 800}}}}
	pages := &drawn{}
	get := document.NewGetPage(readers, held, pages)
	asked := document.PageKey{
		Document: document.Fingerprint{Path: scanPath, Size: int64(len("the bytes of a scan"))},
		Width:    64,
	}
	first, err := document.NewGetDocument(readers, held).Execute(t.Context(), v, scanPath)
	if err != nil {
		t.Fatal(err)
	}
	asked.Document.Mtime = first.Fingerprint.Mtime

	for range 3 {
		body, err := get.Execute(t.Context(), v, asked)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := jpeg.Decode(bytes.NewReader(body)); err != nil {
			t.Fatalf("the page is no JPEG: %v", err)
		}
	}

	if pages.count != 1 {
		t.Errorf("the page was drawn %d times", pages.count)
	}
}

func TestGetPageRefusesAFileThatChangedSinceTheAddressWasGivenOut(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{scanPath: "the bytes of a scan"})
	held := &shelf{scan: scan{leaves: []leaf{{wide: 600, high: 800}}}}
	pages := &drawn{}
	stale := document.PageKey{Document: document.Fingerprint{Path: scanPath, Size: 1, Mtime: 2}, Width: 64}

	_, err := document.NewGetPage(readers, held, pages).Execute(t.Context(), v, stale)

	if !errors.Is(err, document.ErrChanged) {
		t.Fatalf("error is %v, want ErrChanged", err)
	}
	if held.opened != 0 || pages.count != 0 {
		t.Errorf("a stale address opened %d documents and drew %d pages", held.opened, pages.count)
	}
}

func TestGetPageRefusesAPageTheDocumentDoesNotHave(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{scanPath: "the bytes of a scan"})
	held := &shelf{scan: scan{leaves: []leaf{{wide: 600, high: 800}}}}
	found, err := document.NewGetDocument(readers, held).Execute(t.Context(), v, scanPath)
	if err != nil {
		t.Fatal(err)
	}
	past := document.PageKey{Document: found.Fingerprint, Page: 1, Width: 64}

	_, err = document.NewGetPage(readers, held, &drawn{}).Execute(t.Context(), v, past)

	if !errors.Is(err, document.ErrNoPage) {
		t.Fatalf("error is %v, want ErrNoPage", err)
	}
}

const bookPath = "library/book.epub"

func TestGetBookAnswersWhatTheBookIsAndWhereItWasRead(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: bookOf(t)})

	got, err := document.NewGetBook(readers, kept{}).Execute(t.Context(), v, bookPath)
	if err != nil {
		t.Fatal(err)
	}

	if got.Title != "A Reflowed Book" || got.Fingerprint.Path != bookPath || got.Fingerprint.Size == 0 {
		t.Errorf("the book is %q read from %+v", got.Title, got.Fingerprint)
	}
}

func TestGetBookMarkupAnswersOneDocumentOfTheBook(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: bookOf(t)})
	book, err := document.NewGetBook(readers, kept{}).Execute(t.Context(), v, bookPath)
	if err != nil {
		t.Fatal(err)
	}
	get := document.NewGetBookMarkup(readers, kept{})

	markup, err := get.Execute(t.Context(), v, bookPath, &book.Fingerprint, "OEBPS/first.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markup, "Alpha omega.") {
		t.Errorf("the markup is %q", markup)
	}

	stale := document.Fingerprint{Size: 1, Mtime: 2}
	if _, err := get.Execute(t.Context(), v, bookPath, &stale, "OEBPS/first.xhtml"); !errors.Is(err, document.ErrChanged) {
		t.Errorf("a stale baseline was answered with %v", err)
	}
}

func TestGetBookPictureSettlesTheTypeByTheBytes(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: bookOf(t)})
	book, err := document.NewGetBook(readers, kept{}).Execute(t.Context(), v, bookPath)
	if err != nil {
		t.Fatal(err)
	}
	get := document.NewGetBookPicture(readers, kept{})

	got, err := get.Execute(t.Context(), v, bookPath, book.Fingerprint, "OEBPS/pictures/plate.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "image/png" || string(got.Body) != plate {
		t.Errorf("the picture is %q of %d bytes", got.Type, len(got.Body))
	}

	if _, err := get.Execute(t.Context(), v, bookPath, book.Fingerprint, "OEBPS/pictures/cover.svg"); !errors.Is(err, document.ErrNotAPicture) {
		t.Errorf("a document in the archive was answered with %v", err)
	}
}
