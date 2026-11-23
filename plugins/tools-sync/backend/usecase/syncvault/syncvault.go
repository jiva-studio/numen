// Package syncvault runs a pass over a synced vault, pulling what changed on the server and then pushing what
// changed on this device, and loops over the connected vaults.
package syncvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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
	State         State
	Progress      Progress
	Health        ports.Health
	Clock         ports.Clock
	Random        io.Reader
	Subscriptions ports.Subscriptions
	Pauses        ports.Pauses
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
	Execute(ctx context.Context, in transfer.PullInput) (transfer.PullResult, error)
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
	Execute(ctx context.Context, in transfer.MoveInput) ([]transfer.Outcome, error)
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
	Execute(ctx context.Context, in transfer.CommitInput) ([]transfer.Outcome, error)
}

// SubscriptionReader reads the subscription of the account from the server.
type SubscriptionReader interface {
	Execute(ctx context.Context) (subscriptiondomain.State, error)
}

// Steps are the use cases a pass runs.
type Steps struct {
	Rename       Renamer
	Sweep        Sweeper
	Pull         Puller
	Scan         Scanner
	Commit       Committer
	Detect       Detector
	Move         MoveCommitter
	Remove       RemovalCommitter
	Guard        Guard
	Subscription SubscriptionReader
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
	if _, err := u.deps.State.ReadLink(ctx, id); err != nil {
		return err
	}
	if _, err := u.steps.Rename.Execute(ctx, id); err != nil {
		return err
	}
	authors, err := u.getAuthors(ctx)
	if err != nil {
		return err
	}
	if _, err := u.deps.Host.Indexer(id).EnsureIndexed(ctx); err != nil {
		return err
	}
	p := &pass{use: u, id: id, authors: authors, conflicts: nil, unverified: 0, isReset: false}
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
