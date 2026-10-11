package pagecache

import (
	"errors"
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

func TestMemoryHoldsSixtyFourMebibytes(t *testing.T) {
	if MostDrawn != 64<<20 {
		t.Errorf("MostDrawn is %d bytes, want %d", MostDrawn, 64<<20)
	}
}

// A page asked for again is answered from what is held, and drawn once.
func TestAPageDrawnIsHeldAndNotDrawnAgain(t *testing.T) {
	cache := NewMemory(MostDrawn)
	calls := 0
	draw := func() ([]byte, error) {
		calls++
		return []byte("page"), nil
	}

	for range 2 {
		got, err := cache.Draw(t.Context(), newTestID(0), draw)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "page" {
			t.Errorf("got %q, want %q", got, "page")
		}
	}
	if calls != 1 {
		t.Errorf("the page was drawn %d times, want once", calls)
	}
	if !cache.Has(newTestID(0)) {
		t.Error("a page drawn is not held")
	}
}

// A drawing that fails is not held: the failure is answered once and the next
// ask draws the page again.
func TestAFailedDrawingIsNotHeld(t *testing.T) {
	cache := NewMemory(MostDrawn)
	failure := errors.New("the page did not draw")
	if _, err := cache.Draw(t.Context(), newTestID(0), func() ([]byte, error) { return nil, failure }); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
	if cache.Has(newTestID(0)) {
		t.Error("a failed drawing is held")
	}
	got, err := cache.Draw(t.Context(), newTestID(0), func() ([]byte, error) { return []byte("page"), nil })
	if err != nil || string(got) != "page" {
		t.Errorf("the next ask answered %q, %v", got, err)
	}
}

func TestDropTakesThePageOutAndFreesItsRoom(t *testing.T) {
	cache := NewMemory(100)
	cache.Draw(t.Context(), newTestID(0), newDrawing(60))
	cache.Draw(t.Context(), newTestID(1), newDrawing(30))

	cache.Drop(newTestID(0))
	if cache.Has(newTestID(0)) {
		t.Error("a page dropped is still held")
	}

	cache.Draw(t.Context(), newTestID(2), newDrawing(60))
	for _, at := range []int{1, 2} {
		if !cache.Has(newTestID(at)) {
			t.Errorf("page %d was dropped, though what is held is inside the bound", at)
		}
	}
}

// Once what is held passes the bound, the pages asked for longest ago go
// until it is inside the bound again, and no more.
func TestTheLeastRecentlyAskedForGoFirst(t *testing.T) {
	cache := NewMemory(100)
	cache.Draw(t.Context(), newTestID(0), newDrawing(40))
	cache.Draw(t.Context(), newTestID(1), newDrawing(40))
	cache.Draw(t.Context(), newTestID(2), newDrawing(20))
	for at := range 3 {
		if !cache.Has(newTestID(at)) {
			t.Fatalf("page %d was dropped with what is held exactly at the bound", at)
		}
	}

	cache.Draw(t.Context(), newTestID(0), newDrawing(40))
	cache.Draw(t.Context(), newTestID(3), newDrawing(40))

	for at, isHeld := range []bool{true, false, true, true} {
		if got := cache.Has(newTestID(at)); got != isHeld {
			t.Errorf("page %d: held = %v, want %v", at, got, isHeld)
		}
	}
}

// A page dropped and drawn again is one page: the old drawing of it does not
// take the new one out when room is made.
func TestAPageDrawnAgainAfterADropIsHeldOnce(t *testing.T) {
	cache := NewMemory(100)
	cache.Draw(t.Context(), newTestID(0), newDrawing(40))
	cache.Drop(newTestID(0))
	cache.Draw(t.Context(), newTestID(0), newDrawing(40))
	cache.Draw(t.Context(), newTestID(1), newDrawing(40))
	cache.Draw(t.Context(), newTestID(0), newDrawing(40))
	cache.Draw(t.Context(), newTestID(2), newDrawing(40))

	for at, isHeld := range []bool{true, false, true} {
		if got := cache.Has(newTestID(at)); got != isHeld {
			t.Errorf("page %d: held = %v, want %v", at, got, isHeld)
		}
	}
}

// A page dropped while it is being drawn is answered to the one who asked,
// and neither held nor counted against the bound.
func TestAPageDroppedWhileBeingDrawnIsAnsweredAndNotHeld(t *testing.T) {
	cache := NewMemory(100)
	got, err := cache.Draw(t.Context(), newTestID(0), func() ([]byte, error) {
		cache.Drop(newTestID(0))
		return make([]byte, 100), nil
	})
	if err != nil || len(got) != 100 {
		t.Fatalf("the caller was answered %d bytes, %v", len(got), err)
	}
	if cache.Has(newTestID(0)) {
		t.Error("a page dropped while being drawn is held")
	}

	cache.Draw(t.Context(), newTestID(1), newDrawing(60))
	cache.Draw(t.Context(), newTestID(2), newDrawing(40))
	for _, at := range []int{1, 2} {
		if !cache.Has(newTestID(at)) {
			t.Errorf("page %d was dropped, though what is held is inside the bound", at)
		}
	}
}

// newDrawing draws a page of this many bytes.
func newDrawing(size int) func() ([]byte, error) {
	return func() ([]byte, error) { return make([]byte, size), nil }
}
