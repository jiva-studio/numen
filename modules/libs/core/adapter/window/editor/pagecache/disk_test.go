package pagecache

import (
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor/pool"
	"os"
	"testing"
)

// A sweep runs behind the page whose writing triggered it. The window owns it:
// closing ends it and waits for it, so nothing is still deleting files in a
// folder after the process that started it has said it is done.
func TestTheWindowWaitsForTheSweepItStarted(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	page := make([]byte, 1000)
	cache.limit = 2 * int64(len(page))
	cache.every = int64(len(page))

	for at := range 6 {
		cache.Put(newDrawingID(at), page)
	}
	cache.Close()

	if total := countBytes(t, cache); total > cache.limit {
		t.Errorf("the folder holds %d bytes after the window closed, over its bound of %d", total, cache.limit)
	}
}

// A page kept after the window has closed starts no sweep: it would be one
// nobody is left to wait for.
func TestNoSweepIsStartedAfterTheWindowCloses(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	page := make([]byte, 1000)
	cache.limit = 1
	cache.every = int64(len(page))

	cache.Close()
	for at := range 4 {
		cache.Put(newDrawingID(at), page)
	}
	cache.going.Wait()

	if n := countDrawings(t, cache); n != 4 {
		t.Errorf("a sweep nobody was left to wait for deleted %d of 4 drawings", 4-n)
	}
}

// newDrawingID names one page of the one book these tests keep drawings of.
func newDrawingID(at int) ID {
	return ID{Document: pool.Fingerprint{Path: "book.pdf", Size: 1, Mtime: 1}, Page: at, Width: 400}
}

// countBytes is what the folder's drawings come to, in bytes.
func countBytes(t *testing.T, cache *Disk) int64 {
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

// countDrawings is how many drawings the folder holds.
func countDrawings(t *testing.T, cache *Disk) int {
	t.Helper()
	found, err := os.ReadDir(cache.GetDir())
	if err != nil {
		t.Fatal(err)
	}
	return len(found)
}
