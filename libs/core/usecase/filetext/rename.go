package filetext

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// A file renamed outside the core is seen as its old path removed and its new
// path added. Moving the source keeps its chunks and text versions.

// FindRemovedSources is the sources other than notes the index holds at and
// under the paths, with the content hash each was read with. A source never
// read has no hash and matches nothing.
func FindRemovedSources(
	ctx context.Context, queries sourceport.SourceQueries, vaultID domain.VaultID, paths []string,
) ([]sourceport.SourceText, error) {
	var out []sourceport.SourceText
	for _, path := range paths {
		under, err := queries.GetSourcesUnder(ctx, vaultID, path)
		if err != nil {
			return nil, err
		}
		for _, ref := range under {
			if ref.Kind == domain.KindNote {
				continue
			}
			text, ok, err := queries.GetSourceText(ctx, vaultID, ref.Path)
			if err != nil {
				return nil, err
			}
			if ok && text.Hash != "" {
				out = append(out, text)
			}
		}
	}
	return out, nil
}

// MoveRenamedSources moves each removed source to the path among added whose
// file holds the same content, and returns the paths it moved from. Only a
// file of the same size as a removed source is read.
func MoveRenamedSources(
	ctx context.Context,
	sources sourceport.SourceRepository,
	reader vaultport.VaultReader,
	vaultID domain.VaultID,
	removed []sourceport.SourceText,
	added []string,
) ([]string, error) {
	var moved []string
	for _, path := range added {
		if len(removed) == 0 {
			break
		}
		i := findSameContent(ctx, reader, path, removed)
		if i < 0 {
			continue
		}
		if err := sources.MoveSources(ctx, vaultID, removed[i].Path, path); err != nil {
			return moved, err
		}
		moved = append(moved, removed[i].Path)
		removed = slices.Delete(removed, i, i+1)
	}
	return moved, nil
}

// findSameContent is the index of the removed source whose content the file at
// path holds, and -1 where none does or the file cannot be read.
func findSameContent(ctx context.Context, reader vaultport.VaultReader, path string, removed []sourceport.SourceText) int {
	ref, err := reader.StatFile(ctx, path)
	if err != nil || !slices.ContainsFunc(removed, func(old sourceport.SourceText) bool { return old.Size == ref.Size }) {
		return -1
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return -1
	}
	hash := domain.HashContent(raw)
	return slices.IndexFunc(removed, func(old sourceport.SourceText) bool { return old.Hash == hash })
}
