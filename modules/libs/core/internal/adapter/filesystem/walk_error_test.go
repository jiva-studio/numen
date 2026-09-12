package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAFolderTheWalkCouldNotEnterIsTheWholeVault. What the shape does not hold
// is what tells a folder that has gone from a file that has, so a subtree the
// walk could not enter is a question the watch can no longer answer: the folder
// would later leave and be read as a file, and the index would go on answering
// with the notes it took with it.
func TestAFolderTheWalkCouldNotEnterIsTheWholeVault(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters every folder")
	}
	root, shape := makeVaultAndShape(t)

	// The folder arrives whole, with one subfolder nothing may enter.
	at := writeFolderOfNotes(t, root, "library", 1)
	shut := filepath.Join(at, "shut")
	if err := os.MkdirAll(filepath.Join(shut, "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shut, 0o755) })

	if _, whole := shape.inside(t.Context(), at); !whole {
		t.Fatal("a folder the walk could not enter was not answered as the whole vault")
	}
	if shape.knows("library/shut/deep") {
		t.Error("a folder behind one that could not be entered is in the shape")
	}
}
