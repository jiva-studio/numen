package webui

import (
	"container/list"
	"context"
	"sync"
)

// shot is one drawn page: which document it belongs to, which page it is, and
// how wide it was drawn. The width is part of the key because the window asks
// for the width its screen has, and a page drawn for one width is not the page
// another width asks for.
type shot struct {
	of   fingerprint
	at   int
	wide int
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

// kept is one picture in the order it was last asked for.
type kept struct {
	key shot
	pic *picture
}

// pictures are the pages already drawn, the least recently asked for dropped
// once what is held reaches the bound.
//
// They are held in memory alone: a page is drawn again in a fraction of a
// second, and what is here goes when the window does.
type pictures struct {
	mu    sync.Mutex
	by    map[shot]*list.Element
	order *list.List
	// bytes is what the drawings held come to, and most is what they may come
	// to.
	bytes int
	most  int
}

// mostDrawn is how many bytes of drawn pages are held.
const mostDrawn = 64 << 20

// drawings is a window with nothing drawn yet.
func drawings() *pictures {
	return &pictures{by: map[shot]*list.Element{}, order: list.New(), most: mostDrawn}
}

// draw hands over one drawn page, drawing it where it is not held. Several asks
// for the same page draw it once and are answered with the one drawing.
func (p *pictures) draw(ctx context.Context, key shot, drawn func() ([]byte, error)) ([]byte, error) {
	p.mu.Lock()
	if el, held := p.by[key]; held {
		p.order.MoveToFront(el)
		pic := el.Value.(*kept).pic
		p.mu.Unlock()
		return pic.wait(ctx)
	}
	pic := &picture{ready: make(chan struct{})}
	p.by[key] = p.order.PushFront(&kept{key: key, pic: pic})
	p.mu.Unlock()

	pic.body, pic.why = drawn()
	close(pic.ready)

	p.mu.Lock()
	defer p.mu.Unlock()
	// It may have been dropped for room while it was being drawn, and then it
	// is this caller's answer and nothing else's.
	if el, still := p.by[key]; !still || el.Value.(*kept).pic != pic {
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
func (p *pictures) has(key shot) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, held := p.by[key]
	return held
}

// drop takes one drawing out.
func (p *pictures) drop(key shot) {
	el, held := p.by[key]
	if !held {
		return
	}
	delete(p.by, key)
	p.order.Remove(el)
	if pic := el.Value.(*kept).pic; pic.why == nil {
		p.bytes -= len(pic.body)
	}
}

// trim drops the least recently asked for until what is held is inside the
// bound. A drawing still being made is left alone: it is nobody's to count
// until it is done.
func (p *pictures) trim() {
	for el := p.order.Back(); el != nil && p.bytes > p.most; {
		before := el.Prev()
		entry := el.Value.(*kept)
		select {
		case <-entry.pic.ready:
			p.drop(entry.key)
		default:
		}
		el = before
	}
}
