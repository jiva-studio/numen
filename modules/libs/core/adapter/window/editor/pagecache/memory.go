// Package pagecache holds the pages a document has been drawn into, in memory
// and on the disk beside it. A drawing is made from the document alone, so
// either half may be emptied at any moment and the page is drawn again.
package pagecache

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// renderedPage is one page drawn and encoded, or the drawing of it under way.
type renderedPage struct {
	ready chan struct{}
	body  []byte
	why   error
}

// wait is the drawing, once it is done.
func (p *renderedPage) wait(ctx context.Context) ([]byte, error) {
	select {
	case <-p.ready:
		return p.body, p.why
	case <-ctx.Done():
		return nil, pool.ErrBusy
	}
}

// entry is one picture in the order it was last asked for.
type entry struct {
	key  document.PageKey
	pic  *renderedPage
	prev *entry
	next *entry
}

type entryList struct {
	head *entry
	tail *entry
}

func (l *entryList) pushFront(e *entry) {
	e.prev = nil
	e.next = l.head
	if l.head != nil {
		l.head.prev = e
	} else {
		l.tail = e
	}
	l.head = e
}

func (l *entryList) moveToFront(e *entry) {
	if l.head == e {
		return
	}
	l.remove(e)
	l.pushFront(e)
}

func (l *entryList) remove(e *entry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		l.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		l.tail = e.prev
	}
	e.prev = nil
	e.next = nil
}

// Memory are the pages already drawn, the least recently asked for dropped
// once what is held reaches the bound.
//
// They are held in memory alone: a page is drawn again in a fraction of a
// second, and what is here goes when the window does.
type Memory struct {
	mu    sync.Mutex
	index map[document.PageKey]*entry
	order entryList
	// bytes is what the drawings held come to, and limit is what they may come
	// to.
	bytes int
	limit int
}

// MaxMemoryBytes is how many bytes of drawn pages are held.
const MaxMemoryBytes = 64 << 20

// NewMemory is a window with nothing drawn yet, holding up to this many bytes.
func NewMemory(limit int) *Memory {
	return &Memory{index: map[document.PageKey]*entry{}, limit: limit}
}

// GetOrRender hands over one drawn page, drawing it where it is not held. Several asks
// for the same page draw it once and are answered with the one drawing.
func (p *Memory) GetOrRender(ctx context.Context, key document.PageKey, drawn func() ([]byte, error)) ([]byte, error) {
	p.mu.Lock()
	if e, ok := p.index[key]; ok {
		p.order.moveToFront(e)
		pic := e.pic
		p.mu.Unlock()
		return pic.wait(ctx)
	}
	pic := &renderedPage{ready: make(chan struct{})}
	e := &entry{key: key, pic: pic}
	p.order.pushFront(e)
	p.index[key] = e
	p.mu.Unlock()

	pic.body, pic.why = drawn()
	close(pic.ready)

	p.mu.Lock()
	defer p.mu.Unlock()
	// It may have been dropped for room while it was being drawn, and then it
	// is this caller's answer and nothing else's.
	if current, still := p.index[key]; !still || current.pic != pic {
		return pic.body, pic.why
	}
	if pic.why != nil {
		p.remove(key)
		return nil, pic.why
	}
	p.bytes += len(pic.body)
	p.trim()
	return pic.body, nil
}

// Has says whether a page is drawn already, which is what deciding to draw one
// ahead turns on.
func (p *Memory) Has(key document.PageKey) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.index[key]
	return ok
}

// Remove takes one drawing out.
func (p *Memory) Remove(key document.PageKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.remove(key)
}

// remove takes one drawing out. The caller holds p.mu.
func (p *Memory) remove(key document.PageKey) {
	e, ok := p.index[key]
	if !ok {
		return
	}
	delete(p.index, key)
	p.order.remove(e)
	if e.pic.why == nil {
		p.bytes -= len(e.pic.body)
	}
}

// trim drops the least recently asked for until what is held is inside the
// bound. A drawing still being made is left alone: it is nobody's to count
// until it is done.
func (p *Memory) trim() {
	for e := p.order.tail; e != nil && p.bytes > p.limit; {
		before := e.prev
		select {
		case <-e.pic.ready:
			p.remove(e.key)
		default:
		}
		e = before
	}
}
