package file

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// GetFileKinds returns the files in the order asked, with the type of each note when an index exists.
// A path with nothing at it is left out, and each entry carries its path and kind.
type GetFileKinds struct {
	readers vaultport.VaultReaders
	queries noteport.NoteQueries
}

func NewGetFileKinds(readers vaultport.VaultReaders, queries noteport.NoteQueries) *GetFileKinds {
	return &GetFileKinds{readers: readers, queries: queries}
}

func (u *GetFileKinds) Execute(ctx context.Context, v domain.Vault, paths []string) (Folder, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return Folder{}, err
	}
	entries := make([]domain.Entry, 0, len(paths))
	for _, path := range paths {
		ref, err := reader.GetFingerprint(ctx, path)
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, vaultport.ErrOutside) {
			continue
		}
		// A file the vault leaves alone stands there and is no source.
		if err != nil && !errors.Is(err, vaultport.ErrNotANote) {
			return Folder{}, err
		}
		entries = append(entries, domain.Entry{Path: path, Kind: ref.Kind})
	}
	types, err := getNoteTypes(ctx, u.queries, v, entries)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Entries: entries, Types: types}, nil
}
