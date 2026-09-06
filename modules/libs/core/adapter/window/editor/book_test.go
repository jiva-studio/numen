package editor

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// Where the book a test reads sits in its vault, and the documents it is read
// in.
const (
	reflowed = "library/reflowed.epub"
	firstDoc = "OEBPS/first.xhtml"
	lastDoc  = "OEBPS/last.xhtml"
)

// plate is a picture the book carries, and cover is the same picture written
// in a format that is a document rather than a picture.
var (
	plate = "\x89PNG\r\n\x1a\n" + strings.Repeat("pixels", 40)
	cover = `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
)

// What a book is: what it is called, the documents it is read in, what it names
// inside them, and which bytes all of that was read from.
func TestWhatABookIsIsItsDocumentsAndWhatItNamesInThem(t *testing.T) {
	api, _ := readFrom(t)

	told := whatBook(t, api)
	if told.GetTitle() != "A Reflowed Book" {
		t.Errorf("the book is called %q", told.GetTitle())
	}
	if !told.GetReflowable() {
		t.Error("a book made for a screen came back as one laid out once")
	}
	if told.GetProgression() != v1.PageProgression_PAGE_PROGRESSION_RIGHT_TO_LEFT {
		t.Errorf("the pages progress %v", told.GetProgression())
	}
	if got := len(told.GetDocuments()); got != 2 || int(told.GetSpine()) != got {
		t.Fatalf("the book is read in %d documents, and says it holds %d", got, told.GetSpine())
	}
	if got := told.GetDocuments()[0]; got.GetPath() != firstDoc || !got.GetLinear() {
		t.Errorf("the first document is %+v", got)
	}
	// A document the spine sets apart from the reading order is in the book all
	// the same, and stands after the one before it.
	last := told.GetDocuments()[1]
	if last.GetPath() != lastDoc || last.GetLinear() {
		t.Errorf("the last document is %+v", last)
	}
	if last.GetOffset() <= told.GetDocuments()[0].GetOffset() || last.GetLength() <= 0 {
		t.Errorf("the documents run %+v", told.GetDocuments())
	}
	if titles := partsOf(told); len(titles) != 2 || titles[0] != "The First Part" {
		t.Errorf("the book names %v", titles)
	}
	if got := told.GetPrinted(); len(got) != 1 || got[0].GetLabel() != "17" {
		t.Errorf("the printed pages are %v", got)
	}
	if told.GetPages() < 1 {
		t.Errorf("the book is read in %d pages", told.GetPages())
	}
	if told.GetFingerprint().GetPath() != reflowed || told.GetFingerprint().GetSize() == 0 {
		t.Errorf("the book was read from %+v", told.GetFingerprint())
	}
}

// A file that is no book is answered as one a person can do something about,
// and so is one the vault does not hold.
func TestAFileThatIsNoBook(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{
		reflowed: "this is not an archive at all",
	})
	api := &API{Readers: filesystem.VaultReaders{}, Viewer: looking(pdf.Documents{})}
	api.show(vault)
	t.Cleanup(api.Viewer.close)

	for _, one := range []struct {
		path string
		want connect.Code
	}{
		{reflowed, connect.CodeFailedPrecondition},
		{"library/nothing.epub", connect.CodeNotFound},
		{"../outside.epub", connect.CodeNotFound},
	} {
		t.Run(one.path, func(t *testing.T) {
			_, err := api.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: one.path}))
			if got := connect.CodeOf(err); got != one.want {
				t.Errorf("%s was refused %v, want %v", one.path, got, one.want)
			}
		})
	}
}

// One document of a book is markup, and the window reads every offset it needs
// off it.
func TestOneDocumentOfABookIsAnsweredAsMarkup(t *testing.T) {
	api, handler := readFrom(t)
	print := printOf(t, api, reflowed)

	out := ask(handler, markupOf(reflowed, firstDoc, print))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for a document and got %d: %s", out.Code, out.Body)
	}
	if said := out.Header().Get("Content-Type"); said != "text/html; charset=utf-8" {
		t.Errorf("the document came back as %q", said)
	}
	if said := out.Header().Get("X-Content-Type-Options"); said != "nosniff" {
		t.Errorf("the document is sniffed: %q", said)
	}
	page := out.Body.String()
	if !strings.Contains(page, epub.OffsetAttribute+`="`) {
		t.Errorf("the markup says no offsets:\n%s", page)
	}
	if !strings.Contains(page, "The First Part") {
		t.Errorf("the markup is not the document:\n%s", page)
	}
	// A book writes markup of its own and it is words, not markup.
	if strings.Contains(page, "<script") {
		t.Errorf("the book's own script reached the window:\n%s", page)
	}

	// A document the book was not read from is not one to ask for.
	if out := ask(handler, markupOf(reflowed, "OEBPS/gone.xhtml", print)); out.Code != http.StatusNotFound {
		t.Errorf("a document the book does not hold was answered %d", out.Code)
	}
}

