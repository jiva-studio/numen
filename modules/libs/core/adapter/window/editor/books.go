package editor

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// books are the books the window has open. An archive is read and parsed once,
// however many chapters are turned.
//
// Few are held: a book is the whole of its text in memory beside the archive it
// was read out of. One nobody has turned a page of for a while is let go, and
// the reader holding it keeps it as long as it is reading.
type books struct {
	mu   sync.Mutex
	open map[fingerprint]*volume
	// order is what is held, least recently asked for first.
	order []fingerprint
	// limit is how many are held at once, and idleFor how long one nobody is
	// reading is kept.
	limit   int
	idleFor time.Duration
}

// volume is one book read, or the reading of it under way.
type volume struct {
	ready chan struct{}
	book  *epub.Book
	why   error
	idle  *time.Timer
}

// How many books are held at once, and how long one nobody is reading is kept.
const (
	mostRead    = 2
	readIdleFor = 2 * time.Minute
)

// holding is a window with no book open yet: how many books it keeps at once,
// and how long it keeps one nobody is reading.
func holding(limit int, idleFor time.Duration) *books {
	return &books{open: map[fingerprint]*volume{}, limit: limit, idleFor: idleFor}
}

// take hands over the book at a fingerprint, reading it if it is not held.
//
// Reading unpacks and parses the whole archive, so it happens on a goroutine of
// its own: a caller whose ctx ends first is told the book is busy, and the
// reading goes on and is there for the next ask. Two callers asking at once
// read it once.
func (b *books) take(
	ctx context.Context,
	print fingerprint,
	read func() (*epub.Book, error),
) (*epub.Book, error) {
	b.mu.Lock()
	held, open := b.open[print]
	if !open {
		held = &volume{ready: make(chan struct{})}
		b.open[print] = held
		go b.fill(print, held, read)
	}
	if held.idle != nil {
		held.idle.Stop()
	}
	held.idle = time.AfterFunc(b.idleFor, func() { b.retire(print, held) })
	b.touch(print)
	b.evict()
	b.mu.Unlock()

	select {
	case <-held.ready:
		return held.book, held.why
	case <-ctx.Done():
		return nil, errBusy
	}
}

// fill reads the book and tells whoever is waiting.
func (b *books) fill(print fingerprint, held *volume, read func() (*epub.Book, error)) {
	held.book, held.why = read()
	close(held.ready)
	if held.why == nil {
		return
	}
	// A book that could not be read is not kept, so the next ask reads the file
	// again.
	b.mu.Lock()
	defer b.mu.Unlock()
	b.drop(print, held)
}

// retire lets go of a book nobody has asked about for a while.
func (b *books) retire(print fingerprint, held *volume) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.drop(print, held)
}

// evict lets go of what is over the bound, oldest first.
func (b *books) evict() {
	for len(b.order) > b.limit {
		print := b.order[0]
		b.drop(print, b.open[print])
	}
}

// drop takes one book out, which is the whole of letting one go: what a book
// holds is memory.
func (b *books) drop(print fingerprint, held *volume) {
	if held == nil || b.open[print] != held {
		return
	}
	if held.idle != nil {
		held.idle.Stop()
	}
	delete(b.open, print)
	b.forget(print)
}

// touch puts a fingerprint at the end of the order, which is the most recently
// asked for.
func (b *books) touch(print fingerprint) {
	b.forget(print)
	b.order = append(b.order, print)
}

func (b *books) forget(print fingerprint) {
	for i, held := range b.order {
		if held == print {
			b.order = append(b.order[:i], b.order[i+1:]...)
			return
		}
	}
}

// close lets go of every book the window holds.
func (b *books) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, print := range append([]fingerprint(nil), b.order...) {
		b.drop(print, b.open[print])
	}
}
