package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pagecache"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
	"github.com/jiva-studio/numen/modules/libs/core/internal/docfile"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A document reaches the window as pictures: a page of a scan is pixels, and
// pixels are what a window asks a URL for. So a page is a handler, and what it
// answers with is a picture.

// widestPage is the most pixels a page is drawn across. The width is the
// window's, in the pixels of the device it is drawn on, and a page drawn wider
// than a screen is memory spent on pixels nobody sees.
const widestPage = 4096

// quality is what a drawn page is encoded at. A page of a scan is a photograph
// of paper — continuous tone, no flat colour and nothing behind it — which is
// what JPEG carries, and at this quality what it loses is under the grain of
// the paper.
const quality = 82

// errNoPage is a page the document does not have.
var errNoPage = errors.New("no such page in this document")

// viewer holds what the window is looking at: the documents open and the pages
// already drawn.
//
// What is open belongs to the vault it was opened in, and is emptied when
// another vault comes into the window.
type viewer struct {
	// open holds a document open for drawing. It is pdf.OpenReader in the
	// application, and a test puts its own in.
	open          func(file io.ReadSeekCloser) (pool.Scan, error)
	docs          atomic.Pointer[pool.Documents]
	renderedPages atomic.Pointer[pagecache.Memory]
	// books holds the books open. A book is an archive unpacked and parsed, and
	// what it costs is memory.
	books atomic.Pointer[pool.Books]
	// disk is the same pages on disk, so a document opened again is not drawn
	// again. It is nothing where this machine names no cache folder.
	disk *pagecache.Disk

	// patience is how long a request waits for the document before it answers
	// that the document is busy. The library's own wait is minutes, which is
	// not an answer to a person turning a page.
	patience time.Duration

	prefetch prefetchSlot
}

// prefetchSlot is the page drawn before it is asked for: the one slot that drawing
// runs in, and how long it has, being nobody's request.
type prefetchSlot struct {
	slot    chan struct{}
	timeout time.Duration
}

// tryAcquire holds the slot for one drawing ahead, and answers false where one is
// already running: an ask being answered now comes first.
func (a *prefetchSlot) tryAcquire() bool {
	select {
	case a.slot <- struct{}{}:
		return true
	default:
		return false
	}
}

// release gives the slot back.
func (a *prefetchSlot) release() { <-a.slot }

const (
	// openTimeout is what a request waits for the document it is about.
	openTimeout = 5 * time.Second
	// prefetchTimeout is what the page drawn before it is asked for has.
	prefetchTimeout = 30 * time.Second
)

// newDocumentOpener holds a document open with what the application draws with.
func newDocumentOpener(docs port.PageRenderer) func(io.ReadSeekCloser) (pool.Scan, error) {
	return func(file io.ReadSeekCloser) (pool.Scan, error) {
		return docs.Draw(context.Background(), file)
	}
}

// newViewer is a window with nothing open yet, and nothing kept on disk. What is
// kept there outlives the window, so where it goes is said where the window is
// served and not here.
func newViewer(docs port.PageRenderer) *viewer {
	v := &viewer{
		open:     newDocumentOpener(docs),
		patience: openTimeout,
		prefetch: prefetchSlot{slot: make(chan struct{}, 1), timeout: prefetchTimeout},
	}
	v.docs.Store(pool.NewDocuments(pool.MaxOpenDocuments, pool.DocumentIdleTimeout))
	v.renderedPages.Store(pagecache.NewMemory(pagecache.MaxMemoryBytes))
	v.books.Store(pool.NewBooks(pool.MaxOpenBooks, pool.BookIdleTimeout))
	return v
}

// close is the documents the window holds open let go, and the sweep of the
// folder they were kept in ended and waited for.
func (v *viewer) close() {
	v.docs.Load().Close()
	v.books.Load().Close()
	v.disk.Close()
}

// reset closes the documents the window has open, lets go of the books, and
// drops the pages drawn. It goes on looking, at whatever it is given next.
func (v *viewer) reset() {
	v.docs.Swap(pool.NewDocuments(pool.MaxOpenDocuments, pool.DocumentIdleTimeout)).Close()
	v.books.Swap(pool.NewBooks(pool.MaxOpenBooks, pool.BookIdleTimeout)).Close()
	v.renderedPages.Store(pagecache.NewMemory(pagecache.MaxMemoryBytes))
}

