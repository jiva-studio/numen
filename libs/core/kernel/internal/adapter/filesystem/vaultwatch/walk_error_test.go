package vaultwatch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestAFolderTheWalkCouldNotEnterIsTheWholeVault. The shape does not distinguish a
// removed folder from a removed file, so a subtree the walk could not enter is a
// case the watch cannot resolve: the folder would later be removed and read as a
// file, and the index would keep returning the notes it contained.
func TestAFolderTheWalkCouldNotEnterIsTheWholeVault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows folders have no permission bits to remove")
	}
	if os.Geteuid() == 0 {
		t.Skip("root enters every folder")
	}
	root, shape := makeVaultAndShape(t)

	// The folder arrives whole, with one subfolder nothing may enter.
	pos := writeFolderOfNotes(t, root, "library", 1)
	shutPath := filepath.Join(pos, "shut")
	if err := os.MkdirAll(filepath.Join(shutPath, "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shutPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(shutPath, 0o755); err != nil {
			t.Error(err)
		}
	})

	if _, whole := shape.GetPathsUnder(t.Context(), pos); !whole {
		t.Fatal("a folder the walk could not enter was not answered as the whole vault")
	}
	if shape.IsKnownFolder("library/shut/deep") {
		t.Error("a folder behind one that could not be entered is in the shape")
	}
}
