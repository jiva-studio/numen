// Package ports holds the interfaces the use cases of Numen Sync need, in their own words.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// Catalog is the catalog of the vaults of the account on the server.
type Catalog interface {
	AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error)
	GetCatalog(ctx context.Context, sinceVersion uint64) ([]*syncv1.CatalogEntry, error)
}

// CloudVaults is what the server stores of each synced vault, and the removal of one.
type CloudVaults interface {
	GetVaultUsage(ctx context.Context) ([]*syncv1.VaultUsage, error)
	DeleteVault(ctx context.Context, vaultID []byte) (*syncv1.DeleteVaultResponse, error)
}

// Chain is the chain of devices of the account the server stores. SendAccountChain signs in again with the chain of this
// device, which a server that lost devices of its chain takes over.
type Chain interface {
	GetAccountChain(ctx context.Context) ([]byte, error)
	SendAccountChain(ctx context.Context) error
	AppendChain(ctx context.Context, chain []byte) ([]byte, error)
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

// Notes is what the host does with the notes of one vault in its index and its trash.
type Notes interface {
	EnsureIndexed(ctx context.Context) (host.IndexState, error)
	RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error)
}

// Store is the folder of the plugin in the store of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
	Append(ctx context.Context, name string, content []byte) error
	List(ctx context.Context) ([]host.StoreFile, error)
	Delete(ctx context.Context, name string) error
}

// Absent is the files of one vault that the device lists and does not hold.
type Absent interface{ host.AbsentFiles }

// NoteHost is the notes of the vaults of the installation.
type NoteHost interface{ GetNotes(id domain.VaultID) Notes }

// Host is the vaults of the installation.
type Host interface {
	GetVaults(ctx context.Context) ([]domain.Vault, error)
	GetFiles(id domain.VaultID) Files
	GetAbsent(id domain.VaultID) Absent
	GetTransfers(id domain.VaultID) Transfers
	Store(id domain.VaultID) Store
}

// Links keeps the link of each vault.
type Links interface {
	ReadLink(ctx context.Context, id domain.VaultID) (vaultlink.Link, error)
	// WriteLink replaces the whole link; the caller holds the lock of the vault.
	WriteLink(ctx context.Context, id domain.VaultID, link vaultlink.Link) error
	UpdateLink(ctx context.Context, id domain.VaultID, change func(*vaultlink.Link) error) error
}

// VaultLocker lets one pass at a time work on a vault. LockVault waits for the vault and returns the function that
// releases it. IsLocked reports whether a holder has the vault now.
type VaultLocker interface {
	LockVault(ctx context.Context, id domain.VaultID) (func(), error)
	IsLocked(id domain.VaultID) bool
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
	ReadSightings(ctx context.Context, id domain.VaultID) (map[string]note.FileSnapshot, error)
	WriteSightings(ctx context.Context, id domain.VaultID, seen map[string]note.FileSnapshot) error
}

// Absences keeps when each synced file of a vault that is gone from it was first missed, by path, in nanoseconds since
// the Unix epoch.
type Absences interface {
	ReadAbsences(ctx context.Context, id domain.VaultID) (map[string]int64, error)
	WriteAbsences(ctx context.Context, id domain.VaultID, absent map[string]int64) error
}

// Health is where the plugin says what does not work. Reporting is best effort: Report and Clear return no
// error for a host that refuses an item, and no work fails because a report did not arrive (see besthealth).
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

// Epochs is the restore epoch of the server.
type Epochs interface {
	GetRestoreEpoch(ctx context.Context) (*syncv1.RestoreEpoch, []byte, error)
}

// Pins keeps the key that signs the restore epochs of each server, by the address of the server.
type Pins interface {
	ReadPin(ctx context.Context, address string) ([]byte, error)
	WritePin(ctx context.Context, address string, key []byte) error
}

// Mirror keeps the pinned keys and the copies of the logs of the account in one place.
type Mirror interface {
	Pins
	Journals
}

