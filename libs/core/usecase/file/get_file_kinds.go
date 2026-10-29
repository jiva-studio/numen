package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// GetFileKinds returns the files in the requested order, with the type of each note. The type is the
// index's, and a note the index does not hold yet, such as one a plugin has just written, is read
// directly. A path with nothing at it is omitted, and each entry carries its path and kind.
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
	refs := make(map[string]domain.FileStat, len(paths))
	for _, path := range paths {
		ref, err := reader.StatFile(ctx, path)
		if errors.Is(err, vaultport.ErrNotExist) || errors.Is(err, vaultport.ErrOutside) {
			continue
		}
		// A file the vault ignores is present and is not a source.
		if err != nil && !errors.Is(err, vaultport.ErrNotANote) {
			return Folder{}, err
		}
		entries = append(entries, domain.Entry{Path: path, Kind: ref.Kind})
		refs[path] = ref
	}
	types, err := getNoteTypes(ctx, u.queries, v, entries)
	if err != nil {
		return Folder{}, err
	}
	types, err = readMissingTypes(ctx, reader, entries, refs, types)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Entries: entries, Types: types}, nil
}

// readMissingTypes adds the type of each note the types leave out, read from the note itself.
func readMissingTypes(
	ctx context.Context, reader vaultport.VaultReader, entries []domain.Entry,
	refs map[string]domain.FileStat, types map[string]domain.NoteType,
) (map[string]domain.NoteType, error) {
	for _, entry := range entries {
		if entry.Kind != domain.KindNote {
			continue
		}
		if _, found := types[entry.Path]; found {
			continue
		}
		raw, err := reader.Read(ctx, entry.Path)
		if errors.Is(err, vaultport.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if types == nil {
			types = make(map[string]domain.NoteType)
		}
		types[entry.Path] = markdown.Parse(refs[entry.Path], raw).Type
	}
	return types, nil
}
