package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NoteHeads reads the type and the frontmatter of notes from the index, never from their files.
type NoteHeads interface {
	// GetHeads returns the heads of the notes at the paths that the index holds; a path with no
	// indexed note is absent from the result.
	GetHeads(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]frontmatter.Head, error)
}
