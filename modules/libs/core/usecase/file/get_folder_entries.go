package file

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetFolderEntries returns what a folder holds, with the type of each note in it where an index exists.
// Only the failure to list the folder wraps its cause.
type GetFolderEntries struct {
	readers port.VaultReaders
	queries port.NoteQueries
}

func NewGetFolderEntries(readers port.VaultReaders, queries port.NoteQueries) *GetFolderEntries {
	return &GetFolderEntries{readers: readers, queries: queries}
}

func (u *GetFolderEntries) Execute(ctx context.Context, v domain.Vault, path string) (Folder, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return Folder{}, newVaultFailure(err)
	}
	entries, err := reader.List(ctx, path)
	if err != nil {
		return Folder{}, err
	}
	types, err := getNoteTypes(ctx, u.queries, v, entries)
	if err != nil {
		return Folder{}, newVaultFailure(err)
	}
	return Folder{Entries: entries, Types: types}, nil
}
