package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
)

// FrontmatterQueries reads the stored frontmatter of the notes of a vault.
type FrontmatterQueries interface {
	// ListFrontmatter returns the notes the query names that the index holds, in path order.
	ListFrontmatter(ctx context.Context, vaultID domain.VaultID, q frontmatter.Query) ([]frontmatter.Note, error)
}

// NoteHeads reads the type and the frontmatter of notes from the index, never from their files.
type NoteHeads interface {
	// GetHeads returns the heads of the notes at the paths that the index holds; a path with no
	// indexed note is absent from the result.
	GetHeads(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]frontmatter.Head, error)
}
