package pagecache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// Drop is exported and may be called from any goroutine, concurrently with
// Draw and Has on other keys. None of index, order, or bytes may be touched
// outside p.mu by any of the three.
func TestDropIsSafeConcurrentWithDrawAndHas(t *testing.T) {
	cache := NewMemory(MaxMemoryBytes)
	renderFn := func() ([]byte, error) { return make([]byte, 100), nil }

	target := make([]document.PageKey, 10)
	for at := range target {
		target[at] = newTestID(at)
		cache.GetOrRender(t.Context(), target[at], renderFn)
	}

	var wg sync.WaitGroup
	for at := range 50 {
		wg.Add(2)
		go func(at int) {
			defer wg.Done()
			cache.GetOrRender(t.Context(), newTestID(1000+at), renderFn)
		}(at)
		go func(at int) {
			defer wg.Done()
			key := target[at%len(target)]
			cache.Has(key)
			cache.Remove(key)
		}(at)
	}
	wg.Wait()
}

// newTestID names one page of the one book these tests keep drawings of.
func newTestID(at int) document.PageKey {
	return document.PageKey{Document: document.Fingerprint{Path: "book.pdf", Size: 1, Mtime: 1}, Page: at, Width: 400}
}

func TestMemoryContainsSixtyFourMebibytes(t *testing.T) {
	if MaxMemoryBytes != 64<<20 {
		t.Errorf("MaxMemoryBytes is %d bytes, want %d", MaxMemoryBytes, 64<<20)
	}
}

// A page asked for again is answered from what is held, and drawn once.
func TestAPageGeneratedIsHeldAndNotGeneratedAgain(t *testing.T) {
	cache := NewMemory(MaxMemoryBytes)
	calls := 0
	draw := func() ([]byte, error) {
		calls++
		return []byte("page"), nil
	}

	for range 2 {
		got, err := cache.GetOrRender(t.Context(), newTestID(0), draw)
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
	cache := NewMemory(MaxMemoryBytes)
	failure := errors.New("the page did not draw")
	if _, err := cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) { return nil, failure }); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
	if cache.Has(newTestID(0)) {
		t.Error("a failed drawing is held")
	}
	got, err := cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) { return []byte("page"), nil })
	if err != nil || string(got) != "page" {
		t.Errorf("the next ask answered %q, %v", got, err)
	}
}

func TestDropConsumesThePageOutAndFreesItsRoom(t *testing.T) {
	cache := NewMemory(100)
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(60))
	cache.GetOrRender(t.Context(), newTestID(1), newDrawing(30))

	cache.Remove(newTestID(0))
	if cache.Has(newTestID(0)) {
		t.Error("a page dropped is still held")
	}

	cache.GetOrRender(t.Context(), newTestID(2), newDrawing(60))
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
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(1), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(2), newDrawing(20))
	for at := range 3 {
		if !cache.Has(newTestID(at)) {
			t.Fatalf("page %d was dropped with what is held exactly at the bound", at)
		}
	}

	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(3), newDrawing(40))

	for at, isHeld := range []bool{true, false, true, true} {
		if got := cache.Has(newTestID(at)); got != isHeld {
			t.Errorf("page %d: held = %v, want %v", at, got, isHeld)
		}
	}
}

// A page dropped and drawn again is one page: the old drawing of it does not
// take the new one out when room is made.
func TestAPageGeneratedAgainAfterADropIsHeldOnce(t *testing.T) {
	cache := NewMemory(100)
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.Remove(newTestID(0))
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(1), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(2), newDrawing(40))

	for at, isHeld := range []bool{true, false, true} {
		if got := cache.Has(newTestID(at)); got != isHeld {
			t.Errorf("page %d: held = %v, want %v", at, got, isHeld)
		}
	}
}

// A page dropped while it is being drawn is answered to the one who asked,
// and neither held nor counted against the bound.
func TestAPageDroppedWhileBeingGeneratedIsAnsweredAndNotHeld(t *testing.T) {
	cache := NewMemory(100)
	got, err := cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) {
		cache.Remove(newTestID(0))
		return make([]byte, 100), nil
	})
	if err != nil || len(got) != 100 {
		t.Fatalf("the caller was answered %d bytes, %v", len(got), err)
	}
	if cache.Has(newTestID(0)) {
		t.Error("a page dropped while being drawn is held")
	}

	cache.GetOrRender(t.Context(), newTestID(1), newDrawing(60))
	cache.GetOrRender(t.Context(), newTestID(2), newDrawing(40))
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

func TestAFailedPageDroppedWhileBeingGeneratedAnswersItsFailure(t *testing.T) {
	cache := NewMemory(MaxMemoryBytes)
	failure := errors.New("the page did not draw")
	_, err := cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) {
		cache.Remove(newTestID(0))
		return nil, failure
	})
	if !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

// Room is made for as many pages as it takes, oldest first.
func TestRoomIsMadeForAPageLargerThanTheOldest(t *testing.T) {
	cache := NewMemory(100)
	cache.GetOrRender(t.Context(), newTestID(0), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(1), newDrawing(40))
	cache.GetOrRender(t.Context(), newTestID(2), newDrawing(90))

	for at, isHeld := range []bool{false, false, true} {
		if got := cache.Has(newTestID(at)); got != isHeld {
			t.Errorf("page %d: held = %v, want %v", at, got, isHeld)
		}
	}
}

// A second ask for a page being drawn waits for that drawing, and is
// answered with its failure.
func TestAnAskWaitingOnAFailedDrawingIsAnsweredWithItsFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewMemory(MaxMemoryBytes)
		failure := errors.New("the page did not draw")
		release := make(chan struct{})
		answers := make([]error, 2)
		var wg sync.WaitGroup

		wg.Go(func() {
			_, answers[0] = cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) {
				<-release
				return nil, failure
			})
		})
		synctest.Wait()
		wg.Go(func() {
			_, answers[1] = cache.GetOrRender(t.Context(), newTestID(0), newDrawing(10))
		})
		synctest.Wait()
		close(release)
		wg.Wait()

		for at, err := range answers {
			if !errors.Is(err, failure) {
				t.Errorf("ask %d was answered %v, want %v", at, err, failure)
			}
		}
	})
}

func TestAnAskThatStopsWaitingIsAnsweredBusy(t *testing.T) {
	cache := NewMemory(MaxMemoryBytes)
	stopped, stop := context.WithCancel(t.Context())
	stop()
	var waited error
	cache.GetOrRender(t.Context(), newTestID(0), func() ([]byte, error) {
		_, waited = cache.GetOrRender(stopped, newTestID(0), newDrawing(10))
		return make([]byte, 10), nil
	})
	if !errors.Is(waited, pool.ErrBusy) {
		t.Errorf("got %v, want %v", waited, pool.ErrBusy)
	}
}

// A bound below nothing holds nothing, and a page drawn is still answered.
func TestAMemoryBoundedBelowNothingContainsNothing(t *testing.T) {
	cache := NewMemory(-1)
	body, err := cache.GetOrRender(t.Context(), newTestID(0), newDrawing(10))
	if err != nil || len(body) != 10 {
		t.Fatalf("answered %d bytes, %v", len(body), err)
	}
	if cache.Has(newTestID(0)) {
		t.Error("a page was held under a bound below nothing")
	}
}
