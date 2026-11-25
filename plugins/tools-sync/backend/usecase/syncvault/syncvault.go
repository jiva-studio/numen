// Package syncvault runs a pass over a synced vault, pulling what changed on the server and then pushing what
// changed on this device, and loops over the connected vaults.
package syncvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	subscriptiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrNotConnected means the vault has no synced vault to sync with.
var ErrNotConnected = vaultlink.ErrNotConnected

// State is the link and the records a pass keeps of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Progress is how the last pass over the vault ended and the notes it saw.
type Progress interface {
	ports.Statuses
	ports.Sightings
}

// Blobs are the blobs on the server.
type Blobs interface {
	ports.BlobUploader
	ports.BlobDownloader
}

// Deps are what a pass needs.
type Deps struct {
	Catalog       ports.Catalog
	Chain         ports.Chain
	Revisions     ports.Revisions
	Blobs         Blobs
	Crypto        ports.Crypto
	Identity      ports.Identity
	Host          ports.Host
	Notes         ports.NoteHost
	State         State
	Progress      Progress
	Health        ports.Health
	Clock         ports.Clock
	Random        io.Reader
	Subscriptions ports.Subscriptions
	Shop          ports.Shop
	Pauses        ports.Pauses
	Restore       RestoreDeps
	Shares        ports.Shares
	ShareStates   ports.ShareStates
	Box           ports.Box
	Vaults        ports.VaultList
	Folders       ports.Folders
	Forget        ports.Forgetter
	Locker        ports.VaultLocker
	Cloud         ports.CloudVaults
}

// RestoreDeps are where the recovery after a restored server reads the epoch and keeps the pinned key and the copies of
// the logs. Address is the address of the server, the name its key is pinned under.
type RestoreDeps struct {
	Address string
	Epochs  ports.Epochs
	Mirror  ports.Mirror
}

// Renamer gives the synced vault the name of its local vault, and reports whether it did.
type Renamer interface {
	Execute(ctx context.Context, id domain.VaultID) (bool, error)
}

// Sweeper deletes the staged files of the vault that no download names.
type Sweeper interface {
	Sweep(ctx context.Context, vault domain.VaultID) error
}

// Puller brings the changes of the server into the vault.
type Puller interface {
	Execute(ctx context.Context, in ports.PullInput) (restore.PullResult, error)
}

// Scanner lists the notes that changed on this device.
type Scanner interface {
	Execute(ctx context.Context, in transfer.ScanInput) ([]transfer.Change, error)
}

// Detector finds the files that moved and the ones that are gone.
type Detector interface {
	Execute(ctx context.Context, in transfer.ScanInput) (transfer.Scan, error)
}

type MoveCommitter interface {
	Execute(ctx context.Context, in ports.MoveInput) ([]transfer.Outcome, error)
}

type RemovalCommitter interface {
	Execute(ctx context.Context, in transfer.RemoveInput) ([]transfer.Outcome, error)
}

// Guard reports whether a pass may send removals.
type Guard interface {
	Execute(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, error)
}

// Committer commits files to the server.
type Committer interface {
	Execute(ctx context.Context, in ports.CommitInput) ([]transfer.Outcome, error)
}

// SubscriptionReader reads the subscription of the account from the server.
type SubscriptionReader interface {
	Execute(ctx context.Context) (subscriptiondomain.State, error)
}

// AccountRecoverer recovers the logs of the account after the server was restored.
type AccountRecoverer interface {
	Execute(ctx context.Context) error
}

// VaultRecoverer recovers the notes of one vault after the server was restored.
type VaultRecoverer interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// SharesReader reads the shares of a vault.
type SharesReader interface {
	Execute(ctx context.Context, id domain.VaultID) ([]share.Root, error)
}

// Rewrapper rewraps the files of a share.
type Rewrapper interface {
	Execute(ctx context.Context, id domain.VaultID, root share.Root) error
}

// Receiver brings the files of a received share into its vault.
type Receiver interface {
	Execute(ctx context.Context, id domain.VaultID) (restore.PullResult, error)
}

// Registrar registers a vault as a new synced vault with a fresh key.
type Registrar interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// Steps are the use cases a pass runs.
type Steps struct {
	Register       Registrar
	Rename         Renamer
	Sweep          Sweeper
	Pull           Puller
	Scan           Scanner
	Commit         Committer
	Detect         Detector
	Move           MoveCommitter
	Remove         RemovalCommitter
	Guard          Guard
	Subscription   SubscriptionReader
	RecoverAccount AccountRecoverer
	RecoverVault   VaultRecoverer
	Shares         SharesReader
	Rewrap         Rewrapper
	Receive        Receiver
}

// SyncVault runs one pass over one vault.
type SyncVault struct {
	deps    Deps
	steps   Steps
	reports reporter
}

// NewSyncVault returns the use case over the dependencies and the steps of a pass.
func NewSyncVault(deps Deps, steps Steps) *SyncVault {
	return &SyncVault{deps: deps, steps: steps, reports: reporter{health: deps.Health, subscription: steps.Subscription}}
}

// Execute pulls, then pushes, and keeps the warnings of the vault. It returns ErrNotConnected for a vault that is
// not connected.
func (u *SyncVault) Execute(ctx context.Context, id domain.VaultID) error {
	unlock, err := u.deps.Locker.LockVault(ctx, id)
	if err != nil {
		return err
	}
	defer unlock()
	link, err := u.deps.State.ReadLink(ctx, id)
	if err != nil {
		return err
	}
	if link.Kind == vaultlink.KindReceived {
		return u.executeReceived(ctx, id)
	}
	if isForeign, err := ports.IsKeyForeign(ctx, u.deps.Crypto, link); isForeign || err != nil {
		return u.dropForeign(ctx, id, isForeign, err)
	}
	if _, err := u.steps.Rename.Execute(ctx, id); err != nil {
		return err
	}
	authors, err := u.getAuthors(ctx)
	if err != nil {
		return err
	}
	if _, err := u.deps.Notes.GetNotes(id).EnsureIndexed(ctx); err != nil {
		return err
	}
	shares, err := u.steps.Shares.Execute(ctx, id)
	if err != nil {
		return err
	}
	p := &pass{
		use: u, id: id, authors: authors, shares: shares, conflicts: nil, resolved: 0, clashes: nil, unverified: 0, verdict: restore.Same, epochID: nil,
		isSent: false, isStaleEpoch: false,
	}
	if err := p.wrapKeysForShares(ctx); err != nil {
		return err
	}
	return p.run(ctx)
}

func (u *SyncVault) getAuthors(ctx context.Context) (trust.Authors, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	stored, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	return trust.NewAuthors(own.Chain, stored)
}

func (u *SyncVault) dropForeign(ctx context.Context, id domain.VaultID, isForeign bool, err error) error {
	if !isForeign {
		return err
	}
	if err := u.deps.Forget.ForgetVault(ctx, id); err != nil {
		return err
	}
	if err := u.steps.Register.Execute(ctx, id); err != nil {
		return err
	}
	return u.reports.raise(ctx, getKey(keyOtherIdentity, id), textOtherIdentity)
}
