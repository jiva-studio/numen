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

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
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
	asked   []*v1.Fingerprint
	markup  *v1.ReadBookMarkupRequest
	book    *v1.GetBookRequest
}

func (c *fakeBookClient) GetBook(
	_ context.Context, r *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.book = r.Msg
	if c.failure != nil {
		return nil, c.failure
	}
	return connect.NewResponse(&v1.GetBookResponse{Title: "A Reflowed Book"}), nil
}

func (c *fakeBookClient) ReadBookMarkup(
	_ context.Context, r *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.markup = r.Msg
	return connect.NewResponse(&v1.ReadBookMarkupResponse{Markup: "<p>Alpha omega.</p>"}), nil
}

func (c *fakeBookClient) ReadEntry(_ context.Context, baseline *v1.Fingerprint, entry string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, baseline)
	if c.failure != nil {
		return nil, c.failure
	}
	body, held := c.entries[entry]
	if !held {
		return nil, document.ErrNoEntry
	}
	return body, nil
}

func (c *fakeBookClient) countAsked() int {
	return len(c.getAsked())
}

// getAsked is a copy of the fingerprints the plugin was asked at.
func (c *fakeBookClient) getAsked() []*v1.Fingerprint {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.asked)
}

// readFrom is a window looking at one book of its vault, which the plugin it is
// given holds the entries of.
func readFrom(t *testing.T, client *fakeBookClient) (*Services, http.Handler, *int) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{reflowed: "the bytes of a book"})
	api := newViewingAPI(filesystem.VaultReaders{}, newViewer(testsupport.Documents{}))
	api.state.setVault(vault)
	closed := new(int)
	api.Book.openClient = func(context.Context) (any, func(), error) {
		return client, func() { *closed++ }, nil
	}
	t.Cleanup(api.Document.viewer.close)
	return api, api.NewHandler(http.NotFoundHandler()), closed
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
	for at, one := range parts {
		parts[at] = url.PathEscape(one)
	}
	return fmt.Sprintf("%s/%s?%s", assetURL(path), strings.Join(parts, "/"), formatFingerprint(mark))
}

// What a book is comes from the plugin, and is answered as the plugin said it.
func TestWhatABookIsIsAskedOfThePlugin(t *testing.T) {
	client := newBookClient()
	api, _, closed := readFrom(t, client)

	out, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetTitle() != "A Reflowed Book" || client.book.GetPath() != reflowed {
		t.Errorf("the book is %q, asked for %q", out.Msg.GetTitle(), client.book.GetPath())
	}
	if *closed != 1 {
		t.Errorf("the plugin was let go %d times, want once", *closed)
	}
}

// A refusal of the plugin reaches the window with the code the plugin gave it.
func TestARefusalOfThePluginIsPassedOn(t *testing.T) {
	client := newBookClient()
	client.failure = connect.NewError(connect.CodeFailedPrecondition, errors.New("not a book"))
	api, _, _ := readFrom(t, client)

	_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("the book was refused %v, want failed precondition", got)
	}
}

