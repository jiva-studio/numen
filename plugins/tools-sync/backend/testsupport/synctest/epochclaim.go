package synctest

import (
	"context"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

type EpochClaim struct {
	ports.Revisions
	ID []byte
}

func (e EpochClaim) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	response, err := e.Revisions.GetChanges(ctx, vaultID, sinceSeq, limit)
	if err != nil {
		return nil, err
	}
	response.RestoreEpochId = e.ID
	return response, nil
}

func (c *Computer) NewSyncVaultOver(revisions ports.Revisions) *syncvault.SyncVault {
	wired := c.WireDeps()
	wired.Revisions = revisions
	return syncwire.NewSyncVault(wired)
}
