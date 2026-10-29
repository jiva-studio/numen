//go:build linux || darwin

package appstate_test

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
)

// A folder the write makes is open to its owner alone, whatever the umask.
func TestAMadeFolderIsTheOwnersAlone(t *testing.T) {
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })

	dir := filepath.Join(t.TempDir(), "cache")
	if err := appstate.VaultFilesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("mode %v", got)
	}
}

// A write that exceeds the file size limit fails, leaves no temporary file and
// keeps no descriptor open.
func TestWrite_FileSizeLimitRemovesTempFileAndDescriptor(t *testing.T) {
	dir := t.TempDir()
	files := appstate.VaultFilesAt(dir)

	signal.Ignore(syscall.SIGXFSZ)
	t.Cleanup(func() { signal.Reset(syscall.SIGXFSZ) })
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Fatal(err)
	}
	limit := old
	// The limit is above every file the test binary writes for itself, its log
	// among them.
	limit.Cur = 1 << 20
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Skipf("the file size cannot be limited: %v", err)
	}
	before, canCount := countDescriptors()
	err := files.Write(t.Context(), "vault-a", make([]byte, 2<<20))
	after, _ := countDescriptors()
	if restoreErr := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); restoreErr != nil {
		t.Fatal(restoreErr)
	}

	if err == nil {
		t.Fatal("wrote more than the limit allows")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("the folder holds %v", entries)
	}
	if canCount && after != before {
		t.Errorf("descriptors open: %d before, %d after", before, after)
	}
}

func countDescriptors() (int, bool) {
	entries, err := os.ReadDir("/proc/self/fd")
	return len(entries), err == nil
}
