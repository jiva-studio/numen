//go:build linux

package appstate_test

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
)

// A folder the write makes is open to everyone to read and enter.
func TestAMadeFolderIsOpenToEveryone(t *testing.T) {
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })

	dir := filepath.Join(t.TempDir(), "flashcards")
	if err := appstate.SchedulesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("mode %v", got)
	}
}

// A write that runs out of room fails, leaves no temporary file and keeps no
// descriptor open.
func TestAWriteThatRunsOutOfRoomLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	kept := appstate.SchedulesAt(dir)

	signal.Ignore(syscall.SIGXFSZ)
	t.Cleanup(func() { signal.Reset(syscall.SIGXFSZ) })
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Fatal(err)
	}
	limit := old
	limit.Cur = 8
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Skipf("the file size cannot be limited: %v", err)
	}
	before := countDescriptors(t)
	err := kept.Write(t.Context(), "vault-a", make([]byte, 4096))
	after := countDescriptors(t)
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
	if after != before {
		t.Errorf("descriptors open: %d before, %d after", before, after)
	}
}

func countDescriptors(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("descriptors cannot be listed: %v", err)
	}
	return len(entries)
}
