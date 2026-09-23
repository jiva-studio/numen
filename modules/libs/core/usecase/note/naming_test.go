package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// One rule answers for both halves of a name, at either setting and for each of
// the two that name a note.
func TestWhatARenameBringsIntoLine(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		sync      note.SyncTitleAndFilename
		by        note.NameSource
		isMoving  bool
		isWriting bool
	}{
		"one name, a title in the frontmatter": {
			sync: true, by: note.ByFrontmatter, isMoving: true, isWriting: true,
		},
		"one name, and the filename says it": {
			sync: true, by: note.ByFilename, isMoving: true, isWriting: false,
		},
		"told apart, a title in the frontmatter": {
			sync: false, by: note.ByFrontmatter, isMoving: false, isWriting: false,
		},
		"told apart, and the filename says it": {
			sync: false, by: note.ByFilename, isMoving: true, isWriting: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			moves, writes := c.sync.GetRenameEffects(c.by)
			if moves != c.isMoving {
				t.Errorf("the file moves: %v", moves)
			}
			if writes != c.isWriting {
				t.Errorf("the note is written: %v", writes)
			}
		})
	}
}
