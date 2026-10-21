package pagecache

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
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
func newDrawingID(at int) document.PageKey {
	return document.PageKey{Document: document.Fingerprint{Path: "book.pdf", Size: 1, Mtime: 1}, Page: at, Width: 400}
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

func TestTheDiskKeepsHalfAGibibyteAndSweepsEveryEighthOfIt(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	if cache.limit != 512<<20 {
		t.Errorf("the bound is %d bytes, want %d", cache.limit, 512<<20)
	}
	if cache.every != 64<<20 {
		t.Errorf("a sweep is due every %d bytes, want %d", cache.every, 64<<20)
	}
}

func TestAPageKeptIsReadBack(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	for _, size := range []int{1, 2, 1000} {
		page := bytes.Repeat([]byte{byte(size)}, size)
		cache.Put(newDrawingID(size), page)
		if got := cache.Get(newDrawingID(size)); !bytes.Equal(got, page) {
			t.Errorf("a page of %d bytes read back as %d bytes", size, len(got))
		}
	}
	if got := cache.Get(newDrawingID(7)); got != nil {
		t.Errorf("a page never kept read back as %d bytes", len(got))
	}
}

func TestAnEmptyPageIsNotKept(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	cache.Put(newDrawingID(0), nil)
	cache.Put(newDrawingID(1), []byte{})
	if n := countDrawings(t, cache); n != 0 {
		t.Errorf("the folder holds %d files after two empty pages", n)
	}
}

func TestNoDiskKeepsNothing(t *testing.T) {
	var cache *Disk
	cache.Put(newDrawingID(0), []byte("page"))
	if got := cache.Get(newDrawingID(0)); got != nil {
		t.Errorf("no disk read back %q", got)
	}
	cache.TrimTo(0)
	cache.Close()
}

func TestAFolderThatIsNotThereKeepsNothing(t *testing.T) {
	cache := NewDiskIn(filepath.Join(t.TempDir(), "missing"))
	cache.Put(newDrawingID(0), []byte("page"))
	if got := cache.Get(newDrawingID(0)); got != nil {
		t.Errorf("a folder that is not there read back %q", got)
	}
}

// A page that cannot be moved into place leaves no half-written file behind.
func TestAPageThatCannotBeMovedIntoPlaceLeavesNothingBehind(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	blocked := cache.GetPath(newDrawingID(0))
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "inside"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache.Put(newDrawingID(0), []byte("page"))
	if got := cache.Get(newDrawingID(0)); got != nil {
		t.Errorf("a page that was never kept read back as %#v", got)
	}

	found, err := os.ReadDir(cache.GetDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name() != filepath.Base(blocked) {
		names := make([]string, 0, len(found))
		for _, one := range found {
			names = append(names, one.Name())
		}
		t.Errorf("the folder holds %v, want only the directory in the page's place", names)
	}
}

// A sweep is due once as many bytes as the share between sweeps have been
// written since the last one, and not a byte sooner.
func TestASweepIsDueEachTimeItsShareHasBeenWritten(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	cache.limit = 0
	cache.every = 1000

	for _, step := range []struct {
		size int
		kept int
	}{
		{1000, 0},
		{999, 1},
		{1, 0},
	} {
		cache.Put(newDrawingID(step.size), make([]byte, step.size))
		cache.going.Wait()
		if n := countDrawings(t, cache); n != step.kept {
			t.Fatalf("after a page of %d bytes the folder holds %d drawings, want %d", step.size, n, step.kept)
		}
	}
	cache.Close()
}

// A sweep drops the drawings looked at longest ago, and stops at the bound.
func TestASweepDropsWhatWasLookedAtLongestAgo(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	page := make([]byte, 100)
	names := make([]string, 4)
	for at := range names {
		cache.Put(newDrawingID(at), page)
		names[at] = cache.GetPath(newDrawingID(at))
	}
	sort.Strings(names)

	// The two looked at longest ago sort in the middle by name.
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for at, offset := range []time.Duration{3, 1, 2, 4} {
		when := base.Add(offset * time.Hour)
		if err := os.Chtimes(names[at], when, when); err != nil {
			t.Fatal(err)
		}
	}

	cache.TrimTo(2 * int64(len(page)))

	for at, isKept := range []bool{true, false, false, true} {
		_, err := os.Stat(names[at])
		if got := err == nil; got != isKept {
			t.Errorf("drawing %d by name: kept = %v, want %v", at, got, isKept)
		}
	}
}

func TestTheDiskIsAFolderOfThisMachinesCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", t.TempDir())
	under, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}

	cache := NewDisk()
	if cache == nil {
		t.Fatal("no disk where the machine has a cache folder")
	}
	want := filepath.Join(under, "numen", "pages")
	if cache.GetDir() != want {
		t.Errorf("the folder is %s, want %s", cache.GetDir(), want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Errorf("the folder was not made: %v", err)
	}
}

func TestNoDiskWhereTheMachineKeepsNoCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	if cache := NewDisk(); cache != nil {
		t.Errorf("a disk in %s where the machine names no cache folder", cache.GetDir())
	}
}

func TestNoDiskWhereItsFolderCannotBeMade(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(home, []byte("a file"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", home)
	if cache := NewDisk(); cache != nil {
		t.Errorf("a disk in %s, under a file", cache.GetDir())
	}
}

// Closing waits for a sweep that still has files to delete.
func TestCloseReturnsOnlyOnceTheSweepHasEmptiedTheFolder(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	for at := range 200 {
		name := filepath.Join(cache.GetDir(), fmt.Sprintf("old-%d.jpg", at))
		if err := os.WriteFile(name, []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cache.limit = 0
	cache.every = 1
	// The sweep is held until the test releases it, so Close has a sweep to wait
	// for.
	target, release := make(chan struct{}), make(chan struct{})
	cache.beforeSweep = func() {
		close(target)
		<-release
	}

	cache.Put(newDrawingID(0), []byte("page"))
	<-target
	closed := make(chan struct{})
	go func() {
		cache.Close()
		close(closed)
	}()
	for {
		cache.mu.Lock()
		isShut := cache.isShut
		cache.mu.Unlock()
		if isShut {
			break
		}
		runtime.Gosched()
	}
	for range 10000 {
		select {
		case <-closed:
			t.Fatal("Close returned while a sweep was still running")
		default:
			runtime.Gosched()
		}
	}
	close(release)
	<-closed

	if n := countDrawings(t, cache); n != 0 {
		t.Errorf("%d files remain after Close", n)
	}
}

// A sweep keeps what fits the bound when the folder is exactly at it, and
// drops nothing from a folder under it.
func TestASweepOfAFolderWithinTheBoundDropsNothing(t *testing.T) {
	cache := NewDiskIn(t.TempDir())
	for at := range 3 {
		cache.Put(newDrawingID(at), make([]byte, 100))
	}
	cache.TrimTo(300)
	if n := countDrawings(t, cache); n != 3 {
		t.Errorf("a folder at its bound holds %d drawings", n)
	}
	cache.TrimTo(299)
	if n := countDrawings(t, cache); n != 2 {
		t.Errorf("a folder over its bound holds %d drawings", n)
	}
}
