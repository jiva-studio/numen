// Package syncvault runs a pass over a synced vault, pulling what changed on the server and then pushing what
// changed on this device, and loops over the connected vaults.
package syncvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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
	State         State
	Progress      Progress
	Health        ports.Health
	Clock         ports.Clock
	Random        io.Reader
	Subscriptions ports.Subscriptions
	Shares        ports.Shares
	ShareStates   ports.ShareStates
	Box           ports.Box
	Vaults        ports.VaultList
	Folders       ports.Folders
}

// Renamer gives the synced vault the name of its local vault, and reports whether it did.
type Renamer interface {
	Execute(ctx context.Context, id domain.VaultID) (bool, error)
}

// Puller brings the changes of the server into the vault.
type Puller interface {
	Execute(ctx context.Context, in transfer.PullInput, shares []share.Root) (transfer.PullResult, error)
}

// Scanner lists the notes that changed on this device.
type Scanner interface {
	Execute(ctx context.Context, in transfer.ScanInput) ([]transfer.Change, error)
}

// Committer commits notes to the server.
type Committer interface {
	Execute(ctx context.Context, in transfer.CommitInput, shares []share.Root) ([]transfer.Outcome, error)
}

// SubscriptionReader reads the subscription of the account from the server.
type SubscriptionReader interface {
	Execute(ctx context.Context) (subscriptiondomain.State, error)
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
	Execute(ctx context.Context, id domain.VaultID) (transfer.PullResult, error)
}

// Steps are the use cases a pass runs.
type Steps struct {
	Rename       Renamer
	Pull         Puller
	Scan         Scanner
	Commit       Committer
	Subscription SubscriptionReader
	Shares       SharesReader
	Rewrap       Rewrapper
	Receive      Receiver
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
	link, err := u.deps.State.ReadLink(ctx, id)
	if err != nil {
		return err
	}
	if link.Kind == vaultlink.KindReceived {
		return u.executeReceived(ctx, id)
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
	shares, err := u.steps.Shares.Execute(ctx, id)
	if err != nil {
		return err
	}
	p := &pass{use: u, id: id, authors: authors, shares: shares, conflicts: nil, unverified: 0}
	if err := p.rewrap(ctx); err != nil {
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
