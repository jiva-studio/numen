package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func TestIsNotePathIsTrueForTheNoteExtensionInAnyCase(t *testing.T) {
	for path, want := range map[string]bool{"Talks/River.md": true, "Talks/River.MD": true, "Data/README": false, "Books/Atlas.pdf": false, "": false} {
		if got := domain.IsNotePath(path); got != want {
			t.Errorf("%q: %v, want %v", path, got, want)
		}
	}
}
