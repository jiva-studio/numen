//go:build unix

package pagecache

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
)

// A page that cannot be written leaves no file behind and is not kept.
func TestAPageThatCannotBeWrittenLeavesNothingBehind(t *testing.T) {
	cache := NewDiskIn(t.TempDir())

	signal.Ignore(syscall.SIGXFSZ)
	t.Cleanup(func() { signal.Reset(syscall.SIGXFSZ) })
	var before syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &before); err != nil {
		t.Skipf("no file size limit on this platform: %v", err)
	}
	small := syscall.Rlimit{Cur: 4, Max: before.Max}
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &small); err != nil {
		t.Skipf("the file size limit cannot be lowered: %v", err)
	}
	cache.Put(newDrawingID(0), make([]byte, 100))
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &before); err != nil {
		t.Fatal(err)
	}

	if got := cache.Get(newDrawingID(0)); got != nil {
		t.Errorf("a page that could not be written read back as %d bytes", len(got))
	}
	if n := countDrawings(t, cache); n != 0 {
		t.Errorf("the folder holds %d files after a failed write", n)
	}
	if _, err := os.Stat(cache.GetPath(newDrawingID(0))); err == nil {
		t.Error("the page's file exists")
	}
}
