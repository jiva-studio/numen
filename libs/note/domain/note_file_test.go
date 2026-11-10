package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// A file is a note by its extension alone, whatever the case of the letters and the folder it is in.
func TestANoteFileIsRecognizedByItsExtension(t *testing.T) {
	for path, want := range map[string]bool{
		"Atlas.md":        true,
		"harbor/Atlas.MD": true,
		"harbor/atlas.Md": true,
		"Atlas.md.png":    false,
		"Atlas.txt":       false,
		"harbor/md":       false,
		"harbor.md/Atlas": false,
		"":                false,
	} {
		if got := domain.IsNoteFile(path); got != want {
			t.Errorf("IsNoteFile(%q) = %v, want %v", path, got, want)
		}
	}
}
