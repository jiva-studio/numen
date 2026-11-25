package synctest

import (
	"context"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ForgedSignatures is a server that serves every revision with a signature that is not the one of its author.
type ForgedSignatures struct {
	ports.Revisions
}

// GetChanges returns the changes of the server with one byte of each signature flipped. The server answers over the
// wire, so the answer is this call's own and is changed in place.
func (f ForgedSignatures) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	response, err := f.Revisions.GetChanges(ctx, vaultID, sinceSeq, limit)
	if err != nil {
		return nil, err
	}
	for _, stored := range response.GetChanges() {
		if signature := stored.GetRevision().GetSignature(); len(signature) > 0 {
			signature[0] ^= 0xff
		}
	}
	return response, nil
}
