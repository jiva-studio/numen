package synctest

import (
	"context"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

// EpochClaim is a server that puts a restore epoch id of its own choice into every page of changes.
type EpochClaim struct {
	ports.Revisions
	ID []byte
}

// GetChanges returns the changes of the server with the claimed epoch id. The answer is this call's own and is changed in place.
func (e EpochClaim) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	response, err := e.Revisions.GetChanges(ctx, vaultID, sinceSeq, limit)
	if err != nil {
		return nil, err
	}
	response.RestoreEpochId = e.ID
	return response, nil
}

// NewSyncVaultOver returns the pass over one vault with the changes read from revisions instead of the server of this computer.
func (c *Computer) NewSyncVaultOver(revisions ports.Revisions) *syncvault.SyncVault {
	wired := c.WireDeps()
	wired.Revisions = revisions
	return syncwire.NewSyncVault(wired)
}
