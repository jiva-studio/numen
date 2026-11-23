// Package cloud is the catalog, the revisions, the blobs and the chain of the account on the sync server,
// reached through the protocol library. A call the server refuses for want of a session signs in again once.
package cloud

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// Config is where the server is, what reaches it, the session token of each call and how to sign in again.
type Config struct {
	Address string
	HTTP    *http.Client
	Token   func() string
	Refresh func(ctx context.Context) error
}

// Cloud is the sync server at an address.
type Cloud struct {
	http      *http.Client
	refresh   func(ctx context.Context) error
	catalog   *syncprotocol.CatalogClient
	revisions *syncprotocol.RevisionClient
	blobs     *syncprotocol.BlobClient
	account   *syncprotocol.AccountClient
}

// New returns the server of the config.
func New(cfg Config) *Cloud {
	return &Cloud{
		http:      cfg.HTTP,
		refresh:   cfg.Refresh,
		catalog:   syncprotocol.NewCatalogClient(cfg.Address, cfg.HTTP, cfg.Token),
		revisions: syncprotocol.NewRevisionClient(cfg.Address, cfg.HTTP, cfg.Token),
		blobs:     syncprotocol.NewBlobClient(cfg.Address, cfg.HTTP, cfg.Token),
		account:   syncprotocol.NewAccountClient(cfg.Address, cfg.HTTP, cfg.Token),
	}
}

func call[T any](ctx context.Context, c *Cloud, do func() (T, error)) (T, error) {
	result, err := do()
	if !errors.Is(err, syncprotocol.ErrUnauthenticated) {
		return result, err
	}
	if refreshErr := c.refresh(ctx); refreshErr != nil {
		var none T
		return none, refreshErr
	}
	return do()
}

// AppendCatalogEntry adds an entry to the catalog.
func (c *Cloud) AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error) {
	return call(ctx, c, func() (*syncv1.AppendCatalogEntryResponse, error) { return c.catalog.AppendCatalogEntry(ctx, entry) })
}

// GetCatalog returns every entry after the version.
func (c *Cloud) GetCatalog(ctx context.Context, sinceVersion uint64) ([]*syncv1.CatalogEntry, error) {
	return call(ctx, c, func() ([]*syncv1.CatalogEntry, error) { return c.catalog.GetCatalog(ctx, sinceVersion) })
}

// CommitRevisions commits a batch of revisions of a vault.
func (c *Cloud) CommitRevisions(ctx context.Context, vaultID []byte, revisions []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error) {
	return call(ctx, c, func() (*syncv1.CommitRevisionsResponse, error) {
		return c.revisions.CommitRevisions(ctx, vaultID, revisions)
	})
}

// GetChanges returns a page of the head revisions that changed after the sequence number.
func (c *Cloud) GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	return call(ctx, c, func() (*syncv1.GetChangesResponse, error) {
		return c.revisions.GetChanges(ctx, vaultID, sinceSeq, limit)
	})
}

// ReserveBlobs reserves room for blobs of the stored sizes.
func (c *Cloud) ReserveBlobs(ctx context.Context, vaultID []byte, sizes []uint64) ([]*syncv1.BlobUpload, error) {
	return call(ctx, c, func() ([]*syncv1.BlobUpload, error) { return c.blobs.ReserveBlobs(ctx, vaultID, sizes) })
}

// GetBlobUrls returns the download of each blob.
func (c *Cloud) GetBlobUrls(ctx context.Context, vaultID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error) {
	return call(ctx, c, func() ([]*syncv1.BlobDownload, error) { return c.blobs.GetBlobUrls(ctx, vaultID, blobIDs) })
}

// GetAccountChain returns the chain of the account the server stores.
func (c *Cloud) GetAccountChain(ctx context.Context) ([]byte, error) {
	return call(ctx, c, func() ([]byte, error) { return c.account.GetAccountChain(ctx) })
}

// PutBlob uploads a blob to the URL of its reservation.
func (c *Cloud) PutBlob(ctx context.Context, url string, body []byte) error {
	return syncprotocol.PutBlob(ctx, c.http, url, body)
}

// GetBlob downloads a blob from its URL.
func (c *Cloud) GetBlob(ctx context.Context, url string) ([]byte, error) {
	return syncprotocol.GetBlob(ctx, c.http, url)
}
