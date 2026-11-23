// Package syncvault runs a pass over a synced vault, pulling what changed on the server and then pushing what
// changed on this device, and loops over the connected vaults.
package syncvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
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
	Catalog   ports.Catalog
	Chain     ports.Chain
	Revisions ports.Revisions
	Blobs     Blobs
	Crypto    ports.Crypto
	Identity  ports.Identity
	Host      ports.Host
	State     State
	Progress  Progress
	Health    ports.Health
	Clock     ports.Clock
	Random    io.Reader
}

// passSteps are the use cases a pass runs.
type passSteps struct {
	rename *catalog.RenameRemoteVault
	pull   *pull.PullChanges
	scan   *localchanges.ScanLocalChanges
	commit *commit.CommitRevisions
}

// SyncVault runs one pass over one vault.
type SyncVault struct {
	deps    Deps
	steps   passSteps
	reports reporter
}

// NewSyncVault returns the use case over the dependencies.
func NewSyncVault(deps Deps) *SyncVault {
	return &SyncVault{
		deps: deps,
		steps: passSteps{
			rename: catalog.NewRenameRemoteVault(catalog.Deps{
				Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
				Random: deps.Random,
			}),
			pull: pull.NewPullChanges(pull.Deps{
				Revisions: deps.Revisions, Blobs: deps.Blobs, Crypto: deps.Crypto, Host: deps.Host, State: deps.State,
			}),
			scan:   localchanges.NewScanLocalChanges(localchanges.Deps{Host: deps.Host, State: deps.Progress, Clock: deps.Clock}),
			commit: commit.NewCommitRevisions(commit.Deps{Revisions: deps.Revisions, Blobs: deps.Blobs, Identity: deps.Identity, Random: deps.Random}),
		},
		reports: reporter{health: deps.Health},
	}
}

// Execute pulls, then pushes, and keeps the warnings of the vault. It returns ErrNotConnected for a vault that is
// not connected.
func (u *SyncVault) Execute(ctx context.Context, id domain.VaultID) error {
	if _, err := u.deps.State.ReadLink(ctx, id); err != nil {
		return err
	}
	if _, err := u.steps.rename.Execute(ctx, id); err != nil {
		return err
	}
	authors, err := u.getAuthors(ctx)
	if err != nil {
		return err
	}
	if _, err := u.deps.Host.Indexer(id).EnsureIndexed(ctx); err != nil {
		return err
	}
	p := &pass{use: u, id: id, authors: authors, conflicts: nil, unverified: 0}
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
