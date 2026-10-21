// Package pagecache holds the pages a document has been drawn into, in memory
// and on the disk beside it. A drawing is made from the document alone, so
// either half may be emptied at any moment and the page is drawn again.
package pagecache

import (
	"container/list"
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
	key document.PageKey
	pic *renderedPage
}

// Memory are the pages already drawn, the least recently asked for dropped
// once what is held reaches the bound.
//
// They are held in memory alone: a page is drawn again in a fraction of a
// second, and what is here goes when the window does.
type Memory struct {
	mu    sync.Mutex
	index map[document.PageKey]*list.Element
	order *list.List
	// bytes is what the drawings held come to, and limit is what they may come
	// to.
	bytes int
	limit int
}

// MaxMemoryBytes is how many bytes of drawn pages are held.
const MaxMemoryBytes = 64 << 20

// NewMemory is a window with nothing drawn yet, holding up to this many bytes.
func NewMemory(limit int) *Memory {
	return &Memory{index: map[document.PageKey]*list.Element{}, order: list.New(), limit: limit}
}

// GetOrRender hands over one drawn page, drawing it where it is not held. Several asks
// for the same page draw it once and are answered with the one drawing.
func (p *Memory) GetOrRender(ctx context.Context, key document.PageKey, drawn func() ([]byte, error)) ([]byte, error) {
	p.mu.Lock()
	if el, ok := p.index[key]; ok {
		p.order.MoveToFront(el)
		pic := el.Value.(*entry).pic
		p.mu.Unlock()
		return pic.wait(ctx)
	}
	pic := &renderedPage{ready: make(chan struct{})}
	p.index[key] = p.order.PushFront(&entry{key: key, pic: pic})
	p.mu.Unlock()

	pic.body, pic.why = drawn()
	close(pic.ready)

	p.mu.Lock()
	defer p.mu.Unlock()
	// It may have been dropped for room while it was being drawn, and then it
	// is this caller's answer and nothing else's.
	if el, found := p.index[key]; !found || el.Value.(*entry).pic != pic {
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
	el, ok := p.index[key]
	if !ok {
		return
	}
	delete(p.index, key)
	p.order.Remove(el)
	if pic := el.Value.(*entry).pic; pic.why == nil {
		p.bytes -= len(pic.body)
	}
}

// trim drops the least recently asked for until what is held is inside the
// bound. A drawing still being made is left alone: it is nobody's to count
// until it is done.
func (p *Memory) trim() {
	for el := p.order.Back(); el != nil && p.bytes > p.limit; {
		before := el.Prev()
		oldest := el.Value.(*entry)
		select {
		case <-oldest.pic.ready:
			p.remove(oldest.key)
		default:
		}
		el = before
	}
}