// A picture the book carries is drawn out of the archive, and what settles the
// type is the bytes.
func TestAPictureTheBookCarries(t *testing.T) {
	api, handler := readFrom(t)
	print := printOf(t, api, reflowed)

	out := ask(handler, entryOf(reflowed, "OEBPS/pictures/plate.png", print))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for a picture and got %d: %s", out.Code, out.Body)
	}
	if said := out.Header().Get("Content-Type"); said != "image/png" {
		t.Errorf("the picture came back as %q", said)
	}
	if said := out.Header().Get("X-Content-Type-Options"); said != "nosniff" {
		t.Errorf("the picture is sniffed: %q", said)
	}
	if out.Body.String() != plate {
		t.Errorf("the picture is %d bytes", out.Body.Len())
	}
}

// The bytes of an entry are served from the window's own origin, so what is
// served is a picture and the bytes say so. What the manifest calls a file is
// the book's own claim about it.
func TestAnEntryThatIsNotAPictureIsNotServed(t *testing.T) {
	api, handler := readFrom(t)
	print := printOf(t, api, reflowed)

	for _, entry := range []string{
		firstDoc,
		"OEBPS/content.opf",
		"META-INF/container.xml",
		"OEBPS/pictures/cover.svg",
		"OEBPS/pictures/gone.png",
	} {
		t.Run(entry, func(t *testing.T) {
			out := ask(handler, entryOf(reflowed, entry, print))
			if out.Code == http.StatusOK {
				t.Errorf("%s was served as %q", entry, out.Header().Get("Content-Type"))
			}
		})
	}
}

// An address names the bytes it is about, and is answered while the file is
// still those bytes and no longer.
func TestAnAddressIntoABookThatChanged(t *testing.T) {
	api, handler := readFrom(t)
	stale := printOf(t, api, reflowed)
	stale.size++

	for _, url := range []string{
		markupOf(reflowed, firstDoc, stale),
		entryOf(reflowed, "OEBPS/pictures/plate.png", stale),
	} {
		if out := ask(handler, url); out.Code != http.StatusNotFound {
			t.Errorf("%s was answered %d", url, out.Code)
		}
	}
}

// A path that leaves the vault is refused, and so is one the vault holds
// nothing at.
func TestAPathTheVaultDoesNotHoldIsNoBook(t *testing.T) {
	api, handler := readFrom(t)
	print := printOf(t, api, reflowed)

	for _, path := range []string{"../outside.epub", "/etc/passwd", "library/nothing.epub"} {
		for _, url := range []string{
			markupOf(path, firstDoc, print),
			entryOf(path, "OEBPS/pictures/plate.png", print),
		} {
			if out := ask(handler, url); out.Code == http.StatusOK {
				t.Errorf("%s was answered", url)
			}
		}
	}
}

// An archive is unpacked and parsed once, however many chapters are turned.
func TestABookIsReadOnceHoweverManyChaptersAreTurned(t *testing.T) {
	held := holding()
	t.Cleanup(held.close)

	var reads int
	var mu sync.Mutex
	read := func() (*epub.Book, error) {
		mu.Lock()
		defer mu.Unlock()
		reads++
		return epub.Read(bookOf(t))
	}
	print := fingerprint{path: reflowed, size: 1, mtime: 2}
	for range 3 {
		if _, err := held.take(t.Context(), print, read); err != nil {
			t.Fatalf("take the book: %v", err)
		}
	}
	if reads != 1 {
		t.Errorf("the archive was read %d times", reads)
	}
}

