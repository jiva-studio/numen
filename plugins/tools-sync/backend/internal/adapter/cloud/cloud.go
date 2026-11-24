// Package cloud is the catalog, the revisions, the blobs, the chain, the restore epoch and the subscription of the account on the sync server,
// reached through the protocol library. A call the server refuses for want of a session signs in again once.
package cloud

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

// Config is where the server is, what reaches it, the session token of each call and how to sign in again.
type Config struct {
	Address string
	HTTP    *http.Client
	Token   func() string
	Refresh func(ctx context.Context) error
	Stream  *http.Client
	Storage *http.Client
	Roots   Roots
	Now     func() time.Time
}

// Cloud is the sync server at an address.
type Cloud struct {
	http         *http.Client
	storage      *http.Client
	refresh      func(ctx context.Context) error
	catalog      *syncclient.CatalogClient
	revisions    *syncclient.RevisionClient
	blobs        *syncclient.BlobClient
	account      *syncclient.AccountClient
	subscription *syncclient.SubscriptionClient
	restore      *syncclient.RestoreClient
	watch        *syncclient.WatchClient
	shares       *syncclient.ShareClient
}

// New returns the server of the config.
func New(cfg Config) *Cloud {
	return &Cloud{
		http:         cfg.HTTP,
		storage:      cmp.Or(cfg.Storage, cfg.HTTP),
		refresh:      cfg.Refresh,
		catalog:      syncclient.NewCatalogClient(cfg.Address, cfg.HTTP, cfg.Token),
		revisions:    syncclient.NewRevisionClient(cfg.Address, cfg.HTTP, cfg.Token),
		blobs:        syncclient.NewBlobClient(cfg.Address, cfg.HTTP, cfg.Token),
		account:      syncclient.NewAccountClient(cfg.Address, cfg.HTTP, cfg.Token),
		subscription: syncclient.NewSubscriptionClient(cfg.Address, cfg.HTTP, cfg.Token),
		restore:      syncclient.NewRestoreClient(cfg.Address, cfg.HTTP, cfg.Token, cfg.Roots, cfg.Now),
		watch:        syncclient.NewWatchClient(cfg.Address, cfg.Stream, cfg.Token),
		shares:       syncclient.NewShareClient(cfg.Address, cfg.HTTP, cfg.Token),
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
	return syncclient.PutBlob(ctx, c.storage, url, body)
}

func (c *Cloud) GetBlob(ctx context.Context, url string) ([]byte, error) {
	return syncclient.GetBlob(ctx, c.storage, url)
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

// CreatePortalSession returns the address of the page where the person manages the subscription.
func (c *Cloud) CreatePortalSession(ctx context.Context) (string, error) {
	return call(ctx, c, func() (string, error) { return c.subscription.CreatePortalSession(ctx) })
}

// GetRestoreEpoch returns the current restore epoch, nil before the first restore, and the key that signs epochs.
func (c *Cloud) GetRestoreEpoch(ctx context.Context) (*syncv1.RestoreEpoch, []byte, error) {
	var key []byte
	epoch, err := call(ctx, c, func() (*syncv1.RestoreEpoch, error) {
		found, signKey, err := c.restore.GetRestoreEpoch(ctx)
		key = signKey
		return found, err
	})
	if errors.Is(err, epochtrust.ErrUntrusted) {
		err = fmt.Errorf("%w: %w", restore.ErrEpochUntrusted, err)
	}
	return epoch, key, err
}

// SendAccountChain signs in again, which sends the chain of this device, so a server that lost devices of the chain
// takes the longer one.
func (c *Cloud) SendAccountChain(ctx context.Context) error { return c.refresh(ctx) }

// AppendChain presents a longer chain of the account and returns the chain the server stores after the call.
func (c *Cloud) AppendChain(ctx context.Context, chain []byte) ([]byte, error) {
	return call(ctx, c, func() ([]byte, error) { return c.account.AppendChain(ctx, chain) })
}

// AppendShareLogEntry adds an entry to the member log of a share. A refused entry is returned uncommitted, with the
// entry at the head of the log.
func (c *Cloud) AppendShareLogEntry(ctx context.Context, entry *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error) {
	return call(ctx, c, func() (*syncv1.AppendShareLogEntryResponse, error) { return c.shares.AppendShareLogEntry(ctx, entry) })
}

// ListShares returns the share roots of the account.
func (c *Cloud) ListShares(ctx context.Context) ([]*syncv1.ShareSummary, error) {
	return call(ctx, c, func() ([]*syncv1.ShareSummary, error) { return c.shares.ListShares(ctx) })
}

// GetShareLog returns the entries of the share log after the version and the chain of the owner.
func (c *Cloud) GetShareLog(ctx context.Context, shareRootID []byte, sinceVersion uint64) ([]*syncv1.ShareLogEntry, []byte, error) {
	var chain []byte
	entries, err := call(ctx, c, func() ([]*syncv1.ShareLogEntry, error) {
		read, readChain, err := c.shares.GetShareLog(ctx, shareRootID, sinceVersion)
		chain = readChain
		return read, err
	})
	return entries, chain, err
}

// GetShareChanges returns a page of the changes of a share after the sequence number.
func (c *Cloud) GetShareChanges(ctx context.Context, shareRootID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetShareChangesResponse, error) {
	return call(ctx, c, func() (*syncv1.GetShareChangesResponse, error) {
		return c.shares.GetShareChanges(ctx, shareRootID, sinceSeq, limit)
	})
}

func (c *Cloud) GetShareBlobUrls(ctx context.Context, shareRootID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error) {
	return call(ctx, c, func() ([]*syncv1.BlobDownload, error) { return c.shares.GetShareBlobUrls(ctx, shareRootID, blobIDs) })
}

// DeleteAccount asks the server to delete the account after its grace period, and returns when it deletes it for good.
func (c *Cloud) DeleteAccount(ctx context.Context, challengeID, signature []byte) (time.Time, error) {
	return call(ctx, c, func() (time.Time, error) { return c.account.DeleteAccount(ctx, challengeID, signature) })
}

// ExportAccount returns the JSON document of what the server holds of the account.
func (c *Cloud) ExportAccount(ctx context.Context) ([]byte, error) {
	return call(ctx, c, func() ([]byte, error) { return c.account.ExportAccount(ctx) })
}
