package vaultwatch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
)

// TestAShapeShortOfAFolderComesBackAsAnError. The shape distinguishes a folder
// that has gone from a file that has, so a folder the walk could not enter is a
// condition the watch cannot resolve, and it reports an error.
func TestAShapeShortOfAFolderReachesBackAsAnError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows folders have no permission bits to remove")
	}
	if os.Geteuid() == 0 {
		t.Skip("root enters every folder")
	}
	root := t.TempDir()
	shutPath := filepath.Join(root, "shut")
	if err := os.MkdirAll(filepath.Join(shutPath, "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shutPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shutPath, 0o755) })

	reader, err := vaultreader.Open(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	shape, why := vaultreader.ReadShape(reader)
	if why == nil {
		t.Fatal("a folder the walk could not enter was not reported")
	}
	if shape.Indexed["shut/deep"] {
		t.Error("a folder behind one that could not be entered is in the shape")
	}
}
