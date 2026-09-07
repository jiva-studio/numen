package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A document reaches the window as pictures: a page of a scan is pixels, and
// pixels are what a window asks a URL for. So a page is a handler, and what it
// answers with is a picture.

// widestPage is the most pixels a page is drawn across. The width is the
// window's, in the pixels of the device it is drawn on, and a page drawn wider
// than a screen is memory spent on pixels nobody sees.
const widestPage = 4096

// pointsDPI is one of the page's own units to the pixel, which is what a page's
// size is measured in.
const pointsDPI = 72

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
	// open holds a document open for drawing. It is pdf.Open in the
	// application, and a test puts its own in.
	open  func(raw []byte) (scan, error)
	docs  atomic.Pointer[documents]
	drawn atomic.Pointer[pictures]
	// kept is the same pages on disk, so a document opened again is not drawn
	// again. It is nothing where this machine names no cache folder.
	kept *cache

	// patience is how long a request waits for the document before it answers
	// that the document is busy. The library's own wait is minutes, which is
	// not an answer to a person turning a page.
	patience time.Duration

	ahead ahead
}

// ahead is the page drawn before it is asked for: the one slot that drawing
// runs in, and how long it has, being nobody's request.
type ahead struct {
	reading chan struct{}
	within  time.Duration
}

// take holds the slot for one drawing ahead, and answers false where one is
// already running: an ask being answered now comes first.
func (a *ahead) take() bool {
	select {
	case a.reading <- struct{}{}:
		return true
	default:
		return false
	}
}

// done gives the slot back.
func (a *ahead) done() { <-a.reading }

const (
	// patience is what a request waits for the document it is about.
	patience = 5 * time.Second
	// drawnAhead is what the page drawn before it is asked for has.
	drawnAhead = 30 * time.Second
)

// drawnBy holds a document open with what the application draws with.
func drawnBy(docs port.PageRenderer) func([]byte) (scan, error) {
	return func(raw []byte) (scan, error) {
		return docs.Draw(context.Background(), raw)
	}
}

// looking is a window with nothing open yet, and nothing kept on disk. What is
// kept there outlives the window, so where it goes is said where the window is
// served and not here.
func looking(docs port.PageRenderer) *viewer {
	v := &viewer{
		open:     drawnBy(docs),
		patience: patience,
		ahead:    ahead{reading: make(chan struct{}, 1), within: drawnAhead},
	}
	v.docs.Store(keeping())
	v.drawn.Store(drawings())
	return v
}

// close is the documents the window holds open let go, and the sweep of the
// folder they were kept in ended and waited for.
func (v *viewer) close() {
	v.docs.Load().close()
	v.kept.close()
}

// empty closes the documents the window has open and drops the pages drawn from
// them. It goes on looking, at whatever it is given next.
func (v *viewer) empty() {
	v.docs.Swap(keeping()).close()
	v.drawn.Store(drawings())
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

	reader, print, err := a.stat(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}
	doc, give, err := a.opening(ctx, reader, print)
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}
	defer give()

	if !doc.hold(ctx) {
		return nil, connect.NewError(connect.CodeUnavailable, errBusy)
	}
	defer doc.release()

	out := &v1.GetDocumentResponse{
		PageCount:   int32(doc.scan.Pages()),
		Fingerprint: &v1.Fingerprint{Path: print.path, Size: print.size, Mtime: print.mtime},
	}
	out.Pages = make([]*v1.PageSize, out.PageCount)
	for i := range out.Pages {
		width, height, err := doc.scan.Size(i)
		if err != nil {
			// A page whose size could not be read stands at nothing, and the
			// pages after it are still where they were.
			out.Pages[i] = &v1.PageSize{}
			continue
		}
		out.Pages[i] = &v1.PageSize{Width: width, Height: height}
	}
	return connect.NewResponse(out), nil
}

