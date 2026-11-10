package filetext

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FollowRenames moves the index rows of files that were removed to the files added with the same content.
type FollowRenames struct {
	lookup  sourceport.RemovalLookup
	sources sourceport.SourceRepository
}

// NewFollowRenames creates FollowRenames over what the index holds about sources and the repository that
// moves their rows.
func NewFollowRenames(lookup sourceport.RemovalLookup, sources sourceport.SourceRepository) FollowRenames {
	return FollowRenames{lookup: lookup, sources: sources}
}

// Execute moves each source under the removed paths to the added path with the same content, and returns
// the paths it moved from.
func (f FollowRenames) Execute(ctx context.Context, reader vaultport.VaultReader, vaultID domain.VaultID, removed, added []string) ([]string, error) {
	found, err := findRemovedSources(ctx, f.lookup, vaultID, removed)
	if err != nil {
		return nil, err
	}
	return moveRenamedSources(ctx, f.sources, reader, vaultID, found, added)
}

// findRemovedSources is the sources other than notes the index holds at and under the paths, with the
// content hash each was read with. A source never read has no hash and matches nothing.
func findRemovedSources(
	ctx context.Context, queries sourceport.RemovalLookup, vaultID domain.VaultID, paths []string,
) ([]source.Text, error) {
	var out []source.Text
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
			if ok && text.ContentHash != "" {
				out = append(out, text)
			}
		}
	}
	return out, nil
}

// moveRenamedSources moves each removed source to the path among added whose file holds the same content,
// and returns the paths it moved from. Only a file of the same size as a removed source is read.
func moveRenamedSources(
	ctx context.Context,
	sources sourceport.SourceRepository,
	reader vaultport.VaultReader,
	vaultID domain.VaultID,
	removed []source.Text,
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

func findSameContent(ctx context.Context, reader vaultport.VaultReader, path string, removed []source.Text) int {
	ref, err := reader.StatFile(ctx, path)
	if err != nil || !slices.ContainsFunc(removed, func(old source.Text) bool { return old.Size == ref.Size }) {
		return -1
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return -1
	}
	hash := domain.HashContent(raw)
	return slices.IndexFunc(removed, func(old source.Text) bool { return old.ContentHash == hash })
}
