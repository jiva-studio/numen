//go:build unix

package derivedfile_test

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
)

// fileCap is the file size limit under which a partial append occurs. It is
// larger than any other file the run has open.
const fileCap = 1 << 20

// capFileSize holds every file this process writes to a length, and returns what
// lifts it again. It is what a full disk and a quota both do to one append.
func capFileSize(t *testing.T, to int64) func() {
	t.Helper()

	// A write that hits the cap raises SIGXFSZ; the test checks the error the
	// write itself returns.
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
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &was); err != nil {
			t.Error(err)
		}
		signal.Reset(syscall.SIGXFSZ)
	}
	t.Cleanup(lift)
	return lift
}

// An append that does not complete is rolled back, so the next append writes
// its own line and the two are not read as one.
func TestAnAppendThatLandsShortIsTakenBack(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()
	pos := filepath.Join(root, vaultconfig.DefaultServiceDir, testArea, "run.txt")

	first := strings.Repeat("a", 40) + "\n"
	if err := derived.Append(ctx, "area/run.txt", []byte(first)); err != nil {
		t.Fatal(err)
	}

	// The single call must request more than the cap allows. Darwin compares the
	// cap with the offset the file was opened at, so the append length is what
	// exceeds it.
	lift := capFileSize(t, fileCap)
	divided := strings.Repeat("b", 4*fileCap) + "\n"
	err := derived.Append(ctx, "area/run.txt", []byte(divided))
	if err == nil {
		t.Fatal("an append that could not land said it had")
	}
	// Bytes have to have landed for the cut to be the thing under test.
	if strings.Contains(err.Error(), " 0 of ") {
		t.Fatalf("the append landed nothing, so nothing was taken back: %v", err)
	}
	if got, err := os.ReadFile(pos); err != nil || string(got) != first {
		t.Fatalf("the file holds %q, want what stood there before the append: %v", got, err)
	}
	lift()

	third := strings.Repeat("c", 40) + "\n"
	if err := derived.Append(ctx, "area/run.txt", []byte(third)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != first+third {
		t.Errorf("the file holds %q, want the two whole lines", got)
	}
}
