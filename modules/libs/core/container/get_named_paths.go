package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func (Config) GetNamedPaths(queries port.NoteQueries) *note.GetNamedPaths {
	return note.NewGetNamedPaths(queries)
}
