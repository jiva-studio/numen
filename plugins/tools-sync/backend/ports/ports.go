// Package ports holds the interfaces the use cases of Numen Sync need, in their own words.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// Catalog is the catalog of the vaults of the account on the server.
type Catalog interface {
	AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error)
	GetCatalog(ctx context.Context, sinceVersion uint64) ([]*syncv1.CatalogEntry, error)
}

// Chain is the chain of devices of the account the server stores.
type Chain interface {
	GetAccountChain(ctx context.Context) ([]byte, error)
}

// Revisions is the revisions of the notes of a vault on the server.
type Revisions interface {
	CommitRevisions(ctx context.Context, vaultID []byte, revisions []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error)
	GetChanges(ctx context.Context, vaultID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetChangesResponse, error)
}

// BlobUploader reserves room for blobs and uploads them.
type BlobUploader interface {
	ReserveBlobs(ctx context.Context, vaultID []byte, sizes []uint64) ([]*syncv1.BlobUpload, error)
	PutBlob(ctx context.Context, url string, body []byte) error
}

// BlobDownloader finds blobs and downloads them.
type BlobDownloader interface {
	GetBlobUrls(ctx context.Context, vaultID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error)
	GetBlob(ctx context.Context, url string) ([]byte, error)
}

// Crypto seals and opens data with the keys of the account.
type Crypto interface {
	SealData(ctx context.Context, dataContext, plain []byte) ([]byte, error)
	OpenData(ctx context.Context, dataContext, sealed []byte) ([]byte, error)
}

// Identity is the identity of this device.
type Identity interface {
	GetPublicIdentity(ctx context.Context) (session.Identity, error)
	SignAsDevice(ctx context.Context, message []byte) ([]byte, error)
}

// Files is the files of one vault and the feed of their changes.
type Files interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	host.FileChangeLister
}

// Indexer asks the host to read a vault into its index.
type Indexer interface {
	EnsureIndexed(ctx context.Context) (host.IndexState, error)
}

// Store is the folder of the plugin in the store of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
}

// Remover moves notes of one vault to the trash.
type Remover interface {
	RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error)
}

// Host is the vaults of the installation.
type Host interface {
	GetVaults(ctx context.Context) ([]domain.Vault, error)
	GetFiles(id domain.VaultID) Files
	GetIndexer(id domain.VaultID) Indexer
	Store(id domain.VaultID) Store
	GetRemover(id domain.VaultID) Remover
}

// Links keeps the link of each vault.
type Links interface {
	ReadLink(ctx context.Context, id domain.VaultID) (vaultlink.Link, error)
	WriteLink(ctx context.Context, id domain.VaultID, link vaultlink.Link) error
}

// Records keeps what is known of the synced notes of each vault.
type Records interface {
	ReadRecords(ctx context.Context, id domain.VaultID) ([]note.FileRecord, error)
	WriteRecords(ctx context.Context, id domain.VaultID, records []note.FileRecord) error
}

// Status is how the last pass over a vault ended.
type Status struct {
	LastSyncedAt  time.Time
	ConflictCount int
}

// Statuses keeps the status of each vault.
type Statuses interface {
	ReadStatus(ctx context.Context, id domain.VaultID) (Status, error)
	WriteStatus(ctx context.Context, id domain.VaultID, status Status) error
}

// Sightings keeps the notes of each vault as the last look found them.
type Sightings interface {
	ReadSightings(ctx context.Context, id domain.VaultID) (map[string]note.Seen, error)
	WriteSightings(ctx context.Context, id domain.VaultID, seen map[string]note.Seen) error
}

// Health is where the plugin says what does not work.
type Health interface {
	Report(ctx context.Context, item health.Item) error
	Clear(ctx context.Context, key string) error
}

// Clock reads the time and waits.
type Clock interface {
	Now() time.Time
	Wait(ctx context.Context, d time.Duration) error
}

// Subscriptions is the subscription of the account on the server.
type Subscriptions interface {
	GetSubscription(ctx context.Context) (syncprotocol.Subscription, error)
	CreateCheckout(ctx context.Context) (string, error)
}

// Shares is the share service of the server: the signed log of the members of a share and the files it holds for
// its readers.
type Shares interface {
	AppendShareLogEntry(ctx context.Context, entry *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error)
	ListShares(ctx context.Context) ([]*syncv1.ShareSummary, error)
	GetShareLog(ctx context.Context, shareRootID []byte, sinceVersion uint64) ([]*syncv1.ShareLogEntry, []byte, error)
	GetShareChanges(ctx context.Context, shareRootID []byte, sinceSeq uint64, limit uint32) (*syncv1.GetShareChangesResponse, error)
	GetShareBlobUrls(ctx context.Context, shareRootID []byte, blobIDs [][]byte) ([]*syncv1.BlobDownload, error)
}

// BlobReader downloads a blob from the address of a download.
type BlobReader interface {
	GetBlob(ctx context.Context, url string) ([]byte, error)
}

// Box opens what was sealed to the account box key of this device.
type Box interface {
	OpenBox(ctx context.Context, boxContext, sealed []byte) ([]byte, error)
}

// VaultList adds a vault to the vaults of the installation.
type VaultList interface {
	AddManagedVault(ctx context.Context, path, name, managedBy string) (domain.Vault, error)
}

// ReceivedFolder is the folder made for a received share and the name of its vault.
type ReceivedFolder struct {
	Path string
	Name string
}

// Folders makes the folders of received shares.
type Folders interface {
	MakeReceivedFolder(ctx context.Context, name string) (ReceivedFolder, error)
}

// ShareStates keeps where this device stands with the shares it made in each vault.
type ShareStates interface {
	ReadShareStates(ctx context.Context, id domain.VaultID) ([]share.Progress, error)
	WriteShareStates(ctx context.Context, id domain.VaultID, states []share.Progress) error
}
