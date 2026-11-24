package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// VaultSummaries counts what the index holds of one vault.
type VaultSummaries interface{ GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error) }
