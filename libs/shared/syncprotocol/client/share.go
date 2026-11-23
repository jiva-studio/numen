package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
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
	return &ShareClient{rpc: syncv1connect.NewShareServiceClient(httpClient, address, newClientOptions(address, token))}
}

// AppendShareLogEntry adds an entry to the member log of a share root. When another device was first, the answer
// is not committed and carries the entry at the head.
func (c *ShareClient) AppendShareLogEntry(ctx context.Context, entry *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error) {
	response, err := c.rpc.AppendShareLogEntry(ctx, connect.NewRequest(&syncv1.AppendShareLogEntryRequest{Entry: entry}))
	if err != nil {
		return nil, readAppendShareError(entry, err)
	}
	return response.Msg, nil
}

// ListShares returns the share roots of the account of the session.
func (c *ShareClient) ListShares(ctx context.Context) ([]*syncv1.ShareSummary, error) {
	response, err := c.rpc.ListShares(ctx, connect.NewRequest(&syncv1.ListSharesRequest{}))
	if err != nil {
		return nil, syncprotocol.ReadShareError(err)
	}
	return response.Msg.GetShares(), nil
}

// GetShareLog returns every entry after sinceVersion, in order, reading page after page, and the chain of the owner.
func (c *ShareClient) GetShareLog(ctx context.Context, shareRootID []byte, sinceVersion uint64) ([]*syncv1.ShareLogEntry, []byte, error) {
	var entries []*syncv1.ShareLogEntry
	for {
		response, err := c.rpc.GetShareLog(ctx, connect.NewRequest(&syncv1.GetShareLogRequest{ShareRootId: shareRootID, SinceVersion: sinceVersion}))
		if err != nil {
			return nil, nil, syncprotocol.ReadShareError(err)
		}
		page := response.Msg.GetEntries()
		entries = append(entries, page...)
		if !response.Msg.GetHasMore() {
			return entries, response.Msg.GetOwnerChain(), nil
		}
		if len(page) == 0 {
			return nil, nil, fmt.Errorf("the sync server has more share log entries after version %d and sent none", sinceVersion)
		}
		sinceVersion = page[len(page)-1].GetVersion()
	}
}

// GetShareChanges returns one page of the changes of a share root after sinceSeq. A limit of 0 is DefaultPageSize.
func (c *ShareClient) GetShareChanges(ctx context.Context, shareRootID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetShareChangesResponse, error) {
	response, err := c.rpc.GetShareChanges(ctx, connect.NewRequest(&syncv1.GetShareChangesRequest{ShareRootId: shareRootID, SinceSeq: sinceSeq, Limit: limit}))
	if err != nil {
		return nil, syncprotocol.ReadShareError(err)
	}
	return response.Msg, nil
}

func (c *ShareClient) GetShareBlobUrls(ctx context.Context, shareRootID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error) {
	response, err := c.rpc.GetShareBlobUrls(ctx, connect.NewRequest(&syncv1.GetShareBlobUrlsRequest{ShareRootId: shareRootID, BlobIds: blobIDs}))
	if err != nil {
		return nil, syncprotocol.ReadShareError(err)
	}
	return response.Msg.GetDownloads(), nil
}

func readAppendShareError(entry *syncv1.ShareLogEntry, err error) error {
	failure := syncprotocol.ReadShareError(err)
	if entry.GetKind() == syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE && errors.Is(failure, syncprotocol.ErrNoShare) {
		return fmt.Errorf("%w: %w", syncprotocol.ErrNoVault, failure)
	}
	if entry.GetKind() == syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER && connect.CodeOf(failure) == connect.CodeInvalidArgument {
		return fmt.Errorf("%w: %w", syncprotocol.ErrReaderRefused, failure)
	}
	return failure
}
