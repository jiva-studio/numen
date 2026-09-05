package editor

import (
	"container/list"
	"context"
	"sync"
)

// pictureID is one drawn page: which document it belongs to, which page it is, and
// how wide it was drawn. The width is part of the key because the window asks
// for the width its screen has, and a page drawn for one width is not the page
// another width asks for.
type pictureID struct {
	document fingerprint
	page     int
	width    int
}

// picture is one page drawn and encoded, or the drawing of it under way.
type picture struct {
	ready chan struct{}
	body  []byte
	why   error
}

// wait is the drawing, once it is done.
func (p *picture) wait(ctx context.Context) ([]byte, error) {
	select {
	case <-p.ready:
		return p.body, p.why
	case <-ctx.Done():
		return nil, errBusy
	}
}

// entry is one picture in the order it was last asked for.
type entry struct {
	key pictureID
	pic *picture
}

// pictures are the pages already drawn, the least recently asked for dropped
// once what is held reaches the bound.
//
// They are held in memory alone: a page is drawn again in a fraction of a
// second, and what is here goes when the window does.
type pictures struct {
	mu    sync.Mutex
	index map[pictureID]*list.Element
	order *list.List
	// bytes is what the drawings held come to, and limit is what they may come
	// to.
	bytes int
	limit int
}

// mostDrawn is how many bytes of drawn pages are held.
const mostDrawn = 64 << 20

// drawings is a window with nothing drawn yet.
func drawings() *pictures {
	return &pictures{index: map[pictureID]*list.Element{}, order: list.New(), limit: mostDrawn}
}

// draw hands over one drawn page, drawing it where it is not held. Several asks
// for the same page draw it once and are answered with the one drawing.
func (p *pictures) draw(ctx context.Context, key pictureID, drawn func() ([]byte, error)) ([]byte, error) {
	p.mu.Lock()
	if el, held := p.index[key]; held {
		p.order.MoveToFront(el)
		pic := el.Value.(*entry).pic
		p.mu.Unlock()
		return pic.wait(ctx)
	}
	pic := &picture{ready: make(chan struct{})}
	p.index[key] = p.order.PushFront(&entry{key: key, pic: pic})
	p.mu.Unlock()

	pic.body, pic.why = drawn()
	close(pic.ready)

	p.mu.Lock()
	defer p.mu.Unlock()
	// It may have been dropped for room while it was being drawn, and then it
	// is this caller's answer and nothing else's.
	if el, still := p.index[key]; !still || el.Value.(*entry).pic != pic {
		return pic.body, pic.why
	}
	if pic.why != nil {
		p.drop(key)
		return nil, pic.why
	}
	p.bytes += len(pic.body)
	p.trim()
	return pic.body, nil
}

// has says whether a page is drawn already, which is what deciding to draw one
// ahead turns on.
func (p *pictures) has(key pictureID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, held := p.index[key]
	return held
}

// drop takes one drawing out.
func (p *pictures) drop(key pictureID) {
	el, held := p.index[key]
	if !held {
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
func (p *pictures) trim() {
	for el := p.order.Back(); el != nil && p.bytes > p.limit; {
		before := el.Prev()
		oldest := el.Value.(*entry)
		select {
		case <-oldest.pic.ready:
			p.drop(oldest.key)
		default:
		}
		el = before
	}
}
