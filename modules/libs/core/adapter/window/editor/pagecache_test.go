package editor

import (
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pagecache"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newDiskCache gives a window a folder of its own to keep its drawings in.
func newDiskCache(t *testing.T, api *API) *pagecache.Disk {
	t.Helper()
	cache := pagecache.NewDiskIn(t.TempDir())
	t.Cleanup(cache.Close)
	api.Viewer.onDisk = cache
	return cache
}

// countDrawings is how many drawings the folder holds.
func countDrawings(t *testing.T, cache *pagecache.Disk) int {
	t.Helper()
	found, err := os.ReadDir(cache.GetDir())
	if err != nil {
		t.Fatal(err)
	}
	return len(found)
}

// A page of a scan is half a second of decoding whatever size is asked for, and
// it is the same half second every time it is turned back to. A page drawn once
// is drawn once.
func TestAPageDrawnBeforeIsNotDrawnAgain(t *testing.T) {
	from := sheets(4)
	api, handler := openViewerWindow(t, from)
	stopReadAhead(api)
	cache := newDiskCache(t, api)

	if out := ask(handler, getPageAddress(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	_, drawn, _ := from.getCounts()
	if n := countDrawings(t, cache); n != 1 {
		t.Fatalf("the folder holds %d drawings after one page", n)
	}

	// Nothing in memory, the way a window opened again begins.
	api.Viewer.drawn.Store(pagecache.NewMemory(pagecache.MostDrawn))

	out := ask(handler, getPageAddress(t, api, book, 0, 400))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", out.Code)
	}
	if _, again, _ := from.getCounts(); again != drawn {
		t.Errorf("the page was drawn %d times, and %d of them after it was kept", again, again-drawn)
	}
	if out.Body.Len() == 0 {
		t.Error("what came back from the folder is empty")
	}
}

// The width is part of what a drawing is, so a page asked for at another width
// is another drawing.
func TestAPageAtAnotherWidthIsAnotherDrawing(t *testing.T) {
	from := sheets(4)
	api, handler := openViewerWindow(t, from)
	stopReadAhead(api)
	cache := newDiskCache(t, api)

	for _, wide := range []int{400, 800} {
		if out := ask(handler, getPageAddress(t, api, book, 0, wide)); out.Code != http.StatusOK {
			t.Fatalf("asked for a page %d wide and got %d", wide, out.Code)
		}
	}
	if n := countDrawings(t, cache); n != 2 {
		t.Errorf("one page at two widths is %d drawings, want 2", n)
	}
}

// A document rewritten under the same name is another document. A drawing kept
// for the bytes that were there is a picture of a page that is gone.
func TestADocumentRewrittenIsDrawnAgain(t *testing.T) {
	from := sheets(4)
	api, handler := openViewerWindow(t, from)
	stopReadAhead(api)
	cache := newDiskCache(t, api)

	if out := ask(handler, getPageAddress(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	_, drawn, _ := from.getCounts()

	at := filepath.Join(string(api.GetShownVault().Path), book)
	if err := os.WriteFile(at, []byte("the bytes of another scan entirely"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Second)
	if err := os.Chtimes(at, later, later); err != nil {
		t.Fatal(err)
	}
	api.Viewer.drawn.Store(pagecache.NewMemory(pagecache.MostDrawn))

	if out := ask(handler, getPageAddress(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", out.Code)
	}
	if _, again, _ := from.getCounts(); again <= drawn {
		t.Errorf("the rewritten document was not drawn again: %d drawings both times", drawn)
	}
	if n := countDrawings(t, cache); n != 2 {
		t.Errorf("the folder holds %d drawings, want 2 — one for each of the two documents", n)
	}
}

// The folder is bounded, and what goes is what has gone longest without being
// looked at.
func TestTheOldestDrawingsGoWhenTheFolderIsFull(t *testing.T) {
	from := sheets(8)
	api, handler := openViewerWindow(t, from)
	stopReadAhead(api)
	cache := newDiskCache(t, api)

	for at := 0; at < 4; at++ {
		if out := ask(handler, getPageAddress(t, api, book, at, 400)); out.Code != http.StatusOK {
			t.Fatalf("asked for page %d and got %d", at, out.Code)
		}
	}
	if n := countDrawings(t, cache); n != 4 {
		t.Fatalf("the folder holds %d drawings, want 4", n)
	}

	// Older than the rest, the way a page nobody has turned back to is.
	first := cache.GetPath(pagecache.ID{Document: getFingerprint(t, api), Page: 0, Width: 400})
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(first, old, old); err != nil {
		t.Fatal(err)
	}

	cache.SweepTo(countBytes(t, cache) - 1)

	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Errorf("the oldest drawing is still there: %v", err)
	}
	if n := countDrawings(t, cache); n != 3 {
		t.Errorf("the folder holds %d drawings after the sweep, want 3", n)
	}
}

// getFingerprint is what the vault says the book is, which is what a drawing is named
// from.
func getFingerprint(t *testing.T, api *API) pool.Fingerprint {
	t.Helper()
	_, said, err := api.stat(t.Context(), book)
	if err != nil {
		t.Fatal(err)
	}
	return said
}

// countBytes is what the folder's drawings come to, in bytes.
func countBytes(t *testing.T, cache *pagecache.Disk) int64 {
	t.Helper()
	found, err := os.ReadDir(cache.GetDir())
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, one := range found {
		info, err := one.Info()
		if err != nil {
			t.Fatal(err)
		}
		total += info.Size()
	}
	return total
}
