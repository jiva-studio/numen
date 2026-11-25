package client

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// RestoreClient calls the restore service of one server with the bearer of the session.
type RestoreClient struct {
	rpc     syncv1connect.RestoreServiceClient
	roots   []ed25519.PublicKey
	now     func() time.Time
	serials epochtrust.Serials
}

// WithSerials makes the client refuse a certificate with a lower serial than the highest one the log holds, and keep a higher one.
func (c *RestoreClient) WithSerials(serials epochtrust.Serials) { c.serials = serials }

// NewRestoreClient returns the client of the server at address. token returns the session token and is read
// again for each call. roots are the keys that may sign the certificate of the epoch key; now is the clock the
// certificate is checked against.
func NewRestoreClient(address string, httpClient *http.Client, token func() string, roots []ed25519.PublicKey, now func() time.Time) *RestoreClient {
	return &RestoreClient{rpc: syncv1connect.NewRestoreServiceClient(httpClient, address, newClientOptions(address, token)), roots: roots, now: now}
}

// GetRestoreEpoch returns the current epoch, nil before the first restore, and the epoch key of the server. The
// key is returned only when a root key signed its certificate, the certificate is valid now and the epoch is
// signed by that key. Otherwise the error wraps epochtrust.ErrUntrusted and nothing the server sent is used.
func (c *RestoreClient) GetRestoreEpoch(ctx context.Context) (*syncv1.RestoreEpoch, []byte, error) {
	response, err := c.rpc.GetRestoreEpoch(ctx, connect.NewRequest(&syncv1.GetRestoreEpochRequest{}))
	if err != nil {
		return nil, nil, syncprotocol.ReadCallError(err)
	}
	cert := epochtrust.Cert{
		EpochKey: response.Msg.GetEpochSignKey(), Serial: response.Msg.GetEpochSerial(), NotBefore: response.Msg.GetEpochNotBefore(),
		NotAfter: response.Msg.GetEpochNotAfter(), RootSignature: response.Msg.GetRootSignature(),
	}
	if err := epochtrust.VerifyCert(c.roots, cert, c.now()); err != nil {
		return nil, nil, err
	}
	if c.serials != nil {
		if err := epochtrust.CheckSerial(ctx, c.serials, cert); err != nil {
			return nil, nil, err
		}
	}
	epoch := response.Msg.GetEpoch()
	if epoch == nil {
		return nil, cert.EpochKey, nil
	}
	if err := epochtrust.VerifyAnnouncement(cert, epoch); err != nil {
		return nil, nil, err
	}
	if c.serials != nil {
		if err := epochtrust.CheckIssuedAt(ctx, c.serials, epoch); err != nil {
			return nil, nil, err
		}
	}
	return epoch, cert.EpochKey, nil
}