// GetDocument answers what the document at a path in the vault is: how many
// pages it has, and how big each of them is in the page's own units.
//
// A window lays out the pages it has not drawn yet, so it needs their shape
// before it has their pixels: a strip built on one guessed shape moves under
// the hand as the real ones arrive.
func (a *API) GetDocument(
	ctx context.Context,
	r *connect.Request[v1.GetDocumentRequest],
) (*connect.Response[v1.GetDocumentResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, a.Viewer.patience)
	defer cancel()

	reader, mark, err := a.stat(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	doc, give, err := a.openDocument(ctx, reader, mark)
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	defer give()

	if !doc.Lock(ctx) {
		return nil, connect.NewError(connect.CodeUnavailable, pool.ErrBusy)
	}
	defer doc.Unlock()

	out := &v1.GetDocumentResponse{
		Fingerprint: &v1.Fingerprint{Path: mark.Path, Size: mark.Size, Mtime: mark.Mtime},
	}
	out.Pages = make([]*v1.Page, doc.Scan.Pages())
	for i := range out.Pages {
		width, height, err := doc.Scan.Size(i)
		if err != nil {
			// A page whose size could not be read stands at nothing, and the
			// pages after it are still where they were.
			out.Pages[i] = &v1.Page{}
			continue
		}
		out.Pages[i] = &v1.Page{Width: width, Height: height}
	}
	return connect.NewResponse(out), nil
}

// pagesName is the one place a document has a name for: a page of it.
const pagesName = "pages"

// Page answers with one page of a document, drawn as wide as was asked for.
//
// The address names the bytes it was drawn from, so it is answered only while
// the file at that path is still those bytes and the picture it answers with
// never changes. A file rewritten under the same name is a different address,
// and this one is gone.
func (a *API) Page(w http.ResponseWriter, r *http.Request, path, where string) {
	at, wide, named, err := parsePageQuery(where, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	reader, mark, err := a.stat(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if named.Size != mark.Size || named.Mtime != mark.Mtime {
		refuse(w, errChanged)
		return
	}

	key := pagecache.ID{Document: mark, Page: at, Width: wide}
	body, err := a.getPage(ctx, reader, key)
	if err != nil {
		refuse(w, err)
		return
	}
	//nolint:contextcheck // the page drawn ahead is nobody's request, and runs under a.getBackgroundContext()
	a.prefetchNextPage(reader, key)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", immutable)
	// The header is written; a body the client is no longer there to read is
	// nothing this can say anything more about.
	_, _ = w.Write(body)
}

// stat is what the vault says about the file at a path: which bytes they are,
// for the caches to key on.
//
// The path goes through the vault's readers the way everything from outside
// does, so a path leaving the vault is refused there. A window standing on
// nothing holds no file to say anything about.
func (a *API) stat(ctx context.Context, path string) (port.VaultReader, pool.Fingerprint, error) {
	showing := a.GetCurrentVault()
	if showing.ID == "" {
		return nil, pool.Fingerprint{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return nil, pool.Fingerprint{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return nil, pool.Fingerprint{}, err
	}
	return reader, pool.Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}, nil
}

// openDocument hands over the document at a fingerprint, held open.
//
// The file is opened where the document is not open already, and the open
// outlives the request that asked for it: a caller that ran out of patience is
// told the document is busy, and the document is there for the next ask.
func (a *API) openDocument(
	ctx context.Context,
	reader port.VaultReader,
	mark pool.Fingerprint,
) (*pool.Document, func(), error) {
	return a.Viewer.docs.Load().Acquire(ctx, mark, func() (pool.Scan, error) {
		file, err := docfile.Open(context.WithoutCancel(ctx), reader, mark.Path)
		if err != nil {
			return nil, err
		}
		return a.Viewer.open(file)
	})
}

// getPage is one page as the bytes that cross to the window: the one held in
// memory, or the one on disk, or the page drawn.
//
// Several asks for one page draw it once and are answered with the one drawing.
func (a *API) getPage(ctx context.Context, reader port.VaultReader, key pagecache.ID) ([]byte, error) {
	return a.Viewer.renderedPages.Load().GetOrRender(ctx, key, func() ([]byte, error) {
		if body := a.Viewer.disk.Get(key); body != nil {
			return body, nil
		}
		body, err := a.renderPage(ctx, reader, key)
		if err != nil {
			return nil, err
		}
		a.Viewer.disk.Put(key, body)
		return body, nil
	})
}

// renderPage is one page of a document, drawn and encoded.
func (a *API) renderPage(ctx context.Context, reader port.VaultReader, key pagecache.ID) ([]byte, error) {
	doc, give, err := a.openDocument(ctx, reader, key.Document)
	if err != nil {
		return nil, err
	}
	defer give()

	if !doc.Lock(ctx) {
		return nil, pool.ErrBusy
	}
	defer doc.Unlock()

	if key.Page >= doc.Scan.Pages() {
		return nil, fmt.Errorf("%w: page %d of %d", errNoPage, key.Page, doc.Scan.Pages())
	}
	drawn, err := doc.Render(key.Page, key.Width)
	if err != nil {
		return nil, err
	}
	return encodePage(drawn)
}

// prefetchNextPage draws the page after this one, so that turning to it finds it
// drawn. One page is drawn ahead at a time, and an ask being answered now comes
// first.
func (a *API) prefetchNextPage(reader port.VaultReader, key pagecache.ID) {
	next := pagecache.ID{Document: key.Document, Page: key.Page + 1, Width: key.Width}
	if a.Viewer.renderedPages.Load().Has(next) {
		return
	}
	if !a.Viewer.prefetch.tryAcquire() {
		return
	}
	go func() {
		defer a.Viewer.prefetch.release()
		ctx, cancel := context.WithTimeout(a.getBackgroundContext(), a.Viewer.prefetch.timeout)
		defer cancel()
		// Nobody asked for this page. One that would not draw is drawn again
		// when somebody turns to it, and says so then.
		_, _ = a.getPage(ctx, reader, next)
	}()
}

// getBackgroundContext is what a drawing nobody asked for runs under: the
// context the passes behind the vault in the window run under. The vault going
// ends it, so a window that is closing is not held open by a page nobody has
// turned to. A window standing on no vault has no passes, and nothing to end.
func (a *API) getBackgroundContext() context.Context {
	if on := a.showing.Load(); on != nil {
		return on.under
	}
	return context.Background()
}

// encodePage is a drawn page as the bytes that cross to the window.
func encodePage(drawn image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := jpeg.Encode(&out, drawn, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// parsePageQuery is which page the window asks for and how wide, in the pixels of the
// device it draws on. A document is asked for a page of it and no other place
// in it.
func parsePageQuery(where string, query url.Values) (at, width int, named pool.Fingerprint, err error) {
	asked, found := strings.CutPrefix(where, pagesName+"/")
	if !found {
		return 0, 0, pool.Fingerprint{}, fmt.Errorf("%q is not a place in a document", where)
	}
	at, err = strconv.Atoi(asked)
	if err != nil || at < 0 {
		return 0, 0, pool.Fingerprint{}, fmt.Errorf("%q is not a page", asked)
	}
	width, err = strconv.Atoi(query.Get("wide"))
	if err != nil || width < 1 || width > widestPage {
		return 0, 0, pool.Fingerprint{}, fmt.Errorf("wide: %q is not a width", query.Get("wide"))
	}
	named, err = parseFingerprint(query)
	if err != nil {
		return 0, 0, pool.Fingerprint{}, err
	}
	return at, width, named, nil
}

// getDrawCode is the code a question about a document that could not be
// answered is refused under.
//
// A document another reader holds is unavailable and not a failure: the caller
// asks again. Everything else a caller can act on says which of its own doing
// it was.
func getDrawCode(err error) connect.Code {
	switch {
	case errors.Is(err, pool.ErrBusy):
		return connect.CodeUnavailable
	case errors.Is(err, errNoPage), errors.Is(err, errChanged),
		errors.Is(err, epub.ErrNoDocument), errors.Is(err, epub.ErrNoEntry):
		return connect.CodeNotFound
	case errors.Is(err, port.ErrNotADocument), errors.Is(err, port.ErrEncrypted), isMisnamed(err):
		return connect.CodeFailedPrecondition
	default:
		return getReachCode(err)
	}
}

// refuse says why a page is not coming.
func refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pool.ErrBusy):
		// The window is told when to ask again.
		w.Header().Set("Retry-After", "1")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, errNoPage), errors.Is(err, errChanged), port.IsNoNote(err),
		errors.Is(err, epub.ErrNoDocument), errors.Is(err, epub.ErrNoEntry):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, port.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, port.ErrNotADocument), errors.Is(err, port.ErrEncrypted), isMisnamed(err):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// newCachingViewer is a window that keeps the pages it draws where this machine
// keeps what it can make again.
func newCachingViewer(docs port.PageRenderer) *viewer {
	v := newViewer(docs)
	v.disk = pagecache.NewDisk()
	return v
}
