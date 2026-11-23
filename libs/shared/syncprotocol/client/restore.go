package client

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// RestoreClient calls the restore service of one server with the bearer of the session.
type RestoreClient struct {
	rpc syncv1connect.RestoreServiceClient
}

// NewRestoreClient returns the client of the server at address. token returns the session token and is read
// again for each call.
func NewRestoreClient(address string, httpClient *http.Client, token func() string) *RestoreClient {
	return &RestoreClient{rpc: syncv1connect.NewRestoreServiceClient(httpClient, address, newClientOptions(token))}
}

// GetRestoreEpoch returns the current epoch, nil before the first restore, and the key of the server that signs
// epochs. An epoch whose signature does not verify against that key is an error.
func (c *RestoreClient) GetRestoreEpoch(ctx context.Context) (*syncv1.RestoreEpoch, []byte, error) {
	response, err := c.rpc.GetRestoreEpoch(ctx, connect.NewRequest(&syncv1.GetRestoreEpochRequest{}))
	if err != nil {
		return nil, nil, syncprotocol.ReadCallError(err)
	}
	key := response.Msg.GetEpochSignKey()
	epoch := response.Msg.GetEpoch()
	if epoch == nil {
		if len(key) != ed25519.PublicKeySize {
			return nil, nil, fmt.Errorf("%w: epoch_sign_key has %d bytes", syncprotocol.ErrInvalidRestoreEpoch, len(key))
		}
		return nil, key, nil
	}
	if err := syncprotocol.CheckRestoreEpoch(key, epoch); err != nil {
		return nil, nil, err
	}
	return epoch, key, nil
}