// Journals keeps the last verified entries of the logs of the account.
type Journals interface {
	ReadJournal(ctx context.Context, kind restore.Kind) (restore.Journal, error)
	WriteJournal(ctx context.Context, journal restore.Journal) error
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

// HintEvent is a message of the stream: a hint of a synced vault, or a heartbeat.
type HintEvent struct {
	RemoteVaultID string
	Seq           uint64
}

// IsHeartbeat reports whether the event is a heartbeat.
func (e HintEvent) IsHeartbeat() bool { return e.RemoteVaultID == "" }

// Watch is one open stream of hints.
type Watch interface {
	GetHeartbeat() time.Duration
	Receive(ctx context.Context) (HintEvent, error)
	Close() error
}

// Hints opens the stream of change hints of the account.
type Hints interface {
	Watch(ctx context.Context) (Watch, error)
}

// Waker asks the sync loop to pass over a vault now.
type Waker interface {
	Wake(vault domain.VaultID)
}

// LinkWork is what the core of this device wants carried for its pending link.
type LinkWork = account.LinkWork

// LinkSide is the end of a link channel a message goes to or is read by.
type LinkSide = syncclient.LinkSide

const (
	// LinkSideDisplayer is the device that shows the QR code.
	LinkSideDisplayer = syncclient.LinkSideDisplayer
	// LinkSideScanner is the device that reads it.
	LinkSideScanner = syncclient.LinkSideScanner
)

// LinkCore is the pending link of the core of this device, as a courier carries it.
type LinkCore interface {
	GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (account.LinkWork, error)
	SetLinkChannel(ctx context.Context, id []byte) error
	PutLinkInbound(ctx context.Context, blob []byte) error
}

// ChainCore is the chain of this device that the core keeps and the courier publishes.
type ChainCore interface {
	GetChainPublish(ctx context.Context) (account.ChainPublish, error)
	PutChainPublished(ctx context.Context, accepted bool, latest []byte) error
	AdoptChain(ctx context.Context, chain []byte) error
}

// Relay is the link relay of the sync server.
type Relay interface {
	CreateChannel(ctx context.Context) (syncclient.LinkChannel, error)
	SendMessage(ctx context.Context, channelID []byte, to syncclient.LinkSide, body []byte) (uint32, error)
	WaitMessages(ctx context.Context, channelID []byte, reader syncclient.LinkSide, afterSeq uint32, wait time.Duration) (syncclient.LinkMessages, error)
	CloseChannel(ctx context.Context, channelID []byte) error
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

// BlobURLReader downloads a blob from the address of a download.
type BlobURLReader interface {
	GetBlob(ctx context.Context, url string) ([]byte, error)
}

// Box opens what was sealed to the account box key of this device.
type Box interface {
	OpenBox(ctx context.Context, boxContext, sealed []byte) ([]byte, error)
}

// VaultList adds a vault to the vaults of the installation.
type VaultList interface {
	AddManagedVault(ctx context.Context, path, name, managedBy string) (domain.Vault, error)
	RemoveVault(ctx context.Context, id domain.VaultID) (string, error)
}

// Forgetter drops everything Numen Sync keeps of one vault in its store folder, the link included, and leaves the
// files of the vault alone.
type Forgetter interface {
	ForgetVault(ctx context.Context, id domain.VaultID) error
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

// DeleteChallenges is where this device gets the challenge it signs to delete its account.
type DeleteChallenges interface {
	GetAccountDeleteChallenge(ctx context.Context, deviceSignKey []byte) (session.Challenge, error)
}

// AccountData is what the server holds of the account, and the deletion of it.
type AccountData interface {
	DeleteAccount(ctx context.Context, challengeID, signature []byte) (time.Time, error)
	ExportAccount(ctx context.Context) ([]byte, error)
}

// Deletion is whether the account is being deleted, as the last sign-in found it, and the way back.
type Deletion interface {
	GetDeletion() (deleteAt time.Time, isDeleting bool)
	SetDeletion(deleteAt time.Time)
	Restore(ctx context.Context) error
}

// CommitInput is the files to commit to a vault. VaultID names the vault on this device; a note under the folder of
// one of Shares is sealed under the key of that share.
type CommitInput struct {
	VaultID domain.VaultID
	Vault   transfer.Vault
	Files   []transfer.File
	Shares  []share.Root
}

// MoveInput is the moves to commit to a vault, with the shares of the vault, whose folders seal the metadata of the files moved into or inside them.
type MoveInput struct {
	Vault  transfer.Vault
	Moves  []transfer.Move
	Shares []share.Root
}

// PullInput is the vault, the check of the author of a revision and the shares of the vault, whose revisions open with
// the keys of the share.
type PullInput struct {
	VaultID  domain.VaultID
	IsAuthor func(deviceSignKey []byte) bool
	Shares   []share.Root
}

// IsKeyForeign reports whether the vault key of the link was sealed for another identity, so the identity of this
// device cannot open it. Any other failure is returned as it is.
func IsKeyForeign(ctx context.Context, sealer vaultlink.Sealer, link vaultlink.Link) (bool, error) {
	_, err := vaultlink.OpenKeys(ctx, sealer, link)
	if errors.Is(err, session.ErrCannotOpen) {
		return true, nil
	}
	return false, err
}
