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
