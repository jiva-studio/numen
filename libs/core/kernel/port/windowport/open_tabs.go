package windowport

import (
	"context"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// OpenTabsStore keeps the tabs a window has open on a vault across restarts.
type OpenTabsStore interface {
	// ReadOpenTabs returns the tabs last written for the vault, and none where
	// nothing was written.
	ReadOpenTabs(ctx context.Context, v domain.Vault) (windowdomain.OpenTabs, error)
	// WriteOpenTabs replaces the tabs kept for the vault.
	WriteOpenTabs(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error
}
