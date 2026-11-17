package atomicfile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"slices"
	"testing"
	"time"
)

func TestGetWaitsReturnsTheRetrySchedule(t *testing.T) {
	want := []time.Duration{10 * time.Millisecond, 40 * time.Millisecond, 160 * time.Millisecond}
	if got := getWaits(); !slices.Equal(got, want) {
		t.Fatalf("getWaits() = %v, want %v", got, want)
	}
}

func TestRenameMovesFileAndReportsMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/source.txt", []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})

	if err := Rename(t.Context(), root, "source.txt", "target.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := Rename(t.Context(), root, "missing.txt", "target.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Rename missing source = %v, want %v", err, fs.ErrNotExist)
	}
}

func TestRetryRenameRetriesBusyErrorsUntilSuccessOrLimit(t *testing.T) {
	errBusy := errors.New("file in use")
	errOther := errors.New("permission denied")
	isBusy := func(err error) bool { return errors.Is(err, errBusy) }
	waits := []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}

	attempts := 0
	err := retryRename(t.Context(), func() error {
		attempts++
		if attempts < 3 {
			return errBusy
		}
		return nil
	}, isBusy, waits)
	if err != nil || attempts != 3 {
		t.Fatalf("retryRename err = %v after %d attempts, want nil after 3", err, attempts)
	}

	attempts = 0
	err = retryRename(t.Context(), func() error {
		attempts++
		return errBusy
	}, isBusy, waits)
	if !errors.Is(err, errBusy) || attempts != 4 {
		t.Fatalf("retryRename exhausted err = %v after %d attempts, want %v after 4", err, attempts, errBusy)
	}

	attempts = 0
	err = retryRename(t.Context(), func() error {
		attempts++
		return errOther
	}, isBusy, waits)
	if !errors.Is(err, errOther) || attempts != 1 {
		t.Fatalf("retryRename non-busy err = %v after %d attempts, want %v after 1", err, attempts, errOther)
	}
}

func TestRetryRenameStopsWhenContextIsCanceled(t *testing.T) {
	errBusy := errors.New("file in use")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	attempts := 0
	err := retryRename(ctx, func() error {
		attempts++
		return errBusy
	}, func(err error) bool { return errors.Is(err, errBusy) }, []time.Duration{time.Hour})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("retryRename canceled err = %v after %d attempts, want %v after 1", err, attempts, context.Canceled)
	}
}
