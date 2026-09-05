package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// onDisk gives a window a folder of its own to keep its drawings in.
func onDisk(t *testing.T, api *API) *cache {
	t.Helper()
	kept := keptIn(t.TempDir())
	t.Cleanup(kept.close)
	api.Viewer.kept = kept
	return kept
}

// drawingsIn is how many drawings the folder holds.
func drawingsIn(t *testing.T, kept *cache) int {
	t.Helper()
	found, err := os.ReadDir(kept.dir)
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
	api, handler := drawnFrom(t, from)
	alone(api)
	kept := onDisk(t, api)

	if out := ask(handler, drawnAt(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	_, drawn, _ := from.counted()
	if n := drawingsIn(t, kept); n != 1 {
		t.Fatalf("the folder holds %d drawings after one page", n)
	}

	// Nothing in memory, the way a window opened again begins.
	api.Viewer.drawn.Store(drawings())

	out := ask(handler, drawnAt(t, api, book, 0, 400))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", out.Code)
	}
	if _, again, _ := from.counted(); again != drawn {
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
	api, handler := drawnFrom(t, from)
	alone(api)
	kept := onDisk(t, api)

	for _, wide := range []int{400, 800} {
		if out := ask(handler, drawnAt(t, api, book, 0, wide)); out.Code != http.StatusOK {
			t.Fatalf("asked for a page %d wide and got %d", wide, out.Code)
		}
	}
	if n := drawingsIn(t, kept); n != 2 {
		t.Errorf("one page at two widths is %d drawings, want 2", n)
	}
}

// A document rewritten under the same name is another document. A drawing kept
// for the bytes that were there is a picture of a page that is gone.
func TestADocumentRewrittenIsDrawnAgain(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)
	kept := onDisk(t, api)

	if out := ask(handler, drawnAt(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	_, drawn, _ := from.counted()

	at := filepath.Join(string(api.Showing().Path), book)
	if err := os.WriteFile(at, []byte("the bytes of another scan entirely"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Second)
	if err := os.Chtimes(at, later, later); err != nil {
		t.Fatal(err)
	}
	api.Viewer.drawn.Store(drawings())

	if out := ask(handler, drawnAt(t, api, book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", out.Code)
	}
	if _, again, _ := from.counted(); again <= drawn {
		t.Errorf("the rewritten document was not drawn again: %d drawings both times", drawn)
	}
	if n := drawingsIn(t, kept); n != 2 {
		t.Errorf("the folder holds %d drawings, want 2 — one for each of the two documents", n)
	}
}

// The folder is bounded, and what goes is what has gone longest without being
// looked at.
func TestTheOldestDrawingsGoWhenTheFolderIsFull(t *testing.T) {
	from := sheets(8)
	api, handler := drawnFrom(t, from)
	alone(api)
	kept := onDisk(t, api)

	for at := 0; at < 4; at++ {
		if out := ask(handler, drawnAt(t, api, book, at, 400)); out.Code != http.StatusOK {
			t.Fatalf("asked for page %d and got %d", at, out.Code)
		}
	}
	if n := drawingsIn(t, kept); n != 4 {
		t.Fatalf("the folder holds %d drawings, want 4", n)
	}

	// Older than the rest, the way a page nobody has turned back to is.
	first := filepath.Join(kept.dir, kept.named(pictureID{document: print(t, api), page: 0, width: 400}))
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(first, old, old); err != nil {
		t.Fatal(err)
	}

	kept.limit = totalOf(t, kept) - 1
	kept.sweep()

	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Errorf("the oldest drawing is still there: %v", err)
	}
	if n := drawingsIn(t, kept); n != 3 {
		t.Errorf("the folder holds %d drawings after the sweep, want 3", n)
	}
}

// A sweep runs behind the page whose writing triggered it. The window owns it:
// closing ends it and waits for it, so nothing is still deleting files in a
// folder after the process that started it has said it is done.
func TestTheWindowWaitsForTheSweepItStarted(t *testing.T) {
	kept := keptIn(t.TempDir())
	page := make([]byte, 1000)
	kept.limit = 2 * int64(len(page))
	kept.every = int64(len(page))

	for at := range 6 {
		kept.put(drawingOf(at), page)
	}
	kept.close()

	if total := totalOf(t, kept); total > kept.limit {
		t.Errorf("the folder holds %d bytes after the window closed, over its bound of %d", total, kept.limit)
	}
}

// A page kept after the window has closed starts no sweep: it would be one
// nobody is left to wait for.
func TestNoSweepIsStartedAfterTheWindowCloses(t *testing.T) {
	kept := keptIn(t.TempDir())
	page := make([]byte, 1000)
	kept.limit = 1
	kept.every = int64(len(page))

	kept.close()
	for at := range 4 {
		kept.put(drawingOf(at), page)
	}
	kept.going.Wait()

	if n := drawingsIn(t, kept); n != 4 {
		t.Errorf("a sweep nobody was left to wait for deleted %d of 4 drawings", 4-n)
	}
}

// drawingOf is one page of the one book these tests keep drawings of.
func drawingOf(at int) pictureID {
	return pictureID{document: fingerprint{path: "book.pdf", size: 1, mtime: 1}, page: at, width: 400}
}

// totalOf is what the folder's drawings come to.
func totalOf(t *testing.T, kept *cache) int64 {
	t.Helper()
	found, err := os.ReadDir(kept.dir)
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

// print is what the vault says the book is, which is what a drawing is named
// from.
func print(t *testing.T, api *API) fingerprint {
	t.Helper()
	_, said, err := api.standing(t.Context(), book)
	if err != nil {
		t.Fatal(err)
	}
	return said
}
