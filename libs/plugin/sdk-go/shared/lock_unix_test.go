//go:build unix

package shared

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLockFileRetriesAnInterruptedLockAndReportsAFailedOne(t *testing.T) {
	var calls []int
	flock := func(_, how int) error {
		calls = append(calls, how)
		if len(calls) == 1 {
			return syscall.EINTR
		}
		return syscall.ENOLCK
	}

	release, err := lockFileWith(filepath.Join(t.TempDir(), "lock"), flock)
	if !errors.Is(err, syscall.ENOLCK) || release != nil {
		t.Fatalf("lockFileWith = %v, %v, want no release and ENOLCK", release != nil, err)
	}
	if len(calls) != 2 || calls[0] != syscall.LOCK_EX || calls[1] != syscall.LOCK_EX {
		t.Errorf("flock calls = %v, want two exclusive locks", calls)
	}
}
