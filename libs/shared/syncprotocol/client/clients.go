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

func newClientOptions(address string, token func() string) connect.ClientOption {
	return connect.WithInterceptors(syncprotocol.NewVersionInterceptor(), syncprotocol.NewBearerInterceptor(getTokenFor(address, token)))
}

// CatalogClient calls the catalog service of one server with the bearer of the session.
type CatalogClient struct {
	rpc syncv1connect.CatalogServiceClient
}

// NewCatalogClient returns the client of the server at address. token returns the session token and is
// read again for each call.
func NewCatalogClient(address string, httpClient *http.Client, token func() string) *CatalogClient {
	return &CatalogClient{rpc: syncv1connect.NewCatalogServiceClient(httpClient, address, newClientOptions(address, token))}
}

// AppendCatalogEntry adds an entry to the catalog. When another device was first, the answer is not committed
// and carries the entry at the head.
func (c *CatalogClient) AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error) {
	response, err := c.rpc.AppendCatalogEntry(ctx, connect.NewRequest(&syncv1.AppendCatalogEntryRequest{Entry: entry}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg, nil
}

// GetCatalog returns every entry after sinceVersion, in order, reading page after page.
func (c *CatalogClient) GetCatalog(ctx context.Context, sinceVersion uint64) ([]*syncv1.CatalogEntry, error) {
	var entries []*syncv1.CatalogEntry
	for {
		response, err := c.rpc.GetCatalog(ctx, connect.NewRequest(&syncv1.GetCatalogRequest{SinceVersion: sinceVersion}))
		if err != nil {
			return nil, syncprotocol.ReadCallError(err)
		}
		page := response.Msg.GetEntries()
		entries = append(entries, page...)
		if !response.Msg.GetHasMore() {
			return entries, nil
		}
		if len(page) == 0 {
			return nil, fmt.Errorf("the sync server has more catalog entries after version %d and sent none", sinceVersion)
		}
		if next := page[len(page)-1].GetVersion(); next > sinceVersion {
			sinceVersion = next
		} else {
			return nil, fmt.Errorf("the sync server sent a catalog page that ends at version %d, not after %d", next, sinceVersion)
		}
		if len(entries) >= syncprotocol.MaxCatalogEntries {
			return nil, fmt.Errorf("the sync server sent more than %d catalog entries", syncprotocol.MaxCatalogEntries)
		}
	}
}

// GetVaultUsage returns the stored bytes of every vault of the account, also of a vault no device syncs any more.
func (c *CatalogClient) GetVaultUsage(ctx context.Context) ([]*syncv1.VaultUsage, error) {
	response, err := c.rpc.GetVaultUsage(ctx, connect.NewRequest(&syncv1.GetVaultUsageRequest{}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetVaults(), nil
}

// DeleteVault removes the vault from the server. signature is the signature of the device of the session over the message of a challenge from
// SignInClient.GetVaultDeleteChallenge of the same vault. Deleted is false for a vault the server does not store for the
// account, so a repeat succeeds. A reserve or a commit that races with it is ErrNoVault.
func (c *CatalogClient) DeleteVault(ctx context.Context, vaultID, challengeID, signature []byte) (*syncv1.DeleteVaultResponse, error) {
	response, err := c.rpc.DeleteVault(ctx, connect.NewRequest(&syncv1.DeleteVaultRequest{VaultId: vaultID, ChallengeId: challengeID, Signature: signature}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg, nil
}

// RevisionClient calls the revision service of one server with the bearer of the session.
type RevisionClient struct {
	rpc syncv1connect.RevisionServiceClient
}

// NewRevisionClient returns the client of the server at address. token returns the session token and is
// read again for each call.
func NewRevisionClient(address string, httpClient *http.Client, token func() string) *RevisionClient {
	return &RevisionClient{rpc: syncv1connect.NewRevisionServiceClient(httpClient, address, newClientOptions(address, token))}
}

// CommitRevisions commits a batch of revisions of one vault, all of them or none. A batch that is not
// committed answers with its conflicts; a batch whose blob is not uploaded is BlobNotReadyError.
func (c *RevisionClient) CommitRevisions(ctx context.Context, vaultID []byte, revisions []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error) {
	response, err := c.rpc.CommitRevisions(ctx, connect.NewRequest(&syncv1.CommitRevisionsRequest{VaultId: vaultID, Revisions: revisions}))
	if err != nil {
		if failure := syncprotocol.ReadCallError(err); connect.CodeOf(failure) == connect.CodeInvalidArgument {
			return nil, fmt.Errorf("%w: %w", syncprotocol.ErrInvalidRevision, failure)
		}
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg, nil
}

// GetChanges returns one page of the head revisions that changed after sinceSeq. A limit of 0 is DefaultPageSize.
func (c *RevisionClient) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	response, err := c.rpc.GetChanges(ctx, connect.NewRequest(&syncv1.GetChangesRequest{VaultId: vaultID, SinceSeq: sinceSeq, Limit: limit}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg, nil
}

// BlobClient calls the blob service of one server with the bearer of the session.
type BlobClient struct {
	rpc syncv1connect.BlobServiceClient
}

// NewBlobClient returns the client of the server at address. token returns the session token and is read
// again for each call.
func NewBlobClient(address string, httpClient *http.Client, token func() string) *BlobClient {
	return &BlobClient{rpc: syncv1connect.NewBlobServiceClient(httpClient, address, newClientOptions(address, token))}
}

// ReserveBlobs reserves quota for bodies of the given stored sizes and returns one upload for each, in the
// order of the sizes.
func (c *BlobClient) ReserveBlobs(ctx context.Context, vaultID []byte, sizes []uint64) ([]*syncv1.BlobUpload, error) {
	response, err := c.rpc.ReserveBlobs(ctx, connect.NewRequest(&syncv1.ReserveBlobsRequest{VaultId: vaultID, Sizes: sizes}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetUploads(), nil
}

// GetBlobUrls returns one download for each blob id, in the order of the ids.
func (c *BlobClient) GetBlobUrls(ctx context.Context, vaultID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error) {
	response, err := c.rpc.GetBlobUrls(ctx, connect.NewRequest(&syncv1.GetBlobUrlsRequest{VaultId: vaultID, BlobIds: blobIDs}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetDownloads(), nil
}

// AccountClient calls the account service of one server.
type AccountClient struct {
	rpc syncv1connect.AccountServiceClient
}

// NewAccountClient returns the client of the server at address. token returns the session token and is
// read again for each call; GetAccountChainByRoot needs none.
func NewAccountClient(address string, httpClient *http.Client, token func() string) *AccountClient {
	return &AccountClient{rpc: syncv1connect.NewAccountServiceClient(httpClient, address, newClientOptions(address, token))}
}

// GetAccountChain returns the stored chain of the account of the session.
func (c *AccountClient) GetAccountChain(ctx context.Context) ([]byte, error) {
	response, err := c.rpc.GetAccountChain(ctx, connect.NewRequest(&syncv1.GetAccountChainRequest{}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetChain(), nil
}

// GetAccountChainByRoot returns the stored chain of the account with the root key, for a caller that signed
// syncprotocol.GetChainReadMessage of the challenge with the recovery key. found is false when no account has this root.
func (c *AccountClient) GetAccountChainByRoot(ctx context.Context, challengeID, rootSignKey, signature []byte) (chain []byte, found bool, err error) {
	response, err := c.rpc.GetAccountChainByRoot(ctx, connect.NewRequest(&syncv1.GetAccountChainByRootRequest{
		ChallengeId: challengeID, RootSignKey: rootSignKey, Signature: signature,
	}))
	if err != nil {
		return nil, false, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetChain(), response.Msg.GetFound(), nil
}

// ReserveMultipartBlob reserves quota for a body above syncprotocol.MaxBlobSize and starts a multipart upload.
func (c *BlobClient) ReserveMultipartBlob(ctx context.Context, vaultID []byte, size, partSize uint64) (*syncv1.ReserveMultipartBlobResponse, error) {
	response, err := c.rpc.ReserveMultipartBlob(ctx, connect.NewRequest(&syncv1.ReserveMultipartBlobRequest{VaultId: vaultID, Size: size, PartSize: partSize}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg, nil
}

// GetPartUrls returns one presigned PUT for each part number, in the order of the numbers.
func (c *BlobClient) GetPartUrls(ctx context.Context, vaultID, blobID []byte, partNumbers []uint32) ([]*syncv1.PartUpload, error) {
	response, err := c.rpc.GetPartUrls(ctx, connect.NewRequest(&syncv1.GetPartUrlsRequest{VaultId: vaultID, BlobId: blobID, PartNumbers: partNumbers}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetUploads(), nil
}

// ListParts returns the parts the storage holds of an open upload, sorted by number.
func (c *BlobClient) ListParts(ctx context.Context, vaultID, blobID []byte) ([]*syncv1.PartInfo, error) {
	response, err := c.rpc.ListParts(ctx, connect.NewRequest(&syncv1.ListPartsRequest{VaultId: vaultID, BlobId: blobID}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetParts(), nil
}

// CompleteMultipartBlob joins the parts into the object. Parts that are missing or of the wrong size are
// syncprotocol.MultipartIncompleteError.
func (c *BlobClient) CompleteMultipartBlob(ctx context.Context, vaultID, blobID []byte) error {
	if _, err := c.rpc.CompleteMultipartBlob(ctx, connect.NewRequest(&syncv1.CompleteMultipartBlobRequest{VaultId: vaultID, BlobId: blobID})); err != nil {
		return syncprotocol.ReadCallError(err)
	}
	return nil
}

// AbortMultipartBlob drops an open upload and frees its quota. A repeat is a success.
func (c *BlobClient) AbortMultipartBlob(ctx context.Context, vaultID, blobID []byte) error {
	if _, err := c.rpc.AbortMultipartBlob(ctx, connect.NewRequest(&syncv1.AbortMultipartBlobRequest{VaultId: vaultID, BlobId: blobID})); err != nil {
		return syncprotocol.ReadCallError(err)
	}
	return nil
}

func getTokenFor(address string, token func() string) func() string {
	if syncprotocol.CheckServerAddress(address) != nil {
		return func() string { return "" }
	}
	return token
}

// DeleteAccount deletes the account of the session; nothing restores it. signature is the signature of the device of the
// session over the message of a challenge from SignInClient.GetAccountDeleteChallenge.
func (c *AccountClient) DeleteAccount(ctx context.Context, challengeID, signature []byte) error {
	if _, err := c.rpc.DeleteAccount(ctx, connect.NewRequest(&syncv1.DeleteAccountRequest{ChallengeId: challengeID, Signature: signature})); err != nil {
		return syncprotocol.ReadCallError(err)
	}
	return nil
}

// ExportAccount reads the stream of the JSON document of what the server holds of the account of the session and
// returns the chunks joined in order.
func (c *AccountClient) ExportAccount(ctx context.Context) ([]byte, error) {
	stream, err := c.rpc.ExportAccount(ctx, connect.NewRequest(&syncv1.ExportAccountRequest{}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	var document []byte
	for stream.Receive() {
		document = append(document, stream.Msg().GetData()...)
	}
	if err := stream.Err(); err != nil {
		return nil, errors.Join(syncprotocol.ReadCallError(err), stream.Close())
	}
	return document, stream.Close()
}
