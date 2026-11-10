package indexport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// IndexStats reads what the index holds, for a view of the index. It only reads, so it never waits
// for a writer.
type IndexStats interface {
	// GetIndexCounts returns the counts of the vault. A vault the index does not know has none.
	GetIndexCounts(ctx context.Context, vaultID domain.VaultID, recipe string) (index.Counts, error)
	// GetIndexSizes returns the sizes of the index. Reading the size of a table walks its pages, so
	// the caller does not ask often.
	GetIndexSizes(ctx context.Context) (index.Sizes, error)
}
