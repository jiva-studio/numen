package filetext

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FindRemovedSources is the sources other than notes the index holds at and
// under the paths, with the content hash each was read with.
func FindRemovedSources(ctx context.Context, queries sourceport.SourceQueries, vaultID domain.VaultID, paths []string) ([]sourceport.SourceText, error) {
	return vaultreading.FindRemovedSources(ctx, queries, vaultID, paths)
}

// MoveRenamedSources moves each removed source to the path among added whose
// file holds the same content, and returns the paths it moved from.
func MoveRenamedSources(ctx context.Context, sources sourceport.SourceRepository, reader vaultport.VaultReader, vaultID domain.VaultID, removed []sourceport.SourceText, added []string) ([]string, error) {
	return vaultreading.MoveRenamedSources(ctx, sources, reader, vaultID, removed, added)
}
