//go:build unix

package pagecache

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
)

// A page that cannot be written leaves no file behind and is not kept.
func TestAPageThatCannotBeSavedPreservesNothingPrior(t *testing.T) {
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

// The folder of drawings is made readable and searchable by everyone, writable
// by its owner.
func TestTheDiskFolderIsMadeWithItsOwnersWriteAccessOnly(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", t.TempDir())
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })

	cache := NewDisk()
	if cache == nil {
		t.Fatal("no disk")
	}
	info, err := os.Stat(cache.GetDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("the folder is %o, want 755", got)
	}
}
