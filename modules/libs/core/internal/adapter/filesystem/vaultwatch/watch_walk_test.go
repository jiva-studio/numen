package vaultwatch_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultfixture"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// TestAFolderTooLargeToWalkIsTheWholeVault. The walk of a folder that has
// arrived runs on the one goroutine that empties the backlog, so a folder past
// what the watch follows through is answered as the whole vault.
func TestAFolderTooLargeToWalkIsTheWholeVault(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{"Note.md": "# Note\n"}, nil)
	// The folder is filled where the watch cannot see it, and arrives whole.
	aside := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(aside, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range vaultreader.Entries + 1 {
		name := filepath.Join(aside, strconv.Itoa(i)+".md")
		if err := os.WriteFile(name, []byte("# "+strconv.Itoa(i)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, missing, err := vaultwatch.Watcher{}.WatchVault(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(aside, filepath.Join(root, "library")); err != nil {
		t.Skipf("the folder could not be moved into the vault whole: %v", err)
	}

	select {
	case <-missing:
	case <-time.After(10 * time.Second):
		t.Fatal("a folder larger than the watch follows through arrived and nothing was said")
	}
}
