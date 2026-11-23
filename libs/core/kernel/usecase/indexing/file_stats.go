package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// FileStats reads what the index recorded of each file of a vault, keyed by
// path, so a scan can decide what to reparse without reading anything.
type FileStats interface {
	GetFileStats(ctx context.Context, vaultID domain.VaultID) (map[string]domain.FileStat, error)
}
