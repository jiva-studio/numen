package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAShapeShortOfAFolderComesBackAsAnError. The shape is what tells a folder
// that has gone from a file that has, so a folder the walk could not enter is a
// question the watch can no longer answer, and says so.
func TestAShapeShortOfAFolderComesBackAsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters every folder")
	}
	root := t.TempDir()
	shut := filepath.Join(root, "shut")
	if err := os.MkdirAll(filepath.Join(shut, "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shut, 0o755) })

	reader, err := Open(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	shape, why := remembered(reader)
	if why == nil {
		t.Fatal("a folder the walk could not enter was not reported")
	}
	if shape.are["shut/deep"] {
		t.Error("a folder behind one that could not be entered is in the shape")
	}
}
