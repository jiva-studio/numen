package vaultwatch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestAFolderTheWalkCouldNotEnterIsTheWholeVault. What the shape does not hold
// is what tells a folder that has gone from a file that has, so a subtree the
// walk could not enter is a question the watch can no longer answer: the folder
// would later leave and be read as a file, and the index would go on answering
// with the notes it took with it.
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
	t.Cleanup(func() { _ = os.Chmod(shutPath, 0o755) })

	if _, whole := shape.GetPathsUnder(t.Context(), pos); !whole {
		t.Fatal("a folder the walk could not enter was not answered as the whole vault")
	}
	if shape.IsKnownFolder("library/shut/deep") {
		t.Error("a folder behind one that could not be entered is in the shape")
	}
}
