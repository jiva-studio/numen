package indexport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ExtensionCount is the sources of one file extension and the chunks that can carry a vector among
// the chunks they hold.
type ExtensionCount struct {
	Extension string
	Sources   int64
	Chunks    int64
}

// IndexCounts is what the index holds for one vault under one recipe.
type IndexCounts struct {
	Chunks     int64
	Vectors    int64
	Sources    int64
	Extensions []ExtensionCount
}

// IndexSizes is the space the index takes on disk, in bytes.
type IndexSizes struct {
	Database int64
	Vectors  int64
}

// IndexStats reads what the index holds, for a view of the index. It only reads, so it never waits
// for a writer.
type IndexStats interface {
	// GetIndexCounts returns the counts of the vault. A vault the index does not know has none.
	GetIndexCounts(ctx context.Context, vaultID domain.VaultID, recipe string) (IndexCounts, error)
	// GetIndexSizes returns the sizes of the index. Reading the size of a table walks its pages, so
	// the caller does not ask often.
	GetIndexSizes(ctx context.Context) (IndexSizes, error)
}
