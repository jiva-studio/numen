//go:build unix

package filesystem_test

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
)

// capped holds every file this process writes to a length, and hands back what
// lifts it again. It is what a full disk and a quota both do to one append.
func capped(t *testing.T, to int64) func() {
	t.Helper()

	// A write that reaches the cap raises SIGXFSZ, and what the write itself
	// returns is what this is about.
	signal.Ignore(syscall.SIGXFSZ)

	var was syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &was); err != nil {
		t.Skipf("this machine does not cap the size of a file: %v", err)
	}
	if err := syscall.Setrlimit(
		syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: uint64(to), Max: was.Max},
	); err != nil {
		t.Skipf("this machine does not cap the size of a file: %v", err)
	}

	lifted := false
	lift := func() {
		if lifted {
			return
		}
		lifted = true
		_ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &was)
		signal.Reset(syscall.SIGXFSZ)
	}
	t.Cleanup(lift)
	return lift
}

// An append that does not land whole is taken back, so that the next one writes
// a line of its own and the two are not read as one.
func TestAnAppendThatLandsShortIsTakenBack(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()
	at := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir, "run.txt")

	// The cap is on every file this process writes, so what stands under it is
	// larger than anything else the run has open.
	first := strings.Repeat("a", 1<<20) + "\n"
	if err := derived.Append(ctx, "ocr/run.txt", []byte(first)); err != nil {
		t.Fatal(err)
	}

	lift := capped(t, int64(len(first))+10)
	if err := derived.Append(ctx, "ocr/run.txt", []byte(strings.Repeat("b", 4096)+"\n")); err == nil {
		t.Fatal("an append that could not land said it had")
	}
	if got, err := os.ReadFile(at); err != nil || string(got) != first {
		t.Fatalf("the file holds %q, want what stood there before the append: %v", got, err)
	}
	lift()

	third := strings.Repeat("c", 40) + "\n"
	if err := derived.Append(ctx, "ocr/run.txt", []byte(third)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != first+third {
		t.Errorf("the file holds %q, want the two whole lines", got)
	}
}
