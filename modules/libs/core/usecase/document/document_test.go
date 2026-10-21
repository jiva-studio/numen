package document_test

import (
	"bytes"
	"errors"
	"image/jpeg"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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
	want := [][2]float64{{600, 800}, {}, {300, 400}}
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
	staleKey := document.PageKey{Document: document.Fingerprint{Path: scanPath, Size: 1, Mtime: 2}, Width: 64}

	_, err := document.NewGetPage(readers, held, pages).Execute(t.Context(), v, staleKey)

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

func TestGetBookPictureSettlesTheTypeByTheBytes(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: "the bytes of a book"})
	held := newBook()
	get := document.NewGetBookPicture(readers, held, http.DetectContentType)
	named := fingerprintOf(t, v)

	got, err := get.Execute(t.Context(), v, bookPath, named, "OEBPS/pictures/plate.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "image/png" || string(got.Body) != plate {
		t.Errorf("the picture is %q of %d bytes", got.Type, len(got.Body))
	}
	if len(held.asked) != 1 || held.asked[0].Path != named.Path || held.asked[0].Size != named.Size ||
		held.asked[0].ModTime.UnixNano() != named.Mtime {
		t.Errorf("the archive was asked at %+v, want once at %+v", held.asked, named)
	}

	if _, err := get.Execute(t.Context(), v, bookPath, named, "OEBPS/pictures/cover.svg"); !errors.Is(err, document.ErrNotAPicture) {
		t.Errorf("a document in the archive was answered with %v", err)
	}
}

func TestGetBookPictureRefusesAFileThatIsNotTheBytesNamed(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: "the bytes of a book"})
	held := newBook()
	get := document.NewGetBookPicture(readers, held, http.DetectContentType)

	staleFp := document.Fingerprint{Path: bookPath, Size: 1, Mtime: 2}
	if _, err := get.Execute(t.Context(), v, bookPath, staleFp, "OEBPS/pictures/plate.png"); !errors.Is(err, document.ErrChanged) {
		t.Errorf("a stale fingerprint was answered with %v", err)
	}
	if len(held.asked) != 0 {
		t.Errorf("the archive was asked %d times for a file that is not the one named", len(held.asked))
	}
}

func TestGetBookPictureAnswersWithErrChangedWhereTheArchiveSaysTheFileChanged(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: "the bytes of a book"})
	held := newBook()
	held.failure = port.ErrBookChanged
	get := document.NewGetBookPicture(readers, held, http.DetectContentType)

	_, err := get.Execute(t.Context(), v, bookPath, fingerprintOf(t, v), "OEBPS/pictures/plate.png")
	if !errors.Is(err, document.ErrChanged) {
		t.Errorf("err = %v, want ErrChanged", err)
	}
}

func TestGetBookPictureRefusesAnEntryTheArchiveLacksAndAFileThatIsNoBook(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{bookPath: "the bytes of a book"})
	named := fingerprintOf(t, v)

	get := document.NewGetBookPicture(readers, newBook(), http.DetectContentType)
	if _, err := get.Execute(t.Context(), v, bookPath, named, "OEBPS/missing.png"); !errors.Is(err, document.ErrNoEntry) {
		t.Errorf("an entry the archive lacks was answered with %v", err)
	}

	broken := newBook()
	broken.failure = document.ErrNotADocument
	get = document.NewGetBookPicture(readers, broken, http.DetectContentType)
	if _, err := get.Execute(t.Context(), v, bookPath, named, "OEBPS/pictures/plate.png"); !errors.Is(err, document.ErrNotADocument) {
		t.Errorf("a file that is no book was answered with %v", err)
	}
}

func TestGetBookPictureRefusesAPathTheVaultLacks(t *testing.T) {
	t.Parallel()
	v := vaultOf(t, map[string]string{})
	get := document.NewGetBookPicture(readers, newBook(), http.DetectContentType)
	if _, err := get.Execute(t.Context(), v, bookPath, document.Fingerprint{}, "OEBPS/pictures/plate.png"); err == nil {
		t.Error("a path the vault lacks was answered")
	}
}

// fingerprintOf is what the vault says about the book.
func fingerprintOf(t *testing.T, v domain.Vault) document.Fingerprint {
	t.Helper()
	found, err := document.NewGetDocument(readers, &shelf{}).Execute(t.Context(), v, bookPath)
	if err != nil {
		t.Fatal(err)
	}
	return found.Fingerprint
}

func TestGetBookAnswersTheShapeTheBooksGive(t *testing.T) {
	t.Parallel()
	held := newBook()

	got, err := document.NewGetBook(held).Execute(t.Context(), bookPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != bookPath || len(got.Sections) != 1 {
		t.Errorf("the book is %+v", got)
	}

	held.failure = port.ErrNoBook
	if _, err := document.NewGetBook(held).Execute(t.Context(), bookPath); !errors.Is(err, port.ErrNoBook) {
		t.Errorf("a path with no book was answered with %v", err)
	}
}

func TestReadBookMarkupAsksWithTheBaselineAndPassesTheRefusalOn(t *testing.T) {
	t.Parallel()
	held := newBook()
	held.entries["OEBPS/first.xhtml"] = []byte("<p>Alpha</p>")
	read := document.NewReadBookMarkup(held)
	baseline := &domain.Fingerprint{Path: bookPath, Size: 7}

	got, err := read.Execute(t.Context(), bookPath, "OEBPS/first.xhtml", baseline)
	if err != nil || got != "<p>Alpha</p>" {
		t.Fatalf("the markup is %q, err = %v", got, err)
	}
	if len(held.asked) != 1 || held.asked[0] != *baseline {
		t.Errorf("the book was asked at %+v, want once at %+v", held.asked, *baseline)
	}
	if _, err := read.Execute(t.Context(), bookPath, "OEBPS/gone.xhtml", nil); !errors.Is(err, document.ErrNoEntry) {
		t.Errorf("a section the book lacks was answered with %v", err)
	}
}
