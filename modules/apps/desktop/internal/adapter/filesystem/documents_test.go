package filesystem_test

import (
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
)

// Where a person keeps their documents is asked of the system they are on, and
// the answer is a place, whatever the system said.
func TestDocumentsIsAPlace(t *testing.T) {
	named, err := filesystem.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if named == "" {
		t.Fatal("no folder was named")
	}
	if !filepath.IsAbs(named) {
		t.Errorf("%q is not somewhere", named)
	}
	if filepath.Clean(named) != named {
		t.Errorf("%q is not written the way a path is", named)
	}
}
