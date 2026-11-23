package client

import (
	"context"
	"errors"
	"net/http"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// ShareClient calls the share service of one server with the bearer of the session.
type ShareClient struct {
	rpc syncv1connect.ShareServiceClient
}

// NewShareClient returns the client of the server at address. token returns the session token and is read again
// for each call.
func NewShareClient(address string, httpClient *http.Client, token func() string) *ShareClient {
	return &ShareClient{rpc: syncv1connect.NewShareServiceClient(httpClient, address, newClientOptions(token))}
}

// AppendShareLogEntry adds an entry to the member log of a share root. When another device was first, the answer
// is not committed and carries the entry at the head.
func (c *ShareClient) AppendShareLogEntry(_ context.Context, _ *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error) {
	return nil, errors.New("not implemented")
}

// ListShares returns the share roots of the account of the session.
func (c *ShareClient) ListShares(_ context.Context) ([]*syncv1.ShareSummary, error) {
	return nil, errors.New("not implemented")
}

// GetShareLog returns every entry after sinceVersion, in order, reading page after page, and the chain of the owner.
func (c *ShareClient) GetShareLog(_ context.Context, _ []byte, _ uint64) ([]*syncv1.ShareLogEntry, []byte, error) {
	return nil, nil, errors.New("not implemented")
}

// GetShareChanges returns one page of the changes of a share root after sinceSeq. A limit of 0 is DefaultPageSize.
func (c *ShareClient) GetShareChanges(_ context.Context, _ []byte, _ uint64, _ uint32) (*syncv1.GetShareChangesResponse, error) {
	return nil, errors.New("not implemented")
}

// GetShareBlobUrls returns one download for each blob id, in the order of the ids.
func (c *ShareClient) GetShareBlobUrls(_ context.Context, _ []byte, _ [][]byte) ([]*syncv1.BlobDownload, error) {
	return nil, errors.New("not implemented")
}
