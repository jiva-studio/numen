package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
)

// FileChangeFeed reads the file change feed of a vault.
type FileChangeFeed interface {
	// GetHead returns the state of the feed of the vault, the zero head for a vault without one.
	GetHead(ctx context.Context, vaultID domain.VaultID) (filechange.Head, error)
	// ListFileChanges returns the changes of the events numbered after after and up to upTo, at most
	// limit, in the order they were stored.
	ListFileChanges(ctx context.Context, vaultID domain.VaultID, events []string, after, upTo int64, limit int) ([]filechange.Change, error)
}
