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
)

// Scan is a document held open for its pages to be drawn. It is pdf.Scan in
// the application, and a test puts its own in.
type Scan interface {
	Pages() int
	Size(index int) (wide, high float64, err error)
	Image(index, dpi int) (image.Image, error)
	Close()
}

// Fingerprint is which bytes a document is: where it sits in the vault, and
// what the vault says about the file there. Both caches key on it, so a
// document rewritten under the same name is drawn again. The stamp is
// nanoseconds since the epoch: this is a cache key, and a time.Time compares by
// zone and monotonic reading as well as by instant.
type Fingerprint struct {
	Path  string
	Size  int64
	Mtime int64
}

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
	Scan  Scan
	why   error

	// points is how wide each page is in its own units, measured when the page
	// is first drawn. It is read and written under drawing.
	points map[int]int

	uses int
	idle *time.Timer
}

// Hold takes the document for one call into the library, and answers false
// where the caller ran out of time.
func (d *Document) Hold(ctx context.Context) bool {
	select {
	case d.lock <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *Document) Release() { <-d.lock }

// Shut closes the document once whatever is opening it has finished.
func (d *Document) Shut() {
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

// Documents are the documents held open. A file is read and parsed once,
// however many of its pages are turned.
//
// Few are held: an open document keeps one of the library's workers for as long
// as it is open, and a recognition keeps another for as long as it reads. One
// nobody has looked at for a while is closed.
type Documents struct {
	mu   sync.Mutex
	open map[Fingerprint]*Document
	// order is what is held, least recently asked for first.
	order []Fingerprint
	// limit is how many are held open at once, and idleFor how long one nobody
	// is looking at is kept.
	limit   int
	idleFor time.Duration

	// isClosing is the window going. A document being drawn from is left to the
	// reader holding it, and empty is closed when the last of them gives it
	// back.
	isClosing bool
	empty     chan struct{}
	done      sync.Once
}

// How many documents are held open at once, and how long one nobody is looking
// at is kept.
const (
	MostOpen    = 2
	OpenIdleFor = 2 * time.Minute
)

// NewDocuments is a window with nothing open yet: how many documents it keeps
// open at once, and how long one nobody is looking at is kept.
func NewDocuments(limit int, idleFor time.Duration) *Documents {
	return &Documents{
		open:    map[Fingerprint]*Document{},
		limit:   limit,
		idleFor: idleFor,
		empty:   make(chan struct{}),
	}
}

// Take hands over the document at a fingerprint, opening it if it is not open.
// The function returned gives it back.
//
// Opening reads and parses the whole file and waits for a worker of the pool,
// so it happens on a goroutine of its own: a caller whose ctx ends first is told
// the document is busy, and the open goes on and is there for the next ask. Two
// callers asking at once open it once.
func (d *Documents) Take(
	ctx context.Context,
	mark Fingerprint,
	open func() (Scan, error),
) (*Document, func(), error) {
	d.mu.Lock()
	if d.isClosing {
		d.mu.Unlock()
		return nil, nil, ErrBusy
	}
	doc, held := d.open[mark]
	if !held {
		doc = &Document{
			lock:   make(chan struct{}, 1),
			ready:  make(chan struct{}),
			points: map[int]int{},
		}
		d.open[mark] = doc
		go d.Fill(mark, doc, open)
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
		d.Give(mark, doc)
		return nil, nil, ErrBusy
	}
	if doc.why != nil {
		d.Give(mark, doc)
		return nil, nil, doc.why
	}
	return doc, func() { d.Give(mark, doc) }, nil
}

// Fill opens the document and tells whoever is waiting.
func (d *Documents) Fill(mark Fingerprint, doc *Document, open func() (Scan, error)) {
	doc.Scan, doc.why = open()
	close(doc.ready)
	if doc.why == nil {
		return
	}
	// A document that could not be opened is not kept, so the next ask reads
	// the file again.
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.open[mark] == doc {
		delete(d.open, mark)
		d.Forget(mark)
	}
}

// Give puts a document back and starts the clock on closing it.
func (d *Documents) Give(mark Fingerprint, doc *Document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	doc.uses--
	if doc.uses > 0 {
		return
	}
	if d.isClosing {
		delete(d.open, mark)
		doc.Shut()
		if len(d.open) == 0 {
			d.done.Do(func() { close(d.empty) })
		}
		return
	}
	if d.open[mark] != doc {
		doc.Shut()
		return
	}
	doc.idle = time.AfterFunc(d.idleFor, func() { d.Retire(mark, doc) })
}

// Retire closes a document nobody has asked about for a while.
func (d *Documents) Retire(mark Fingerprint, doc *Document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.open[mark] != doc || doc.uses > 0 {
		return
	}
	delete(d.open, mark)
	d.Forget(mark)
	doc.Shut()
}

// Close closes every document held open, once nobody is drawing from one.
//
// Closing gives the library's worker back, and a worker given back while it is
// drawing is taken from under the drawing. What is left to close is what the
// last reader gives back.
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
		doc.Shut()
	}
	d.order = nil
	held := len(d.open)
	d.mu.Unlock()
	if held == 0 {
		return
	}
	<-d.empty
}

// Evict closes what is over the bound, oldest first. A document somebody is
// drawing from stays, and the bound is over until they are done with it.
func (d *Documents) Evict() {
	for len(d.open) > d.limit {
		oldest := -1
		for i, print := range d.order {
			if doc := d.open[print]; doc != nil && doc.uses == 0 {
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
		doc.Shut()
	}
}

// Touch puts a fingerprint at the end of the order, which is the most recently
// asked for.
func (d *Documents) Touch(mark Fingerprint) {
	d.Forget(mark)
	d.order = append(d.order, mark)
}

func (d *Documents) Forget(mark Fingerprint) {
	for i, held := range d.order {
		if held == mark {
			d.order = append(d.order[:i], d.order[i+1:]...)
			return
		}
	}
}

// PointsDPI is one of the page's own units to the pixel, which is what a
// page's size is measured in.
const PointsDPI = 72

// Draw is one page drawn as wide as was asked for. It is called with the
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
func (d *Document) Draw(at, width int) (image.Image, error) {
	points, measured := d.points[at]
	if !measured {
		across, _, err := d.Scan.Size(at)
		if err != nil {
			return nil, err
		}
		points = int(across)
		if points < 1 {
			return nil, fmt.Errorf("page %d has no width", at)
		}
		d.points[at] = points
	}
	dpi := (width*PointsDPI + points - 1) / points
	drawn, err := d.Scan.Image(at, max(dpi, 1))
	if err != nil {
		return nil, err
	}
	return drawn, nil
}
