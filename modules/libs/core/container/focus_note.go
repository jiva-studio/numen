package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// FocusNote builds what puts a note in front of the person, through their window.
func (c Config) FocusNote(queries port.NoteQueries, view port.Window) *note.FocusNote {
	return note.NewFocusNote(queries, view)
}