// Page answers with one page of a document, drawn as wide as was asked for.
//
// The address names the bytes it was drawn from, so it is answered only while
// the file at that path is still those bytes and the picture it answers with
// never changes. A file rewritten under the same name is a different address,
// and this one is gone.
func (a *API) Page(w http.ResponseWriter, r *http.Request, path, page string) {
	at, wide, named, err := wanted(page, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	reader, print, err := a.stat(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if named.size != print.size || named.mtime != print.mtime {
		refuse(w, errChanged)
		return
	}

	key := pictureID{document: print, page: at, width: wide}
	body, err := a.picture(ctx, reader, key)
	if err != nil {
		refuse(w, err)
		return
	}
	//nolint:contextcheck // the page drawn ahead is nobody's request, and runs under a.behind()
	a.readAhead(reader, key)

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
func (a *API) stat(ctx context.Context, path string) (port.VaultReader, fingerprint, error) {
	showing := a.Showing()
	if showing.ID == "" {
		return nil, fingerprint{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return nil, fingerprint{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return nil, fingerprint{}, err
	}
	return reader, fingerprint{path: ref.Path, size: ref.Size, mtime: stamp(ref.ModTime)}, nil
}

// opening hands over the document at a fingerprint, held open.
//
// The bytes are read where the document is not open already, and the read
// outlives the request that asked for it: a caller that ran out of patience is
// told the document is busy, and the document is there for the next ask.
func (a *API) opening(
	ctx context.Context,
	reader port.VaultReader,
	print fingerprint,
) (*document, func(), error) {
	return a.Viewer.docs.Load().take(ctx, print, func() (scan, error) {
		raw, err := reader.Read(context.WithoutCancel(ctx), print.path)
		if err != nil {
			return nil, err
		}
		return a.Viewer.open(raw)
	})
}

// picture is one page as the bytes that cross to the window: the one held in
// memory, or the one on disk, or the page drawn.
//
// Several asks for one page draw it once and are answered with the one drawing.
func (a *API) picture(ctx context.Context, reader port.VaultReader, key pictureID) ([]byte, error) {
	return a.Viewer.drawn.Load().draw(ctx, key, func() ([]byte, error) {
		if body := a.Viewer.kept.get(key); body != nil {
			return body, nil
		}
		body, err := a.drawing(ctx, reader, key)
		if err != nil {
			return nil, err
		}
		a.Viewer.kept.put(key, body)
		return body, nil
	})
}

// drawing is one page of a document, drawn and encoded.
func (a *API) drawing(ctx context.Context, reader port.VaultReader, key pictureID) ([]byte, error) {
	doc, give, err := a.opening(ctx, reader, key.document)
	if err != nil {
		return nil, err
	}
	defer give()

	if !doc.hold(ctx) {
		return nil, errBusy
	}
	defer doc.release()

	if key.page >= doc.scan.Pages() {
		return nil, fmt.Errorf("%w: page %d of %d", errNoPage, key.page, doc.scan.Pages())
	}
	drawn, err := doc.picture(key.page, key.width)
	if err != nil {
		return nil, err
	}
	return encoded(drawn)
}

// readAhead draws the page after this one, so that turning to it finds it
// drawn. One page is drawn ahead at a time, and an ask being answered now comes
// first.
func (a *API) readAhead(reader port.VaultReader, key pictureID) {
	next := pictureID{document: key.document, page: key.page + 1, width: key.width}
	if a.Viewer.drawn.Load().has(next) {
		return
	}
	if !a.Viewer.ahead.take() {
		return
	}
	go func() {
		defer a.Viewer.ahead.done()
		ctx, cancel := context.WithTimeout(a.behind(), a.Viewer.ahead.within)
		defer cancel()
		// Nobody asked for this page. One that would not draw is drawn again
		// when somebody turns to it, and says so then.
		_, _ = a.picture(ctx, reader, next)
	}()
}

// behind is what a drawing nobody asked for runs under: the context the passes
// behind the vault in the window run under. The vault going ends it, so a
// window that is closing is not held open by a page nobody has turned to. A
// window standing on no vault has no passes, and nothing to end.
func (a *API) behind() context.Context {
	if on := a.showing.Load(); on != nil {
		return on.under
	}
	return context.Background()
}

// picture is one page drawn as wide as was asked for. It is called with the
// document held.
//
// The library draws at a resolution, so the resolution is worked back from the
// width asked for and the page's own size, rounded up. The size is the
// document's own answer, and is asked once.
//
// It comes back a pixel or two wider than was asked for, and goes as it is. The
// window lays the page out at the width it asked for, so the browser takes those
// pixels off; resampling them off here is a pass over every pixel of the page to
// change nothing anybody sees.
func (d *document) picture(at, width int) (image.Image, error) {
	points, measured := d.points[at]
	if !measured {
		across, _, err := d.scan.Size(at)
		if err != nil {
			return nil, err
		}
		points = int(across)
		if points < 1 {
			return nil, fmt.Errorf("page %d has no width", at)
		}
		d.points[at] = points
	}
	dpi := (width*pointsDPI + points - 1) / points
	drawn, err := d.scan.Image(at, max(dpi, 1))
	if err != nil {
		return nil, err
	}
	return drawn, nil
}

// encoded is a drawn page as the bytes that cross to the window.
func encoded(drawn image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := jpeg.Encode(&out, drawn, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// wanted is which page the window asks for and how wide, in the pixels of the
// device it draws on.
func wanted(page string, query url.Values) (at, width int, named fingerprint, err error) {
	at, err = strconv.Atoi(page)
	if err != nil || at < 0 {
		return 0, 0, fingerprint{}, fmt.Errorf("%q is not a page", page)
	}
	width, err = strconv.Atoi(query.Get("wide"))
	if err != nil || width < 1 || width > widestPage {
		return 0, 0, fingerprint{}, fmt.Errorf("wide: %q is not a width", query.Get("wide"))
	}
	named, err = printed(query)
	if err != nil {
		return 0, 0, fingerprint{}, err
	}
	return at, width, named, nil
}

// refusedDrawing is the code a question about a document that could not be
// answered is refused under.
//
// A document another reader holds is unavailable and not a failure: the caller
// asks again. Everything else a caller can act on says which of its own doing
// it was.
func refusedDrawing(err error) connect.Code {
	switch {
	case errors.Is(err, errBusy):
		return connect.CodeUnavailable
	case errors.Is(err, errNoPage):
		return connect.CodeNotFound
	case errors.Is(err, port.ErrNotADocument), errors.Is(err, port.ErrEncrypted):
		return connect.CodeFailedPrecondition
	default:
		return reaching(err)
	}
}

// refuse says why a page is not coming.
func refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBusy):
		// The window is told when to ask again.
		w.Header().Set("Retry-After", "1")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, errNoPage), errors.Is(err, errChanged), port.NoNote(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, port.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, port.ErrNotADocument), errors.Is(err, port.ErrEncrypted):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// keepingDrawings is a window that keeps the pages it draws where this machine
// keeps what it can make again.
func keepingDrawings(docs port.PageRenderer) *viewer {
	v := looking(docs)
	v.kept = shelved()
	return v
}
