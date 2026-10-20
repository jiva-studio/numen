package editor

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The use cases a handler holds, for the tests that wrap one in a fake.

func (s *Services) NoteCreate() *note.Create { return s.Note.create }
func (s *Services) NoteWrite() *note.Write   { return s.Note.write }
func (s *Services) NoteRename() *note.Rename { return s.Note.rename }
func (s *Services) FileRemove() *note.Remove { return s.File.remove }
func (s *Services) FileMove() *vaults.Move   { return s.File.move }

func (s *Services) SetFileMove(move *vaults.Move)  { s.File.move = move }
func (s *Services) SetNoteWrite(write *note.Write) { s.Note.write = write }

// NewWireModels is the models as the wire carries them.
var NewWireModels = newWireModels
