// Package pool holds what a window has a document or a book open on: the
// library worker a scan is read through, and the reader an EPUB is read
// through. Each is kept while it is wanted and let go when it is not.
package pool

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// ErrBusy is a document that could not be reached inside the bound: a worker of
// the library's pool is held elsewhere, or the document is answering somebody
// else. It is an answer and not a failure, and the window asks again.
var ErrBusy = errors.New("the document is busy")

// Document is one document open on a worker of the library's pool.
type Document struct {
	// lock is held for the length of one call into the library, which
	// answers one question about one document at a time. It is a channel
	// because a request waiting for it waits under a bound.
	lock chan struct{}

	// ready is closed once the document is open or the reason it is not is
	// known. Whoever asked for it while it was opening waits here.
	ready chan struct{}
	Scan  port.OpenDocument
	why   error

	// points is how wide each page is in its own units, measured when the page
	// is first drawn. It is read and written under drawing.
	points map[int]int

	uses int
	idle *time.Timer
}

// Lock takes the document for one call into the library, and answers false
// where the caller ran out of time.
func (d *Document) Lock(ctx context.Context) bool {
	select {
	case d.lock <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *Document) Unlock() { <-d.lock }

// Close closes the document once whatever is opening it has finished.
func (d *Document) Close() {
	select {
	case <-d.ready:
		if d.Scan != nil {
			d.Scan.Close()
		}
	default:
		go func() {
			<-d.ready
			if d.Scan != nil {
				d.Scan.Close()
			}
		}()
	}
}

// Documents manages an LRU pool of open documents.
type Documents struct {
	mu   sync.Mutex
	open map[document.Fingerprint]*Document
	// order tracks access order, least recently used first.
	order []document.Fingerprint
	// limit is the maximum number of concurrent open documents.
	// idleFor is the duration after which an unused document is closed.
	limit   int
	idleFor time.Duration

	// isClosing indicates the pool is shutting down.
	isClosing bool
	empty     chan struct{}
	closeOnce sync.Once
}

// Pool limits and idle timeout defaults.
const (
	MaxOpenDocuments    = 2
	DocumentIdleTimeout = 2 * time.Minute
)

// NewDocuments is a window with nothing open yet: how many documents it keeps
// open at once, and how long one nobody is looking at is kept.
func NewDocuments(limit int, idleFor time.Duration) *Documents {
	return &Documents{
		open:    map[document.Fingerprint]*Document{},
		limit:   limit,
		idleFor: idleFor,
		empty:   make(chan struct{}),
	}
}

// Acquire hands over the document at a fingerprint, opening it if it is not open.
// The function returned gives it back.
//
// Opening reads and parses the whole file and waits for a worker of the pool,
// so it happens on a goroutine of its own: a caller whose ctx ends first is told
// the document is busy, and the open goes on and is there for the next ask. Two
// callers asking at once open it once.
func (d *Documents) Acquire(
	ctx context.Context,
	mark document.Fingerprint,
	open func() (port.OpenDocument, error),
) (*Document, func(), error) {
	d.mu.Lock()
	if d.isClosing {
		d.mu.Unlock()
		return nil, nil, ErrBusy
	}
	doc, target := d.open[mark]
	if !target {
		doc = &Document{
			lock:   make(chan struct{}, 1),
			ready:  make(chan struct{}),
			points: map[int]int{},
		}
		d.open[mark] = doc
		go d.Load(mark, doc, open)
	}
	doc.uses++
	if doc.idle != nil {
		doc.idle.Stop()
		doc.idle = nil
	}
	d.Touch(mark)
	d.Evict()
	d.mu.Unlock()

	select {
	case <-doc.ready:
	case <-ctx.Done():
		d.Release(mark, doc)
		return nil, nil, ErrBusy
	}
	if doc.why != nil {
		d.Release(mark, doc)
		return nil, nil, doc.why
	}
	return doc, func() { d.Release(mark, doc) }, nil
}

// Use calls use with the document at a fingerprint, held for the length of the
// call. A caller who ran out of time waiting for the document or its worker is
// told ErrBusy.
func (d *Documents) Use(
	ctx context.Context,
	mark document.Fingerprint,
	open func() (port.OpenDocument, error),
	use func(document.Scan) error,
) error {
	doc, give, err := d.Acquire(ctx, mark, open)
	if err != nil {
		return err
	}
	defer give()

	if !doc.Lock(ctx) {
		return ErrBusy
	}
	defer doc.Unlock()
	return use(doc)
}

// Load opens the document and tells whoever is waiting. A document that could
// not be opened is forgotten before anyone is told, so the next ask reads the
// file again.
func (d *Documents) Load(mark document.Fingerprint, doc *Document, open func() (port.OpenDocument, error)) {
	doc.Scan, doc.why = open()
	if doc.why != nil {
		d.mu.Lock()
		if d.open[mark] == doc {
			delete(d.open, mark)
			d.RemoveFromOrder(mark)
		}
		d.mu.Unlock()
	}
	close(doc.ready)
}

// Release puts a document back and starts the clock on closing it.
func (d *Documents) Release(mark document.Fingerprint, doc *Document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	doc.uses--
	if doc.uses > 0 {
		return
	}
	if d.isClosing {
		if d.open[mark] == doc {
			delete(d.open, mark)
		}
		doc.Close()
		if len(d.open) == 0 {
			d.closeOnce.Do(func() { close(d.empty) })
		}
		return
	}
	if d.open[mark] != doc {
		doc.Close()
		return
	}
	doc.idle = time.AfterFunc(d.idleFor, func() { d.CloseIfIdle(mark, doc) })
}

// CloseIfIdle closes a document nobody has asked about for a while.
func (d *Documents) CloseIfIdle(mark document.Fingerprint, doc *Document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.open[mark] != doc || doc.uses > 0 {
		return
	}
	delete(d.open, mark)
	d.RemoveFromOrder(mark)
	doc.Close()
}

// Close closes all open documents when active usage completes.
func (d *Documents) Close() {
	d.mu.Lock()
	d.isClosing = true
	for print, doc := range d.open {
		if doc.idle != nil {
			doc.idle.Stop()
		}
		if doc.uses > 0 {
			continue
		}
		delete(d.open, print)
		doc.Close()
	}
	d.order = nil
	target := len(d.open)
	d.mu.Unlock()
	if target == 0 {
		return
	}
	<-d.empty
}

// Evict closes what is over the bound, oldest first. A document somebody is
// drawing from stays, and the bound is over until they are done with it.
func (d *Documents) Evict() {
	for len(d.open) > d.limit {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="any negative start reads as nothing found"
		oldest := -1
		for i, print := range d.order {
			if d.open[print].uses == 0 {
				oldest = i
				break
			}
		}
		if oldest < 0 {
			return
		}
		mark := d.order[oldest]
		doc := d.open[mark]
		if doc.idle != nil {
			doc.idle.Stop()
		}
		delete(d.open, mark)
		d.order = append(d.order[:oldest], d.order[oldest+1:]...)
		doc.Close()
	}
}

// Touch puts a fingerprint at the end of the order, which is the most recently
// asked for.
func (d *Documents) Touch(mark document.Fingerprint) {
	d.RemoveFromOrder(mark)
	d.order = append(d.order, mark)
}

func (d *Documents) RemoveFromOrder(mark document.Fingerprint) {
	for i, target := range d.order {
		if target == mark {
			d.order = append(d.order[:i], d.order[i+1:]...)
			return
		}
	}
}

func (d *Document) GetPageCount() int { return d.Scan.GetPageCount() }

func (d *Document) GetPageSize(index int) (width, height float64, err error) {
	return d.Scan.GetPageSize(index)
}

// PointsDPI is one of the page's own units to the pixel, which is what a
// page's size is measured in.
const PointsDPI = 72

// Render is one page drawn as wide as was asked for. It is called with the
// document held.
//
// The library draws at a resolution, so the resolution is worked back from the
// width asked for and the page's own size, rounded up. The size is the
// document's own answer, and is asked once.
//
// window lays the page out at the width it asked for, so the browser takes those
// pixels off; resampling them off here is a pass over every pixel of the page to
// change nothing anybody sees.
func (d *Document) Render(pos, width int) (image.Image, error) {
	points, measured := d.points[pos]
	if !measured {
		item, _, err := d.Scan.GetPageSize(pos)
		if err != nil {
			return nil, err
		}
		points = int(item)
		if points < 1 {
			return nil, fmt.Errorf("page %d has no width", pos)
		}
		d.points[pos] = points
	}
	dpi := (width*PointsDPI + points - 1) / points
	rendered, err := d.Scan.RenderPage(pos, max(dpi, 1))
	if err != nil {
		return nil, err
	}
	return rendered, nil
}
