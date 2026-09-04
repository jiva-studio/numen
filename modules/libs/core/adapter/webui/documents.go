package webui

import (
	"context"
	"errors"
	"image"
	"sync"
	"time"
)

// scan is a document held open for its pages to be drawn. It is pdf.Scan in
// the application, and a test puts its own in.
type scan interface {
	Pages() int
	Size(index int) (wide, high float64, err error)
	Image(index, dpi int) (image.Image, error)
	Close()
}

// fingerprint is which bytes a document is: where it sits in the vault, and
// what the vault says about the file there. Both caches key on it, so a
// document rewritten under the same name is drawn again.
type fingerprint struct {
	path  string
	size  int64
	mtime int64
}

// errBusy is a document that could not be reached inside the bound: a worker of
// the library's pool is held elsewhere, or the document is answering somebody
// else. It is an answer and not a failure, and the window asks again.
var errBusy = errors.New("the document is busy")

// document is one document open on a worker of the library's pool.
type document struct {
	// lock is held for the length of one call into the library, which
	// answers one question about one document at a time. It is a channel
	// because a request waiting for it waits under a bound.
	lock chan struct{}

	// ready is closed once the document is open or the reason it is not is
	// known. Whoever asked for it while it was opening waits here.
	ready chan struct{}
	scan  scan
	why   error

	// points is how wide each page is in its own units, measured when the page
	// is first drawn. It is read and written under drawing.
	points map[int]int

	uses int
	idle *time.Timer
}

// hold takes the document for one call into the library, and answers false
// where the caller ran out of time.
func (d *document) hold(ctx context.Context) bool {
	select {
	case d.lock <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *document) release() { <-d.lock }

// shut closes the document once whatever is opening it has finished.
func (d *document) shut() {
	select {
	case <-d.ready:
		if d.scan != nil {
			d.scan.Close()
		}
	default:
		go func() {
			<-d.ready
			if d.scan != nil {
				d.scan.Close()
			}
		}()
	}
}

// documents are the documents held open. A file is read and parsed once,
// however many of its pages are turned.
//
// Few are held: an open document keeps one of the library's workers for as long
// as it is open, and a recognition keeps another for as long as it reads. One
// nobody has looked at for a while is closed.
type documents struct {
	mu   sync.Mutex
	open map[fingerprint]*document
	// order is what is held, least recently asked for first.
	order []fingerprint
	// limit is how many are held open at once, and idleFor how long one nobody
	// is looking at is kept.
	limit   int
	idleFor time.Duration

	// closing is the window going. A document being drawn from is left to the
	// reader holding it, and empty is closed when the last of them gives it
	// back.
	closing bool
	empty   chan struct{}
	done    sync.Once
}

// How many documents are held open at once, and how long one nobody is looking
// at is kept.
const (
	mostOpen    = 2
	openIdleFor = 2 * time.Minute
)

// keeping is a window with nothing open yet.
func keeping() *documents {
	return &documents{
		open:    map[fingerprint]*document{},
		limit:   mostOpen,
		idleFor: openIdleFor,
		empty:   make(chan struct{}),
	}
}

// take hands over the document at a fingerprint, opening it if it is not open.
// The function returned gives it back.
//
// Opening reads and parses the whole file and waits for a worker of the pool,
// so it happens on a goroutine of its own: a caller whose ctx ends first is told
// the document is busy, and the open goes on and is there for the next ask. Two
// callers asking at once open it once.
func (d *documents) take(
	ctx context.Context,
	print fingerprint,
	open func() (scan, error),
) (*document, func(), error) {
	d.mu.Lock()
	if d.closing {
		d.mu.Unlock()
		return nil, nil, errBusy
	}
	doc, held := d.open[print]
	if !held {
		doc = &document{
			lock:   make(chan struct{}, 1),
			ready:  make(chan struct{}),
			points: map[int]int{},
		}
		d.open[print] = doc
		go d.fill(print, doc, open)
	}
	doc.uses++
	if doc.idle != nil {
		doc.idle.Stop()
		doc.idle = nil
	}
	d.touch(print)
	d.evict()
	d.mu.Unlock()

	select {
	case <-doc.ready:
	case <-ctx.Done():
		d.give(print, doc)
		return nil, nil, errBusy
	}
	if doc.why != nil {
		d.give(print, doc)
		return nil, nil, doc.why
	}
	return doc, func() { d.give(print, doc) }, nil
}

// fill opens the document and tells whoever is waiting.
func (d *documents) fill(print fingerprint, doc *document, open func() (scan, error)) {
	doc.scan, doc.why = open()
	close(doc.ready)
	if doc.why == nil {
		return
	}
	// A document that could not be opened is not kept, so the next ask reads
	// the file again.
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.open[print] == doc {
		delete(d.open, print)
		d.forget(print)
	}
}

// give puts a document back and starts the clock on closing it.
func (d *documents) give(print fingerprint, doc *document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	doc.uses--
	if doc.uses > 0 {
		return
	}
	if d.closing {
		delete(d.open, print)
		doc.shut()
		if len(d.open) == 0 {
			d.done.Do(func() { close(d.empty) })
		}
		return
	}
	if d.open[print] != doc {
		doc.shut()
		return
	}
	doc.idle = time.AfterFunc(d.idleFor, func() { d.retire(print, doc) })
}

// retire closes a document nobody has asked about for a while.
func (d *documents) retire(print fingerprint, doc *document) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.open[print] != doc || doc.uses > 0 {
		return
	}
	delete(d.open, print)
	d.forget(print)
	doc.shut()
}

// close closes every document held open, once nobody is drawing from one.
//
// Closing gives the library's worker back, and a worker given back while it is
// drawing is taken from under the drawing. What is left to close is what the
// last reader gives back.
func (d *documents) close() {
	d.mu.Lock()
	d.closing = true
	for print, doc := range d.open {
		if doc.idle != nil {
			doc.idle.Stop()
		}
		if doc.uses > 0 {
			continue
		}
		delete(d.open, print)
		doc.shut()
	}
	d.order = nil
	held := len(d.open)
	d.mu.Unlock()
	if held == 0 {
		return
	}
	<-d.empty
}

// evict closes what is over the bound, oldest first. A document somebody is
// drawing from stays, and the bound is over until they are done with it.
func (d *documents) evict() {
	for len(d.open) > d.limit {
		dropped := -1
		for i, print := range d.order {
			if doc := d.open[print]; doc != nil && doc.uses == 0 {
				dropped = i
				break
			}
		}
		if dropped < 0 {
			return
		}
		print := d.order[dropped]
		doc := d.open[print]
		if doc.idle != nil {
			doc.idle.Stop()
		}
		delete(d.open, print)
		d.order = append(d.order[:dropped], d.order[dropped+1:]...)
		doc.shut()
	}
}

// touch puts a fingerprint at the end of the order, which is the most recently
// asked for.
func (d *documents) touch(print fingerprint) {
	d.forget(print)
	d.order = append(d.order, print)
}

func (d *documents) forget(print fingerprint) {
	for i, held := range d.order {
		if held == print {
			d.order = append(d.order[:i], d.order[i+1:]...)
			return
		}
	}
}
