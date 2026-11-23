// Package cloud is the catalog, the revisions, the blobs, the chain and the subscription of the account on the sync server,
// reached through the protocol library. A call the server refuses for want of a session signs in again once.
package cloud

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
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
	http         *http.Client
	refresh      func(ctx context.Context) error
	catalog      *syncclient.CatalogClient
	revisions    *syncclient.RevisionClient
	blobs        *syncclient.BlobClient
	account      *syncclient.AccountClient
	subscription *syncclient.SubscriptionClient
}

// New returns the server of the config.
func New(cfg Config) *Cloud {
	return &Cloud{
		http:         cfg.HTTP,
		refresh:      cfg.Refresh,
		catalog:      syncclient.NewCatalogClient(cfg.Address, cfg.HTTP, cfg.Token),
		revisions:    syncclient.NewRevisionClient(cfg.Address, cfg.HTTP, cfg.Token),
		blobs:        syncclient.NewBlobClient(cfg.Address, cfg.HTTP, cfg.Token),
		account:      syncclient.NewAccountClient(cfg.Address, cfg.HTTP, cfg.Token),
		subscription: syncclient.NewSubscriptionClient(cfg.Address, cfg.HTTP, cfg.Token),
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

func (c *Cloud) AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error) {
	return call(ctx, c, func() (*syncv1.AppendCatalogEntryResponse, error) { return c.catalog.AppendCatalogEntry(ctx, entry) })
}

// GetCatalog returns every entry after the version.
func (c *Cloud) GetCatalog(ctx context.Context, sinceVersion uint64) ([]*syncv1.CatalogEntry, error) {
	return call(ctx, c, func() ([]*syncv1.CatalogEntry, error) { return c.catalog.GetCatalog(ctx, sinceVersion) })
}

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

func (c *Cloud) GetBlobUrls(ctx context.Context, vaultID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error) {
	return call(ctx, c, func() ([]*syncv1.BlobDownload, error) { return c.blobs.GetBlobUrls(ctx, vaultID, blobIDs) })
}

// GetAccountChain returns the chain of the account the server stores.
func (c *Cloud) GetAccountChain(ctx context.Context) ([]byte, error) {
	return call(ctx, c, func() ([]byte, error) { return c.account.GetAccountChain(ctx) })
}

// PutBlob uploads a blob to the URL of its reservation.
func (c *Cloud) PutBlob(ctx context.Context, url string, body []byte) error {
	return syncclient.PutBlob(ctx, c.http, url, body)
}

func (c *Cloud) GetBlob(ctx context.Context, url string) ([]byte, error) {
	return syncclient.GetBlob(ctx, c.http, url)
}

func (c *Cloud) GetSubscription(ctx context.Context) (syncprotocol.Subscription, error) {
	return call(ctx, c, func() (syncprotocol.Subscription, error) { return c.subscription.GetSubscription(ctx) })
}

// CreateCheckout returns the address of a payment page for the plan.
func (c *Cloud) CreateCheckout(ctx context.Context, plan string) (string, error) {
	return call(ctx, c, func() (string, error) { return c.subscription.CreateCheckout(ctx, plan) })
}

// ListPlans returns the plans the server sells, ascending by quota.
func (c *Cloud) ListPlans(ctx context.Context) ([]syncprotocol.Plan, error) {
	return call(ctx, c, func() ([]syncprotocol.Plan, error) { return c.subscription.ListPlans(ctx) })
}

// ChangePlan moves the active subscription to the plan and returns the subscription after the change.
func (c *Cloud) ChangePlan(ctx context.Context, plan string) (syncprotocol.Subscription, error) {
	return call(ctx, c, func() (syncprotocol.Subscription, error) { return c.subscription.ChangePlan(ctx, plan) })
}

// ListProducts returns the products the server sells that are no plans.
func (c *Cloud) ListProducts(ctx context.Context) ([]syncprotocol.Product, error) {
	return call(ctx, c, func() ([]syncprotocol.Product, error) { return c.subscription.ListProducts(ctx) })
}

// GetEntitlements returns every product the account owns right now.
func (c *Cloud) GetEntitlements(ctx context.Context) ([]syncprotocol.Entitlement, error) {
	return call(ctx, c, func() ([]syncprotocol.Entitlement, error) { return c.subscription.GetEntitlements(ctx) })
}

// CreatePortalSession returns the address of the page where the person manages the subscription.
func (c *Cloud) CreatePortalSession(ctx context.Context) (string, error) {
	return call(ctx, c, func() (string, error) { return c.subscription.CreatePortalSession(ctx) })
}
