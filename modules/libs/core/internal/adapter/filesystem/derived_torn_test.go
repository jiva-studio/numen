//go:build unix

package filesystem_test

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
)

// fileCap is the length every file this process writes is held to while the
// torn append is made. It stands above anything else the run has open.
const fileCap = 1 << 20

// capFileSize holds every file this process writes to a length, and hands back what
// lifts it again. It is what a full disk and a quota both do to one append.
func capFileSize(t *testing.T, to int64) func() {
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

	first := strings.Repeat("a", 40) + "\n"
	if err := derived.Append(ctx, "ocr/run.txt", []byte(first)); err != nil {
		t.Fatal(err)
	}

	// The one call has to ask for more than the cap allows. Darwin weighs the
	// cap against the offset the file was opened at, so the length of the
	// append is what reaches past it.
	lift := capFileSize(t, fileCap)
	torn := strings.Repeat("b", 4*fileCap) + "\n"
	err := derived.Append(ctx, "ocr/run.txt", []byte(torn))
	if err == nil {
		t.Fatal("an append that could not land said it had")
	}
	// Bytes have to have landed for the cut to be the thing under test.
	if strings.Contains(err.Error(), " 0 of ") {
		t.Fatalf("the append landed nothing, so nothing was taken back: %v", err)
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
