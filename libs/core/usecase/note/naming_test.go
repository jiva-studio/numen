package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
)

// One rule answers for both halves of a name, at either setting and for each of
// the two that name a note.
func TestWhatARenameRestoresIntoLine(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		sync      note.SyncTitleAndFilename
		delta     note.NameSource
		isMoving  bool
		isWriting bool
	}{
		"one name, a title in the frontmatter": {
			sync: true, delta: note.ByFrontmatter, isMoving: true, isWriting: true,
		},
		"one name, and the filename says it": {
			sync: true, delta: note.ByFilename, isMoving: true, isWriting: false,
		},
		"told apart, a title in the frontmatter": {
			sync: false, delta: note.ByFrontmatter, isMoving: false, isWriting: false,
		},
		"told apart, and the filename says it": {
			sync: false, delta: note.ByFilename, isMoving: true, isWriting: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			moves, writes := c.sync.GetRenameEffects(c.delta)
			if moves != c.isMoving {
				t.Errorf("the file moves: %v", moves)
			}
			if writes != c.isWriting {
				t.Errorf("the note is written: %v", writes)
			}
		})
	}
}

// A setting that is asked is answered as it says, whichever way it says it.
func TestASettingIsAnsweredAsItIsAsked(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]note.SyncTitleAndFilename{"one name": true, "told apart": false} {
		ask := note.SyncSetting(func() note.SyncTitleAndFilename { return want })
		if got := ask.GetSetting(); got != want {
			t.Errorf("%s: got %v", name, got)
		}
	}
	var unasked note.SyncSetting
	if !unasked.GetSetting() {
		t.Error("nothing asked keeps the two one name")
	}
}
