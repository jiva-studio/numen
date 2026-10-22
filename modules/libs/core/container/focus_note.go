package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// FocusNote builds what puts a note in front of the person, through their window.
func (c Config) FocusNote(queries noteport.NoteQueries, view windowport.Window) *note.FocusNote {
	return note.NewFocusNote(queries, view)
}