func TestOneDocumentOfABookIsAskedOfThePlugin(t *testing.T) {
	client := newBookClient()
	api, _, closed := readFrom(t, client)

	out, err := api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{
		Path: reflowed, Document: "OEBPS/first.xhtml",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetMarkup() != "<p>Alpha omega.</p>" || client.markup.GetDocument() != "OEBPS/first.xhtml" {
		t.Errorf("the markup is %q, asked for %q", out.Msg.GetMarkup(), client.markup.GetDocument())
	}
	if *closed != 1 {
		t.Errorf("the plugin was let go %d times, want once", *closed)
	}
}

// Without a plugin that reads books, nothing about a book is answered.
func TestABookIsNotAnsweredWithoutAPlugin(t *testing.T) {
	for name, open := range map[string]func(context.Context) (any, func(), error){
		"no client is opened":      nil,
		"the client is no reader":  func(context.Context) (any, func(), error) { return struct{}{}, nil, nil },
		"the plugin does not open": func(context.Context) (any, func(), error) { return nil, nil, errors.New("gone") },
	} {
		t.Run(name, func(t *testing.T) {
			api, handler, _ := readFrom(t, newBookClient())
			api.Book.openClient = open
			mark := printOf(t, api, reflowed)

			_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
			if err == nil {
				t.Error("a book was answered")
			}
			_, err = api.Book.ReadBookMarkup(t.Context(), connect.NewRequest(&v1.ReadBookMarkupRequest{Path: reflowed}))
			if err == nil {
				t.Error("a document of a book was answered")
			}
			if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code == http.StatusOK {
				t.Error("a picture was served")
			}
		})
	}
}

// A book plugin that is not there, for any reason, is a build that cannot read
// books, and every ask of a book says so with the one code.
func TestAnAbsentBookPluginIsAnsweredAsUnimplemented(t *testing.T) {
	for name, why := range map[string]error{
		"not installed": &port.UnavailableError{Capability: "book", Reason: port.PluginNotInstalled},
		"disabled":      &port.UnavailableError{Capability: "book", Reason: port.PluginDisabled},
		"failed":        &port.UnavailableError{PluginID: "md.numen.epub", Reason: port.PluginFailed, Err: errors.New("exit 1")},
		"no program": &port.UnavailableError{
			PluginID: "md.numen.epub", Reason: port.PluginFailed, Err: fmt.Errorf("start bin/epub: %w", fs.ErrNotExist),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api, handler, _ := readFrom(t, newBookClient())
			api.Book.openClient = func(context.Context) (any, func(), error) { return nil, nil, why }
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

// A plugin that goes away once its client is open is answered the same way,
// a code it gives is kept, and an uncoded failure is internal.
func TestABookPluginThatGoesAwayIsAnsweredAsUnimplemented(t *testing.T) {
	for why, want := range map[error]connect.Code{
		&port.UnavailableError{PluginID: "md.numen.epub", Reason: port.PluginFailed}: connect.CodeUnimplemented,
		connect.NewError(connect.CodeNotFound, errors.New("no such book")):           connect.CodeNotFound,
		errors.New("the plugin broke"):                                               connect.CodeInternal,
	} {
		client := newBookClient()
		client.failure = why
		api, _, _ := readFrom(t, client)

		_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
		if code := connect.CodeOf(err); code != want {
			t.Errorf("%v was answered %v, want %v", why, code, want)
		}
	}
}

// A failure to open the plugin that carries a code, or ends the caller's
// context, is answered with that code.
func TestAFailureToOpenTheBookPluginKeepsItsCode(t *testing.T) {
	for why, want := range map[error]connect.Code{
		connect.NewError(connect.CodePermissionDenied, errors.New("no")): connect.CodePermissionDenied,
		context.DeadlineExceeded: connect.CodeDeadlineExceeded,
	} {
		api, _, _ := readFrom(t, newBookClient())
		api.Book.openClient = func(context.Context) (any, func(), error) { return nil, nil, why }

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

// A client that is no book reader is no book plugin, and the client is let go.
func TestAClientThatReadsNoBooksIsAnsweredAsUnimplemented(t *testing.T) {
	api, _, _ := readFrom(t, newBookClient())
	closed := 0
	api.Book.openClient = func(context.Context) (any, func(), error) { return struct{}{}, func() { closed++ }, nil }

	_, err := api.Book.GetBook(t.Context(), connect.NewRequest(&v1.GetBookRequest{Path: reflowed}))
	if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
		t.Errorf("a book was answered %v, want unimplemented", code)
	}
	if closed != 1 {
		t.Errorf("the client was let go %d times, want once", closed)
	}
}

func TestAPictureIsNotServedWithoutAPluginThatReadsBooks(t *testing.T) {
	api, handler, _ := readFrom(t, newBookClient())
	api.Book.openClient = nil
	mark := printOf(t, api, reflowed)

	out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark))
	if out.Code != http.StatusNotImplemented {
		t.Errorf("the picture was answered %d, want not implemented", out.Code)
	}
}

// A picture the book carries is drawn out of the archive the plugin reads, and
// what settles the type is the bytes.
func TestAPictureTheBookCarries(t *testing.T) {
	client := newBookClient()
	api, handler, closed := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark))
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
	if asked := client.getAsked(); len(asked) != 1 || asked[0].GetPath() != reflowed ||
		asked[0].GetSize() != mark.Size || asked[0].GetMtime() != mark.Mtime {
		t.Errorf("the plugin was asked at %v, want the bytes the address names", asked)
	}
	if *closed != 1 {
		t.Errorf("the plugin was let go %d times, want once", *closed)
	}
}

// The bytes of an entry are served from the window's own origin, so what is
// served is a picture and the bytes say so. What the manifest calls a file is
// the book's own claim about it.
func TestAnEntryThatIsNotAPictureIsNotServed(t *testing.T) {
	api, handler, _ := readFrom(t, newBookClient())
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
	api, handler, _ := readFrom(t, newBookClient())
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/gone.png", mark)); out.Code != http.StatusNotFound {
		t.Errorf("the entry was answered %d, want not found", out.Code)
	}
}

func TestAFileThatIsNoBookIsUnsupported(t *testing.T) {
	client := newBookClient()
	client.failure = document.ErrNotADocument
	api, handler, _ := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusUnsupportedMediaType {
		t.Errorf("the picture was answered %d, want unsupported media type", out.Code)
	}
}

// An address names the bytes it is about, and is answered while the file is
// still those bytes and no longer.
func TestAnAddressIntoABookThatChanged(t *testing.T) {
	client := newBookClient()
	api, handler, _ := readFrom(t, client)
	stale := printOf(t, api, reflowed)
	stale.Size++

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", stale)); out.Code != http.StatusNotFound {
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
	api, handler, _ := readFrom(t, client)
	mark := printOf(t, api, reflowed)

	if out := ask(handler, pictureOf(reflowed, "OEBPS/pictures/plate.png", mark)); out.Code != http.StatusNotFound {
		t.Errorf("the picture was answered %d, want not found", out.Code)
	}
}

// A path that leaves the vault is refused, and so is one the vault holds
// nothing at.
func TestAPathTheVaultDoesNotHoldIsNoBook(t *testing.T) {
	client := newBookClient()
	api, handler, _ := readFrom(t, client)
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
