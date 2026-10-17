package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pagecache"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// documentHandler answers what a document of the vault is, and draws its pages.
type documentHandler struct {
	state       *State
	viewer      *viewer
	getDocument *document.GetDocument
	getPage     *document.GetPage
}

// A document reaches the window as pictures: a page of a scan is pixels, and
// pixels are what a window asks a URL for. So a page is a handler, and what it
// answers with is a picture.

// widestPage is the most pixels a page is drawn across. The width is the
// window's, in the pixels of the device it is drawn on, and a page drawn wider
// than a screen is memory spent on pixels nobody sees.
const widestPage = 4096

// errNoDocuments is what a build that serves no documents answers.
var errNoDocuments = errors.New("this build serves no documents")

// viewer holds what the window is looking at: the documents open and the pages
// already drawn.
//
// What is open belongs to the vault it was opened in, and is emptied when
// another vault comes into the window.
type viewer struct {
	// open holds a document open for drawing. It is pdf.OpenReader in the
	// application, and a test puts its own in.
	open          func(file io.ReadSeekCloser) (port.OpenDocument, error)
	docs          atomic.Pointer[pool.Documents]
	renderedPages atomic.Pointer[pagecache.Memory]
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
func newDocumentOpener(docs port.PageRenderer) func(io.ReadSeekCloser) (port.OpenDocument, error) {
	return func(file io.ReadSeekCloser) (port.OpenDocument, error) {
		return docs.OpenDocument(context.Background(), file)
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
	return v
}

// newCachingViewer is a window that keeps the pages it draws where this machine
// keeps what it can make again.
func newCachingViewer(docs port.PageRenderer) *viewer {
	v := newViewer(docs)
	v.disk = pagecache.NewDisk()
	return v
}

// close is the documents the window holds open let go, and the sweep of the
// folder they were kept in ended and waited for.
func (v *viewer) close() {
	v.docs.Load().Close()
	v.disk.Close()
}

// reset closes the documents the window has open and drops the pages drawn. It
// goes on looking, at whatever it is given next.
func (v *viewer) reset() {
	v.docs.Swap(pool.NewDocuments(pool.MaxOpenDocuments, pool.DocumentIdleTimeout)).Close()
	v.renderedPages.Store(pagecache.NewMemory(pagecache.MaxMemoryBytes))
}

// UseDocument calls use with the document at a fingerprint, opened from its file
// where it is not open. The open outlives the request that asked for it: a
// caller that ran out of patience is told the document is busy, and the document
// is there for the next ask.
func (v *viewer) UseDocument(
	ctx context.Context,
	mark document.Fingerprint,
	file func(context.Context) (io.ReadSeekCloser, error),
	use func(document.Scan) error,
) error {
	return v.docs.Load().Use(ctx, mark, func() (port.OpenDocument, error) {
		opened, err := file(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		return v.open(opened)
	}, use)
}

// GetPage is one page as the bytes that cross to the window: the one held in
// memory, or the one on disk, or the page drawn.
//
// Several asks for one page draw it once and are answered with the one drawing.
func (v *viewer) GetPage(
	ctx context.Context, key document.PageKey, draw func() ([]byte, error),
) ([]byte, error) {
	return v.renderedPages.Load().GetOrRender(ctx, key, func() ([]byte, error) {
		if body := v.disk.Get(key); body != nil {
			return body, nil
		}
		body, err := draw()
		if err != nil {
			return nil, err
		}
		v.disk.Put(key, body)
		return body, nil
	})
}

// GetDocument answers what the document at a path in the vault is: how many
// pages it has, and how big each of them is in the page's own units.
//
// A window lays out the pages it has not drawn yet, so it needs their shape
// before it has their pixels: a strip built on one guessed shape moves under
// the hand as the real ones arrive.
func (h *documentHandler) GetDocument(
	ctx context.Context,
	r *connect.Request[v1.GetDocumentRequest],
) (*connect.Response[v1.GetDocumentResponse], error) {
	if h.getDocument == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
	}
	showing, err := h.state.getShownVault()
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	ctx, cancel := context.WithTimeout(ctx, h.viewer.patience)
	defer cancel()

	got, err := h.getDocument.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	out := &v1.GetDocumentResponse{Fingerprint: newWireFingerprint(got.Fingerprint)}
	out.Pages = make([]*v1.Page, len(got.Pages))
	for i, size := range got.Pages {
		out.Pages[i] = &v1.Page{Width: size[0], Height: size[1]}
	}
	return connect.NewResponse(out), nil
}

func (h *documentHandler) RenderPage(
	ctx context.Context,
	r *connect.Request[v1.RenderPageRequest],
) (*connect.Response[v1.RenderPageResponse], error) {
	if h.getPage == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
	}
	showing, err := h.state.getShownVault()
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	ctx, cancel := context.WithTimeout(ctx, h.viewer.patience)
	defer cancel()

	wide := int(r.Msg.GetWidth())
	if wide <= 0 {
		wide = 800
	}
	key := document.PageKey{
		Document: document.Fingerprint{Path: r.Msg.GetPath()},
		Page:     int(r.Msg.GetPage()),
		Width:    wide,
	}
	body, err := h.getPage.Execute(ctx, showing, key)
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	return connect.NewResponse(&v1.RenderPageResponse{
		ImageData:   body,
		ContentType: "image/jpeg",
		Width:       float64(wide),
		Height:      0,
	}), nil
}

func (h *documentHandler) ExtractText(
	_ context.Context,
	_ *connect.Request[v1.ExtractTextRequest],
) (*connect.Response[v1.ExtractTextResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
}

func (h *documentHandler) GetHighlights(
	_ context.Context,
	_ *connect.Request[v1.GetHighlightsRequest],
) (*connect.Response[v1.GetHighlightsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
}

// newWireFingerprint is a fingerprint as the schema carries it.
func newWireFingerprint(mark document.Fingerprint) *v1.Fingerprint {
	return &v1.Fingerprint{Path: mark.Path, Size: mark.Size, Mtime: mark.Mtime}
}

// pagesName is the one place a document has a name for: a page of it.
const pagesName = "pages"

// Page answers with one page of a document, drawn as wide as was asked for.
//
// The address names the bytes it was drawn from, so it is answered only while
// the file at that path is still those bytes and the picture it answers with
// never changes. A file rewritten under the same name is a different address,
// and this one is gone.
func (h *documentHandler) Page(w http.ResponseWriter, r *http.Request, path, where string) {
	at, wide, named, err := parsePageQuery(where, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.getPage == nil {
		http.Error(w, errNoDocuments.Error(), http.StatusNotImplemented)
		return
	}
	showing, err := h.state.getShownVault()
	if err != nil {
		refuse(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.viewer.patience)
	defer cancel()

	key := document.PageKey{
		Document: document.Fingerprint{Path: path, Size: named.Size, Mtime: named.Mtime},
		Page:     at,
		Width:    wide,
	}
	body, err := h.getPage.Execute(ctx, showing, key)
	if err != nil {
		refuse(w, err)
		return
	}
	//nolint:contextcheck // the page drawn ahead is nobody's request, and runs under h.state.getBackgroundContext()
	h.prefetchNextPage(showing, key)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", immutable)
	// The header is written; a body the client is no longer there to read is
	// nothing this can say anything more about.
	_, _ = w.Write(body)
}

// prefetchNextPage draws the page after this one, so that turning to it finds it
// drawn. One page is drawn ahead at a time, and an ask being answered now comes
// first.
func (h *documentHandler) prefetchNextPage(v domain.Vault, key document.PageKey) {
	next := document.PageKey{Document: key.Document, Page: key.Page + 1, Width: key.Width}
	if h.viewer.renderedPages.Load().Has(next) {
		return
	}
	if !h.viewer.prefetch.tryAcquire() {
		return
	}
	go func() {
		defer h.viewer.prefetch.release()
		ctx, cancel := context.WithTimeout(h.state.getBackgroundContext(), h.viewer.prefetch.timeout)
		defer cancel()
		// Nobody asked for this page. One that would not draw is drawn again
		// when somebody turns to it, and says so then.
		_, _ = h.getPage.Execute(ctx, v, next)
	}()
}

// parsePageQuery is which page the window asks for and how wide, in the pixels of the
// device it draws on. A document is asked for a page of it and no other place
// in it.
func parsePageQuery(where string, query url.Values) (at, width int, named document.Fingerprint, err error) {
	asked, found := strings.CutPrefix(where, pagesName+"/")
	if !found {
		return 0, 0, document.Fingerprint{}, fmt.Errorf("%q is not a place in a document", where)
	}
	at, err = strconv.Atoi(asked)
	if err != nil || at < 0 || at > math.MaxInt32 {
		return 0, 0, document.Fingerprint{}, fmt.Errorf("%q is not a page", asked)
	}
	width, err = strconv.Atoi(query.Get("wide"))
	if err != nil || width < 1 || width > widestPage {
		return 0, 0, document.Fingerprint{}, fmt.Errorf("wide: %q is not a width", query.Get("wide"))
	}
	named, err = parseFingerprint(query)
	if err != nil {
		return 0, 0, document.Fingerprint{}, err
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
	case errors.Is(err, document.ErrNoPage), errors.Is(err, document.ErrChanged),
		errors.Is(err, document.ErrNoEntry):
		return connect.CodeNotFound
	case errors.Is(err, document.ErrNotADocument), errors.Is(err, document.ErrEncrypted), isMisnamed(err):
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
	case errors.Is(err, document.ErrNoPage), errors.Is(err, document.ErrChanged), source.IsNoNote(err),
		errors.Is(err, document.ErrNoEntry):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, source.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, document.ErrNotADocument), errors.Is(err, document.ErrEncrypted), isMisnamed(err):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, errNoDocuments):
		http.Error(w, err.Error(), http.StatusNotImplemented)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
