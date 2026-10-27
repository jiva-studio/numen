package windowport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// OpenTabsStore keeps the tabs a window has open on a vault across restarts.
type OpenTabsStore interface {
	// ReadOpenTabs returns the tabs last written for the vault, and none where
	// nothing was written.
	ReadOpenTabs(ctx context.Context, v domain.Vault) (domain.OpenTabs, error)
	// WriteOpenTabs replaces the tabs kept for the vault.
	WriteOpenTabs(ctx context.Context, v domain.Vault, open domain.OpenTabs) error
}
