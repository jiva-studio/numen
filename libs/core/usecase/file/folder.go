package file

import "github.com/jiva-studio/numen/libs/notes/domain"

// Folder is the entries of the vault with the type of each note among them, where an index provides it.
type Folder struct {
	Entries []domain.Entry
	Types   map[string]domain.NoteType
}