// Few are held: a book is the whole of its text in memory beside the archive it
// was read out of.
func TestOnlyTheBooksInFrontOfThePersonAreHeld(t *testing.T) {
	held := holding()
	t.Cleanup(held.close)

	raw := bookOf(t)
	for at := range mostRead + 1 {
		print := fingerprint{path: fmt.Sprintf("library/%d.epub", at), size: int64(at)}
		if _, err := held.take(t.Context(), print, func() (*epub.Book, error) { return epub.Read(raw) }); err != nil {
			t.Fatalf("take the book: %v", err)
		}
	}
	held.mu.Lock()
	open := len(held.open)
	held.mu.Unlock()
	if open != mostRead {
		t.Errorf("%d books are held, want %d", open, mostRead)
	}
}

// A caller that ran out of patience is told the book is busy, and the reading
// goes on and is there for the next ask.
func TestABookThatIsStillBeingReadIsBusy(t *testing.T) {
	held := holding()
	t.Cleanup(held.close)

	gate := make(chan struct{})
	read := func() (*epub.Book, error) {
		<-gate
		return epub.Read(bookOf(t))
	}
	print := fingerprint{path: reflowed, size: 1}

	waited, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	if _, err := held.take(waited, print, read); !errors.Is(err, errBusy) {
		t.Errorf("a book still being read answered %v", err)
	}
	close(gate)
	if _, err := held.take(t.Context(), print, read); err != nil {
		t.Errorf("the book the first ask left open: %v", err)
	}
}

// readFrom is a window looking at one book of its vault.
func readFrom(t *testing.T) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{reflowed: string(bookOf(t))})
	api := &API{Readers: filesystem.VaultReaders{}, Viewer: looking(pdf.Documents{})}
	api.show(vault)
	t.Cleanup(api.Viewer.close)
	return api, api.Serving(http.NotFoundHandler())
}

// whatBook is what the book is, as the window is told it.
func whatBook(t *testing.T, api *API) *v1.GetBookResponse {
	t.Helper()
	out, err := api.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if err != nil {
		t.Fatalf("asked what the book is and was refused: %v", err)
	}
	return out.Msg
}

// named is what the book calls its parts.
func partsOf(told *v1.GetBookResponse) []string {
	var out []string
	for _, part := range told.GetParts() {
		out = append(out, part.GetTitle())
	}
	return out
}

// bookOf is an EPUB a window reads: two documents, the second set apart from the
// reading order, a picture, a page of the printed book, and markup of its own
// in the text and in what the picture is described as.
func bookOf(t *testing.T) []byte {
	t.Helper()
	return archived(t, map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>A Reflowed Book</dc:title>
    <dc:identifier id="id">reflowed</dc:identifier>
  </metadata>
  <manifest>
    <item id="a" href="first.xhtml" media-type="application/xhtml+xml"/>
    <item id="b" href="last.xhtml" media-type="application/xhtml+xml"/>
    <item id="c" href="pictures/plate.png" media-type="image/png"/>
    <item id="d" href="pictures/cover.svg" media-type="image/svg+xml"/>
  </manifest>
  <spine page-progression-direction="rtl">
    <itemref idref="a"/>
    <itemref idref="b" linear="no"/>
  </spine>
</package>`,
		firstDoc: `<html><body>
  <h1>The First Part</h1>
  <p>Alpha &lt;/p&gt;&lt;script&gt;alert(1)&lt;/script&gt; omega.</p>
  <span epub:type="pagebreak" id="p17" title="17"></span>
  <p><img src="pictures/plate.png" alt="A plate"/></p>
</body></html>`,
		lastDoc:                    `<html><body><h1>The Last Part</h1><p>Set apart from the reading order.</p></body></html>`,
		"OEBPS/pictures/plate.png": plate,
		"OEBPS/pictures/cover.svg": cover,
	})
}

// archived writes the files into one archive, the mimetype first because the
// specification asks for it.
func archived(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	write := func(name string) {
		method := zip.Deflate
		if name == "mimetype" {
			method = zip.Store
		}
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatalf("write the fixture: %v", err)
		}
		if _, err := entry.Write([]byte(parts[name])); err != nil {
			t.Fatalf("write the fixture: %v", err)
		}
	}
	write("mimetype")
	for name := range parts {
		if name != "mimetype" {
			write(name)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("write the fixture: %v", err)
	}
	return out.Bytes()
}
