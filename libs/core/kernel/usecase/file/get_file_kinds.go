package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// GetFileKinds returns the files in the requested order, with the type and the frontmatter of each
// note. The type is the index's, and a note the index does not hold yet, such as one a plugin created
// in this session, gets the type its file declares. A path with nothing at it is omitted, and each
// entry carries its path and kind.
type GetFileKinds struct {
	readers vaultport.VaultReaders
	queries noteport.NoteTypes
}

func NewGetFileKinds(readers vaultport.VaultReaders, queries noteport.NoteTypes) *GetFileKinds {
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
		if errors.Is(err, edit.ErrNotExist) || errors.Is(err, vault.ErrOutside) {
			continue
		}
		// A file the vault ignores is present and is not a source.
		if err != nil && !errors.Is(err, edit.ErrNotANote) {
			return Folder{}, err
		}
		entries = append(entries, domain.Entry{Path: path, Kind: ref.Kind})
		refs[path] = ref
	}
	types, err := getNoteTypes(ctx, u.queries, v, entries)
	if err != nil {
		return Folder{}, err
	}
	properties, err := readNotes(ctx, reader, entries, refs, &types)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Entries: entries, Types: types, Properties: properties}, nil
}

func readNotes(
	ctx context.Context, reader vaultport.VaultReader, entries []domain.Entry,
	refs map[string]domain.FileStat, types *map[string]domain.NoteType,
) (map[string]map[string]any, error) {
	properties := map[string]map[string]any{}
	for _, entry := range entries {
		if entry.Kind != domain.KindNote {
			continue
		}
		raw, err := reader.Read(ctx, entry.Path)
		if errors.Is(err, edit.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		note := markdown.Parse(refs[entry.Path], raw)
		properties[entry.Path] = note.Frontmatter
		if _, found := (*types)[entry.Path]; found {
			continue
		}
		if *types == nil {
			*types = make(map[string]domain.NoteType)
		}
		(*types)[entry.Path] = note.Type
	}
	return properties, nil
}
