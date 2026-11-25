package synctest

import (
	"context"

	"google.golang.org/protobuf/proto"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ForgedSignatures is a server that serves every revision with a signature that is not the one of its author.
type ForgedSignatures struct {
	ports.Revisions
}

// GetChanges returns the changes of the server with one byte of each signature flipped.
func (f ForgedSignatures) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	response, err := f.Revisions.GetChanges(ctx, vaultID, sinceSeq, limit)
	if err != nil {
		return nil, err
	}
	forged, isResponse := proto.Clone(response).(*syncv1.GetChangesResponse)
	if !isResponse {
		return response, nil
	}
	for _, stored := range forged.GetChanges() {
		if signature := stored.GetRevision().GetSignature(); len(signature) > 0 {
			signature[0] ^= 0xff
		}
	}
	return forged, nil
}
