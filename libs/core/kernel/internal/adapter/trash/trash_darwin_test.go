//go:build darwin

package trash

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestMoveToTrashFromManyGoroutines trashes folders from many goroutines at once, the way several
// vaults are erased while the runtime moves goroutines between threads.
func TestMoveToTrashFromManyGoroutines(t *testing.T) {
	const callers = 16
	folders := make([]string, callers)
	for i := range folders {
		folders[i] = filepath.Join(t.TempDir(), "vault")
		if err := os.MkdirAll(folders[i], 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folders[i], "note.md"), []byte("hello"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	failures := make([]error, callers)
	for i, pos := range folders {
		wg.Go(func() {
			failures[i] = moveToTrash(pos, time.Time{})
		})
	}
	wg.Wait()

	for i, err := range failures {
		if err != nil && !errors.Is(err, machine.ErrNoTrash) {
			t.Errorf("folder %d: %v", i, err)
			continue
		}
		if err == nil {
			if _, statErr := os.Stat(folders[i]); !os.IsNotExist(statErr) {
				t.Errorf("folder %d is still in place after it was trashed", i)
			}
		}
	}
}
