package editor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const reflowed = "library/reflowed.epub"

// plate is a picture the book carries, and cover is a picture written in a
// format that is a document.
var (
	plate = "\x89PNG\r\n\x1a\n" + strings.Repeat("pixels", 40)
	cover = `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
)

// fakeBookClient is a book plugin that holds the entries it is given, and
// records what it was asked.
type fakeBookClient struct {
	mu      sync.Mutex
	entries map[string][]byte
	failure error
	asked   []domain.Fingerprint
	markup  *markupAsk
	book    string
}

// markupAsk is what a plugin was asked of one section of a book.
type markupAsk struct {
	path, section string
	baseline      *domain.Fingerprint
}

func (c *fakeBookClient) GetBook(_ context.Context, path string) (domain.BookShape, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.book = path
	if c.failure != nil {
		return domain.BookShape{}, c.failure
	}
	return domain.BookShape{
		Title:       "A Reflowed Book",
		Sections:    []domain.BookSection{{Path: "OEBPS/first.xhtml", Offset: 0, Length: 11}},
		Parts:       []domain.BookPart{{Title: "One", Offset: 0, Level: 1}},
		Pages:       []domain.BookPage{{Label: "iv", Offset: 3}},
		PageCount:   2,
		TextBytes:   11,
		PageBytes:   6,
		Fingerprint: domain.Fingerprint{Path: path, Size: 5, ModTime: time.Unix(0, 7)},
	}, nil
}

func (c *fakeBookClient) ReadBookMarkup(
	_ context.Context, path, section string, baseline *domain.Fingerprint,
) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.markup = &markupAsk{path: path, section: section, baseline: baseline}
	if c.failure != nil {
		return "", c.failure
	}
	return "<p>Alpha omega.</p>", nil
}

func (c *fakeBookClient) ReadEntry(_ context.Context, baseline domain.Fingerprint, entry string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, baseline)
	if c.failure != nil {
		return nil, c.failure
	}
	body, target := c.entries[entry]
	if !target {
		return nil, document.ErrNoEntry
	}
	return body, nil
}

func (c *fakeBookClient) countAsked() int {
	return len(c.getAsked())
}

// getAsked is a copy of the fingerprints the plugin was asked at.
func (c *fakeBookClient) getAsked() []domain.Fingerprint {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.asked)
}

// readFrom is a window looking at one book of its vault, which the plugin it is
// given holds the entries of.
func readFrom(t *testing.T, client port.Book) (*Services, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{reflowed: "the bytes of a book"})
	api := newBooksAPI(filesystem.VaultReaders{}, client, newViewer(testsupport.Documents{}))
	api.state.setVault(vault)
	t.Cleanup(api.Document.viewer.close)
	return api, api.NewHandler(http.NotFoundHandler())
}

func newBookClient() *fakeBookClient {
	return &fakeBookClient{entries: map[string][]byte{
		"OEBPS/pictures/plate.png": []byte(plate),
		"OEBPS/pictures/cover.svg": []byte(cover),
		"OEBPS/first.xhtml":        []byte("<html><body>Alpha</body></html>"),
	}}
}

// pictureOf is where a picture a book carries is asked for: the place it has in
// the archive, each segment of it escaped on its own.
func pictureOf(path, entry string, mark document.Fingerprint) string {
	parts := strings.Split(entry, "/")
	for item, one := range parts {
		parts[item] = url.PathEscape(one)
	}
	return fmt.Sprintf("%s/%s?%s", assetURL(path), strings.Join(parts, "/"), formatFingerprint(mark))
}

// What a book is comes from the plugin, and is answered as the plugin said it.
func TestWhatABookIsIsAskedOfThePlugin(t *testing.T) {
	client := newBookClient()
	api, _ := readFrom(t, client)

	out, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetTitle() != "A Reflowed Book" || client.book != reflowed {
		t.Errorf("the book is %q, asked for %q", out.Msg.GetTitle(), client.book)
	}
	if out.Msg.GetPageCount() != 2 || out.Msg.GetTextBytes() != 11 || out.Msg.GetPageBytes() != 6 {
		t.Errorf("the book is %d pages, %d bytes, %d to a page", out.Msg.GetPageCount(), out.Msg.GetTextBytes(), out.Msg.GetPageBytes())
	}
	if docs := out.Msg.GetDocuments(); len(docs) != 1 || docs[0].GetPath() != "OEBPS/first.xhtml" || docs[0].GetLength() != 11 {
		t.Errorf("the documents are %v", docs)
	}
	if parts := out.Msg.GetParts(); len(parts) != 1 || parts[0].GetTitle() != "One" || parts[0].GetLevel() != 1 {
		t.Errorf("the parts are %v", parts)
	}
	if pages := out.Msg.GetPrintedPages(); len(pages) != 1 || pages[0].GetLabel() != "iv" || pages[0].GetOffset() != 3 {
		t.Errorf("the printed pages are %v", pages)
	}
	if mark := out.Msg.GetFingerprint(); mark.GetPath() != reflowed || mark.GetSize() != 5 || mark.GetMtime() != 7 {
		t.Errorf("the fingerprint is %v", mark)
	}
}

// given it.
func TestAErrorOfTheBookIsCoded(t *testing.T) {
	for why, want := range map[error]connect.Code{
		port.ErrNotADocument: connect.CodeInvalidArgument,
		port.ErrEncrypted:    connect.CodeFailedPrecondition,
		port.ErrNoBook:       connect.CodeNotFound,
		port.ErrNoBookEntry:  connect.CodeNotFound,
		port.ErrBookChanged:  connect.CodeAborted,
	} {
		client := newBookClient()
		client.failure = fmt.Errorf("a book: %w", why)
		api, _ := readFrom(t, client)

		_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
		if got := connect.CodeOf(err); got != want {
			t.Errorf("the book failed %v for %v, want %v", got, why, want)
		}
		_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
		if got := connect.CodeOf(err); got != want {
			t.Errorf("the markup failed %v for %v, want %v", got, why, want)
		}
	}
}

func TestOneDocumentOfABookIsAskedOfThePlugin(t *testing.T) {
	client := newBookClient()
	api, _ := readFrom(t, client)

	out, err := api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{
		Path: reflowed, Document: "OEBPS/first.xhtml",
		Baseline: &v1.Fingerprint{Path: reflowed, Size: 5, Mtime: 7},
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := client.markup
	if out.Msg.GetMarkup() != "<p>Alpha omega.</p>" || request.path != reflowed || request.section != "OEBPS/first.xhtml" {
		t.Errorf("the markup is %q, asked for %+v", out.Msg.GetMarkup(), request)
	}
	if want := (domain.Fingerprint{Path: reflowed, Size: 5, ModTime: time.Unix(0, 7)}); request.baseline == nil || !request.baseline.ModTime.Equal(want.ModTime) ||
		request.baseline.Path != want.Path || request.baseline.Size != want.Size {
		t.Errorf("the baseline is %+v, want %+v", request.baseline, want)
	}
}

// An ask that names no baseline is made with none.
func TestAMarkupAskWithNoBaselineNamesNone(t *testing.T) {
	client := newBookClient()
	api, _ := readFrom(t, client)

	if _, err := api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed})); err != nil {
		t.Fatal(err)
	}
	if client.markup.baseline != nil {
		t.Errorf("the baseline is %+v, want none", client.markup.baseline)
	}
}

// A window with no use case bound for books answers nothing about a book.
func TestABookIsNotAnsweredWithoutAUseCase(t *testing.T) {
	api, handler := readFrom(t, newBookClient())
	mark := printOf(t, api, reflowed)
	api.Book.getBook, api.Book.readBookMarkup, api.Book.getBookPicture = nil, nil, nil

	_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
		t.Errorf("a book was answered %v, want unimplemented", code)
	}
	_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
		t.Errorf("a document of a book was answered %v, want unimplemented", code)
	}
	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusNotImplemented {
		t.Errorf("a picture was answered %d, want not implemented", out.Code)
	}
}

// A book plugin that is not there, for any reason, is a build that cannot read
// books, and every ask of a book says so with the one code.
func TestAnAbsentBookPluginIsAnsweredAsUnimplemented(t *testing.T) {
	for name, why := range map[string]error{
		"not installed": &port.UnavailableError{Capability: "book", Reason: port.PluginNotInstalled},
		"disabled":      &port.UnavailableError{Capability: "book", Reason: port.PluginDisabled},
		"failed":        &port.UnavailableError{PluginID: "md.numen.readers.epub", Reason: port.PluginFailed, Err: errors.New("exit 1")},
		"no program": &port.UnavailableError{
			PluginID: "md.numen.readers.epub", Reason: port.PluginFailed, Err: fmt.Errorf("start bin/epub: %w", fs.ErrNotExist),
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := newBookClient()
			client.failure = why
			api, handler := readFrom(t, client)
			mark := printOf(t, api, reflowed)

			_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
			if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
				t.Errorf("a book was answered %v, want unimplemented", code)
			}
			_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
			if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
				t.Errorf("a document of a book was answered %v, want unimplemented", code)
			}
			if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusNotImplemented {
				t.Errorf("a picture was answered %d, want not implemented", out.Code)
			}
		})
	}
}

// A window over a build with no book plugin answers every ask as unimplemented.
func TestABuildWithNoBookPluginAnswersUnimplemented(t *testing.T) {
	api, handler := readFrom(t, absentBooks{})
	mark := printOf(t, api, reflowed)

	_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
		t.Errorf("a book was answered %v, want unimplemented", code)
	}
	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusNotImplemented {
		t.Errorf("a picture was answered %d, want not implemented", out.Code)
	}
}

// A code a failure carries is kept, the end of the caller's context is coded as
// such, and an uncoded failure is internal.
func TestAFailureOfTheBookKeepsItsCode(t *testing.T) {
	for why, want := range map[error]connect.Code{
		connect.NewError(connect.CodeNotFound, errors.New("no such book")): connect.CodeNotFound,
		connect.NewError(connect.CodePermissionDenied, errors.New("no")):   connect.CodePermissionDenied,
		context.DeadlineExceeded:       connect.CodeDeadlineExceeded,
		context.Canceled:               connect.CodeCanceled,
		errors.New("the plugin broke"): connect.CodeInternal,
	} {
		client := newBookClient()
		client.failure = why
		api, _ := readFrom(t, client)

		_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
		if code := connect.CodeOf(err); code != want {
			t.Errorf("%v was answered %v, want %v", why, code, want)
		}
		_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
		if code := connect.CodeOf(err); code != want {
			t.Errorf("%v was answered %v for markup, want %v", why, code, want)
		}
	}
}

// A picture the book carries is drawn out of the archive the plugin reads, and
// what settles the type is the bytes.
func TestAPictureTheBookCarries(t *testing.T) {
	client := newBookClient()
	api, handler := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for a picture and got %d: %s", out.Code, out.Body)
	}
	if response := out.Header().Get("Content-Type"); response != "image/png" {
		t.Errorf("the picture came back as %q", response)
	}
	if responseVal := out.Header().Get("X-Content-Type-Options"); responseVal != "nosniff" {
		t.Errorf("the picture is sniffed: %q", responseVal)
	}
	if out.Body.String() != plate {
		t.Errorf("the picture is %d bytes", out.Body.Len())
	}
	if request := client.getAsked(); len(request) != 1 || request[0].Path != reflowed ||
		request[0].Size != mark.Size || stamp(request[0].ModTime) != mark.Mtime {
		t.Errorf("the plugin was asked at %v, want the bytes the address names", request)
	}
}

// The bytes of an entry are served from the window's own origin, so what is
// served is a picture and the bytes say so. What the manifest calls a file is
// the book's own claim about it.
func TestAnEntryThatIsNotAPictureIsNotServed(t *testing.T) {
	api, handler := readFrom(t, newBookClient())
	mark := printOf(t, api, reflowed)

	for _, entry := range []string{
		"OEBPS/first.xhtml",
		"OEBPS/pictures/cover.svg",
		"OEBPS/pictures/gone.png",
	} {
		t.Run(entry, func(t *testing.T) {
			out := ask(handler, pictureOf(reflowed, entry, mark))
			if out.Code == http.StatusOK {
				t.Errorf("%s was served as %q", entry, out.Header().Get("Content-Type"))
			}
		})
	}
}

func TestAnEntryTheArchiveLacksIsNotFound(t *testing.T) {
	api, handler := readFrom(t, newBookClient())
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/gone.png", mark)); out.Code != http.StatusNotFound {
		t.Errorf("the entry was answered %d, want not found", out.Code)
	}
}

func TestAFileThatIsNoBookIsUnsupported(t *testing.T) {
	client := newBookClient()
	client.failure = document.ErrNotADocument
	api, handler := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusUnsupportedMediaType {
		t.Errorf("the picture was answered %d, want unsupported media type", out.Code)
	}
}

// An address names the bytes it is about, and is answered while the file is
// still those bytes and no longer.
func TestAnAddressIntoABookThatChanged(t *testing.T) {
	client := newBookClient()
	api, handler := readFrom(t, client)
	stalePrint := printOf(t, api, reflowed)
	stalePrint.Size++

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", stalePrint)); out.Code != http.StatusNotFound {
		t.Errorf("the picture was answered %d, want not found", out.Code)
	}
	if client.countAsked() != 0 {
		t.Error("the plugin was asked for a file that is not the one the address names")
	}
}

// A file the plugin finds changed after the window looked is answered as one that
// changed.
func TestABookThePluginFindsChangedIsNotFound(t *testing.T) {
	client := newBookClient()
	client.failure = document.ErrChanged
	api, handler := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusNotFound {
		t.Errorf("the picture was answered %d, want not found", out.Code)
	}
}

// A path that leaves the vault fails, and so is one the vault holds
// nothing at.
func TestAPathTheVaultDoesNotHoldIsNoBook(t *testing.T) {
	client := newBookClient()
	api, handler := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	for _, path := range []string{"../outside.epub", "/etc/passwd", "library/nothing.epub"} {
		t.Run(path, func(t *testing.T) {
			if out := ask(handler, pictureOf(path, "OEBPS/pictures/plate.png", mark)); out.Code == http.StatusOK {
				t.Errorf("the picture of %s was answered", path)
			}
		})
	}
	if client.countAsked() != 0 {
		t.Error("the plugin was asked for a path the vault does not hold")
	}
}

// GetBook and ReadBookMarkup refuse a path that leaves the vault or that the
// vault does not hold.
func TestGetBookAndReadBookMarkupRejectPathsOutsideVault(t *testing.T) {
	client := newBookClient()
	api, _ := readFrom(t, client)

	for _, path := range []string{"../outside.epub", "/etc/passwd", "library/nothing.epub"} {
		t.Run(path, func(t *testing.T) {
			_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: path}))
			if err == nil {
				t.Errorf("GetBook(%q) succeeded, want error", path)
			}
			_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: path}))
			if err == nil {
				t.Errorf("ReadBookMarkup(%q) succeeded, want error", path)
			}
		})
	}
	if client.book != "" {
		t.Errorf("client was asked for book: %s", client.book)
	}
	if client.markup != nil {
		t.Errorf("client was asked for markup: %v", client.markup)
	}
}

// A window without a shown vault refuses GetBook and ReadBookMarkup.
func TestGetBookAndReadBookMarkupWithoutShownVault(t *testing.T) {
	client := newBookClient()
	api, _ := readFrom(t, client)
	api.state.setVault(domain.Vault{})

	_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Errorf("GetBook without vault = %v, want CodeFailedPrecondition", code)
	}
	_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Errorf("ReadBookMarkup without vault = %v, want CodeFailedPrecondition", code)
	}
}
