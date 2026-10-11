package pagecache

import (
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
)

// Drop is exported and may be called from any goroutine, concurrently with
// Draw and Has on other keys. None of index, order, or bytes may be touched
// outside p.mu by any of the three.
func TestDropIsSafeConcurrentWithDrawAndHas(t *testing.T) {
	cache := NewMemory(MostDrawn)
	drawn := func() ([]byte, error) { return make([]byte, 100), nil }

	held := make([]ID, 10)
	for at := range held {
		held[at] = newTestID(at)
		cache.Draw(t.Context(), held[at], drawn)
	}

	var wg sync.WaitGroup
	for at := range 50 {
		wg.Add(2)
		go func(at int) {
			defer wg.Done()
			cache.Draw(t.Context(), newTestID(1000+at), drawn)
		}(at)
		go func(at int) {
			defer wg.Done()
			key := held[at%len(held)]
			cache.Has(key)
			cache.Drop(key)
		}(at)
	}
	wg.Wait()
}

// newTestID names one page of the one book these tests keep drawings of.
func newTestID(at int) ID {
	return ID{Document: pool.Fingerprint{Path: "book.pdf", Size: 1, Mtime: 1}, Page: at, Width: 400}
}
