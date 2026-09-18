package pool

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
)

// Books are the books the window has open. An archive is read and parsed once,
// however many chapters are turned.
//
// Few are held: a book is the whole of its text in memory beside the archive it
// was read out of. One nobody has turned a page of for a while is let go, and
// the reader holding it keeps it as long as it is reading.
type Books struct {
	// mu is held while open, order, or a volume's timer changes.
	mu sync.Mutex
	// open is the books read, keyed by the fingerprint of the file each
	// came out of.
	open map[Fingerprint]*Volume
	// order is what is held, least recently asked for first.
	order []Fingerprint
	// limit is how many are held at once, and idleFor how long one nobody is
	// reading is kept.
	limit   int
	idleFor time.Duration
}

// Volume is one book read, or the reading of it under way.
type Volume struct {
	// ready is closed once the reading is over, whichever way it went.
	ready chan struct{}
	// book is the book read, and why the reason it is not.
	book *epub.Book
	why  error
	// idle lets the book go once nobody has asked about it for a while.
	idle *time.Timer
}

// How many books are held at once, and how long one nobody is reading is kept.
const (
	MostRead    = 2
	ReadIdleFor = 2 * time.Minute
)

// NewBooks is a window with no book open yet: how many books it keeps at once,
// and how long it keeps one nobody is reading.
func NewBooks(limit int, idleFor time.Duration) *Books {
	return &Books{open: map[Fingerprint]*Volume{}, limit: limit, idleFor: idleFor}
}

// Take hands over the book at a fingerprint, reading it if it is not held.
//
// Reading unpacks and parses the whole archive, so it happens on a goroutine of
// its own: a caller whose ctx ends first is told the book is busy, and the
// reading goes on and is there for the next ask. Two callers asking at once
// read it once.
func (b *Books) Take(
	ctx context.Context,
	mark Fingerprint,
	read func() (*epub.Book, error),
) (*epub.Book, error) {
	b.mu.Lock()
	held, open := b.open[mark]
	if !open {
		held = &Volume{ready: make(chan struct{})}
		b.open[mark] = held
		go b.Fill(mark, held, read)
	}
	if held.idle != nil {
		held.idle.Stop()
	}
	held.idle = time.AfterFunc(b.idleFor, func() { b.Retire(mark, held) })
	b.Touch(mark)
	b.Evict()
	b.mu.Unlock()

	select {
	case <-held.ready:
		return held.book, held.why
	case <-ctx.Done():
		return nil, ErrBusy
	}
}

// Fill reads the book and tells whoever is waiting.
func (b *Books) Fill(mark Fingerprint, held *Volume, read func() (*epub.Book, error)) {
	held.book, held.why = read()
	close(held.ready)
	if held.why == nil {
		return
	}
	// A book that could not be read is not kept, so the next ask reads the file
	// again.
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Drop(mark, held)
}

// Retire lets go of a book nobody has asked about for a while.
func (b *Books) Retire(mark Fingerprint, held *Volume) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Drop(mark, held)
}

// Evict lets go of what is over the bound, oldest first.
func (b *Books) Evict() {
	for len(b.order) > b.limit {
		mark := b.order[0]
		b.Drop(mark, b.open[mark])
	}
}

// Drop takes one book out, which is the whole of letting one go: what a book
// holds is memory.
func (b *Books) Drop(mark Fingerprint, held *Volume) {
	if held == nil || b.open[mark] != held {
		return
	}
	if held.idle != nil {
		held.idle.Stop()
	}
	delete(b.open, mark)
	b.Forget(mark)
}

// Touch puts a fingerprint at the end of the order, which is the most recently
// asked for.
func (b *Books) Touch(mark Fingerprint) {
	b.Forget(mark)
	b.order = append(b.order, mark)
}

// Forget takes a fingerprint out of the order, wherever it stands in it.
func (b *Books) Forget(mark Fingerprint) {
	for i, held := range b.order {
		if held == mark {
			b.order = append(b.order[:i], b.order[i+1:]...)
			return
		}
	}
}

// Close lets go of every book the window holds.
func (b *Books) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, print := range append([]Fingerprint(nil), b.order...) {
		b.Drop(print, b.open[print])
	}
}

// Len is how many books are held.
func (b *Books) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.open)
}
