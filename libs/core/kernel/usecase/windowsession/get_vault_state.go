package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// CoverageReader reads what the search source tells a person about how far it has indexed a vault.
type CoverageReader interface {
	DescribeCoverage(ctx context.Context, vaultID domain.VaultID) (string, error)
}

// GetVaultState reads what the window knows about the vault it shows: whether the first scan is done,
// what stopped it, and what the search source says of its index.
func GetVaultState(ctx context.Context, rt *Runtime, coverage CoverageReader) (window.VaultState, error) {
	showing := rt.GetCurrentVault()
	out := window.VaultState{
		Vault:      showing,
		IsReady:    rt.IsReady(),
		Error:      rt.GetError(),
		WatchError: rt.GetWatchError(),
	}
	if coverage != nil && showing.ID != "" {
		text, err := coverage.DescribeCoverage(ctx, showing.ID)
		if err != nil {
			return out, err
		}
		out.Coverage = text
	}
	return out, nil
}
