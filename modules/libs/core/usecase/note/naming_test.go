package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// One rule answers for both halves of a name, at either setting and for each of
// the two that name a note.
func TestWhatARenameBringsIntoLine(t *testing.T) {
	for name, c := range map[string]struct {
		sync   note.Sync
		by     note.Naming
		moves  bool
		writes bool
	}{
		"one name, a title in the frontmatter": {
			sync: true, by: note.ByFrontmatter, moves: true, writes: true,
		},
		"one name, and the filename says it": {
			sync: true, by: note.ByFilename, moves: true, writes: false,
		},
		"told apart, a title in the frontmatter": {
			sync: false, by: note.ByFrontmatter, moves: false, writes: false,
		},
		"told apart, and the filename says it": {
			sync: false, by: note.ByFilename, moves: true, writes: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			moves, writes := c.sync.Renaming(c.by)
			if moves != c.moves {
				t.Errorf("the file moves: %v", moves)
			}
			if writes != c.writes {
				t.Errorf("the note is written: %v", writes)
			}
		})
	}
}
