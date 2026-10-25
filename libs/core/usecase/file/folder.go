package file

import "github.com/jiva-studio/numen/libs/notes/domain"

// Folder is entries of the vault and what each note among them is, where an index says.
type Folder struct {
	Entries []domain.Entry
	Types   map[string]domain.NoteType
}
