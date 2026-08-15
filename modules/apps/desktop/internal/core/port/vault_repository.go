package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultRepository holds the vaults the index knows about.
type VaultRepository interface {
	Save(ctx context.Context, v domain.Vault) error
}
