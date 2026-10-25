package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// GetFolderEntries returns what a folder holds, with the type of each note in it where an index exists.
// Only the failure to list the folder wraps its cause.
type GetFolderEntries struct {
	readers vaultport.VaultReaders
	queries noteport.NoteQueries
}

func NewGetFolderEntries(readers vaultport.VaultReaders, queries noteport.NoteQueries) *GetFolderEntries {
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
