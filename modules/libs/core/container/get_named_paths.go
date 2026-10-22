package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

func (Config) GetNamedPaths(queries noteport.NoteQueries) *note.GetNamedPaths {
	return note.NewGetNamedPaths(queries)
}
