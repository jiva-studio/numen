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
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
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

// Transfers is what the host does with the bytes of files of any kind.
type Transfers interface {
	ReadBytes(ctx context.Context, path string, start, length int64) (host.ByteRange, error)
	MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (host.MoveFileResult, error)
	RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (host.RemoveFileResult, error)
	PlaceFile(ctx context.Context, name, path, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error)
}

// Indexer asks the host to read a vault into its index.
type Indexer interface {
	EnsureIndexed(ctx context.Context) (host.IndexState, error)
}

// Store is the folder of the plugin in the store of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
	Append(ctx context.Context, name string, content []byte) error
	List(ctx context.Context) ([]host.StoreFile, error)
	Delete(ctx context.Context, name string) error
}

// Host is the vaults of the installation.
type Host interface {
	GetVaults(ctx context.Context) ([]domain.Vault, error)
	GetFiles(id domain.VaultID) Files
	GetTransfers(id domain.VaultID) Transfers
	GetIndexer(id domain.VaultID) Indexer
	Store(id domain.VaultID) Store
}

// Links keeps the link of each vault.
type Links interface {
	ReadLink(ctx context.Context, id domain.VaultID) (vaultlink.Link, error)
	WriteLink(ctx context.Context, id domain.VaultID, link vaultlink.Link) error
}

// Records keeps what is known of the synced notes of each vault.
type Records interface {
	ReadRecords(ctx context.Context, id domain.VaultID) ([]transfer.FileRecord, error)
	WriteRecords(ctx context.Context, id domain.VaultID, records []transfer.FileRecord) error
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

// Absences keeps when each synced file of a vault that is gone from it was first missed, by path, in nanoseconds since
// the Unix epoch.
type Absences interface {
	ReadAbsences(ctx context.Context, id domain.VaultID) (map[string]int64, error)
	WriteAbsences(ctx context.Context, id domain.VaultID, absent map[string]int64) error
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
	ListPlans(ctx context.Context) ([]syncprotocol.Plan, error)
	CreateCheckout(ctx context.Context, plan string) (string, error)
	ChangePlan(ctx context.Context, plan string) (syncprotocol.Subscription, error)
}

// Shop is the page where the person manages the subscription.
type Shop interface {
	CreatePortalSession(ctx context.Context) (string, error)
}

// Head is the head of a live note of a synced vault: its path and the hash of its text.
type Head struct {
	Path string
	Hash string
}

// Heads lists the verified heads of the live notes of a synced vault of the account.
type Heads interface {
	ListHeads(ctx context.Context, remoteID string) ([]Head, error)
}

// Pauses reports whether the removals of a vault wait for an answer.
type Pauses interface {
	HasPending(ctx context.Context, id domain.VaultID) (bool, error)
}

// FileSide is the local file of a conflict: its hash and the stat the host gave it.
type FileSide struct {
	Hash string
	Stat domain.FileStat
}

// HeadSide is the head of a conflict: its hash and size, and Replace, which writes the head over the file at the path
// of the conflict when the file still has the baseline.
type HeadSide struct {
	Hash    string
	Size    int64
	Replace func(ctx context.Context, baseline domain.FileStat) error
}

// ConflictInput is a file both computers changed.
type ConflictInput struct {
	Vault domain.VaultID
	Path  string
	Local FileSide
	Head  HeadSide
	Day   time.Time
}

// ConflictOutcome is the paths a resolve wrote, and the path of the copy, empty when there is none.
type ConflictOutcome struct {
	Applied []string
	Copy    string
}

// ConflictResolver settles a file both computers changed.
type ConflictResolver interface {
	Resolve(ctx context.Context, in ConflictInput) (ConflictOutcome, error)
}
