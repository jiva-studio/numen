package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/pdf"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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

// errNoDrawing is what a build with nothing to draw a document with answers.
var errNoDrawing = errors.New("this build cannot draw a document")

// viewer holds what the window is looking at: the documents open and the pages
// already drawn.
type viewer struct {
	// open holds a document open for drawing. It is pdf.Open in the
	// application, and a test puts its own in.
	open  func(raw []byte) (drawable, error)
	docs  *documents
	drawn *pictures
	// kept is the same pages on disk, so a document opened again is not drawn
	// again. It is nothing where this machine names no cache folder.
	kept *shelf

	// patience is how long a request waits for the document before it answers
	// that the document is busy. The library's own wait is minutes, which is
	// not an answer to a person turning a page.
	patience time.Duration

	// reading is the one page drawn before it is asked for.
	reading chan struct{}
	// ahead is how long that drawing has, being nobody's request.
	ahead time.Duration
}

const (
	// patience is what a request waits for the document it is about.
	patience = 5 * time.Second
	// drawnAhead is what the page drawn before it is asked for has.
	drawnAhead = 30 * time.Second
)

// drawnByTheLibrary holds a document open with the library the application
// draws with.
func drawnByTheLibrary(raw []byte) (drawable, error) {
	scan, err := pdf.Open(raw)
	if err != nil {
		return nil, err
	}
	return scan, nil
}

// looking is a window with nothing open yet, and nothing kept on disk. What is
// kept there outlives the window, so where it goes is said where the window is
// served and not here.
func looking() *viewer {
	return &viewer{
		open:     drawnByTheLibrary,
		docs:     keeping(),
		drawn:    drawings(),
		patience: patience,
		reading:  make(chan struct{}, 1),
		ahead:    drawnAhead,
	}
}

func (v *viewer) close() { v.docs.close() }

// said is what the window is told a document is.
type said struct {
	Path  string `json:"path"`
	Pages int    `json:"pages"`
	// Sheets is how big each page is, in the page's own units. A window lays
	// out the pages it has not drawn yet, so it needs their shape before it has
	// their pixels, and a strip built on one guessed shape moves under the hand
	// as the real ones arrive.
	Sheets []sheet `json:"sheets"`
}

// A sheet is one page's size, in the page's own units.
type sheet struct {
	Wide float64 `json:"wide"`
	High float64 `json:"high"`
}

// Document answers what the document at a path in the vault is.
func (a *API) Document(w http.ResponseWriter, r *http.Request, path string) {
	if a.Viewer == nil || a.Readers == nil {
		refuse(w, errNoDrawing)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	reader, print, err := a.standing(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	doc, give, err := a.opening(ctx, reader, print)
	if err != nil {
		refuse(w, err)
		return
	}
	defer give()

	if !doc.hold(ctx) {
		refuse(w, errBusy)
		return
	}
	told := said{Path: print.path, Pages: doc.scan.Pages()}
	told.Sheets = make([]sheet, told.Pages)
	for i := range told.Sheets {
		wide, high, err := doc.scan.Size(i)
		if err != nil {
			continue
		}
		told.Sheets[i] = sheet{Wide: wide, High: high}
	}
	doc.release()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(told)
}

// Page answers with one page of a document, drawn as wide as was asked for.
func (a *API) Page(w http.ResponseWriter, r *http.Request, path, page string) {
	if a.Viewer == nil || a.Readers == nil {
		refuse(w, errNoDrawing)
		return
	}
	at, wide, err := wanted(page, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	reader, print, err := a.standing(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}

	key := shot{of: print, at: at, wide: wide}
	body, err := a.picture(ctx, reader, key)
	if err != nil {
		refuse(w, err)
		return
	}
	a.readAhead(reader, key)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.Write(body)
}

// standing is what the vault says about the file at a path: which bytes they
// are, for the caches to key on.
//
// The path goes through the vault's readers the way everything from outside
// does, so a path leaving the vault is refused there.
func (a *API) standing(ctx context.Context, path string) (port.VaultReader, fingerprint, error) {
	reader, err := a.Readers.Open(a.Vault)
	if err != nil {
		return nil, fingerprint{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return nil, fingerprint{}, err
	}
	return reader, fingerprint{path: ref.Path, size: ref.Size, mtime: ref.MTime}, nil
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
	return a.Viewer.docs.take(ctx, print, func() (drawable, error) {
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
func (a *API) picture(ctx context.Context, reader port.VaultReader, key shot) ([]byte, error) {
	return a.Viewer.drawn.draw(ctx, key, func() ([]byte, error) {
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
func (a *API) drawing(ctx context.Context, reader port.VaultReader, key shot) ([]byte, error) {
	doc, give, err := a.opening(ctx, reader, key.of)
	if err != nil {
		return nil, err
	}
	defer give()

	if !doc.hold(ctx) {
		return nil, errBusy
	}
	defer doc.release()

	if key.at >= doc.scan.Pages() {
		return nil, fmt.Errorf("%w: page %d of %d", errNoPage, key.at, doc.scan.Pages())
	}
	drawn, err := doc.picture(key.at, key.wide)
	if err != nil {
		return nil, err
	}
	return encoded(drawn)
}

// readAhead draws the page after this one, so that turning to it finds it
// drawn. One page is drawn ahead at a time, and an ask being answered now comes
// first.
func (a *API) readAhead(reader port.VaultReader, key shot) {
	next := shot{of: key.of, at: key.at + 1, wide: key.wide}
	if a.Viewer.drawn.has(next) {
		return
	}
	select {
	case a.Viewer.reading <- struct{}{}:
	default:
		return
	}
	go func() {
		defer func() { <-a.Viewer.reading }()
		ctx, cancel := context.WithTimeout(context.Background(), a.Viewer.ahead)
		defer cancel()
		a.picture(ctx, reader, next)
	}()
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
func (d *document) picture(at, wide int) (image.Image, error) {
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
	dpi := (wide*pointsDPI + points - 1) / points
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
func wanted(page string, query url.Values) (at, wide int, err error) {
	at, err = strconv.Atoi(page)
	if err != nil || at < 0 {
		return 0, 0, fmt.Errorf("%q is not a page", page)
	}
	wide, err = strconv.Atoi(query.Get("wide"))
	if err != nil || wide < 1 || wide > widestPage {
		return 0, 0, fmt.Errorf("wide: %q is not a width", query.Get("wide"))
	}
	return at, wide, nil
}

// refuse says why a document or a page is not coming.
func refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBusy):
		// The window is told when to ask again.
		w.Header().Set("Retry-After", "1")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, errNoPage), port.NoNote(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, filesystem.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, pdf.ErrNotPDF), errors.Is(err, pdf.ErrEncrypted):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, errNoDrawing):
		http.Error(w, err.Error(), http.StatusNotImplemented)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// keepingDrawings is a window that keeps the pages it draws where this machine
// keeps what it can make again.
func keepingDrawings() *viewer {
	v := looking()
	v.kept = shelved()
	return v
}
