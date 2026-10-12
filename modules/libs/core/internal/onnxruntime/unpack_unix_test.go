//go:build unix

package onnxruntime

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// An unpacked library can be run by anybody who can read it.
func TestAnUnpackedLibraryCanBeRun(t *testing.T) {
	dir := isolate(t)
	found := createArchive(t, dir)
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)

	at, err := unpack(found.archive, "libonnxruntime.dylib", found)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Clean(at))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("the library is %v, want rwxr-xr-x", info.Mode().Perm())
	}
}
