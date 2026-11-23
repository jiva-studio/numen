package shell

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ClearIndexFailure releases the reason why a vault could not be indexed. A vault that moved
// underneath the window is indexed again.
func (a *API) ClearIndexFailure(vaultID domain.VaultID) {
	a.Runs.Clear(vaultID)
}

// EnsureIndexed brings a vault into the index when it is missing and reports its state, which
// is the state a plugin counting the vault receives.
func (a *API) EnsureIndexed(ctx context.Context, v domain.Vault) (plugin.IndexState, error) {
	return a.Ensure.Execute(ctx, v)
}

// ReportVaultRead is what a window does each time a vault is read into the index: the failure it showed
// for the vault is released and a change is reported.
func (a *API) ReportVaultRead(v domain.Vault) {
	a.ClearIndexFailure(v.ID)
	a.Changes.ReportChange()
}
