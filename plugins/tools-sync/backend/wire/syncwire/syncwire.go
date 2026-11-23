// Package syncwire builds the pass over a synced vault from the use cases it runs.
package syncwire

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/upload"
)

// Deps are the dependencies of a pass and what the transfers of files of any size need besides.
type Deps struct {
	syncvault.Deps
	Uploads         upload.Uploads
	UploadEntries   upload.Entries
	DownloadBlobs   download.Blobs
	DownloadEntries download.Entries
	Pending         deletions.Pending
	Approvals       deletions.Approvals
	Absences        localchanges.DetectState
	RemovalDelay    func() time.Duration
}

// guard adapts the guard of removals to the words of the steps.
type guard struct{ use *deletions.GuardDeletions }

func (g guard) Execute(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, error) {
	candidates := make([]deletions.Removal, len(in.Candidates))
	for i, one := range in.Candidates {
		candidates[i] = deletions.Removal{FileID: one.FileID, Path: one.Path, BaseHash: one.BaseHash, RevID: one.RevID}
	}
	verdict, err := g.use.Execute(ctx, deletions.Input{
		Vault: in.Vault, Direction: deletions.Direction(in.Direction), Candidates: candidates, Tracked: in.Tracked,
	})
	switch verdict {
	case deletions.Pause:
		return transfer.Pause, err
	case deletions.Keep:
		return transfer.Keep, err
	}
	return transfer.Go, err
}

// NewResolveDeletions returns the use case that takes the answer of the person to a pause.
func NewResolveDeletions(deps Deps) *deletions.ResolveDeletions {
	return deletions.NewResolveDeletions(newDeletionsDeps(deps))
}

func newDeletionsDeps(deps Deps) deletions.Deps {
	return deletions.Deps{Pending: deps.Pending, Approvals: deps.Approvals, Health: deps.Health, Clock: deps.Clock}
}

// NewSyncVault returns the pass over one vault with the use cases of its steps over the dependencies.
func NewSyncVault(deps Deps) *syncvault.SyncVault {
	downloads := download.NewDownloadFile(download.Deps{Blobs: deps.DownloadBlobs, Entries: deps.DownloadEntries, Host: deps.Host})
	uploads := upload.NewUploadFile(upload.Deps{
		Uploads: deps.Uploads, Entries: deps.UploadEntries, Crypto: deps.Crypto, Host: deps.Host, Random: deps.Random,
	})
	guarded := guard{use: deletions.NewGuardDeletions(newDeletionsDeps(deps))}
	commitDeps := commit.Deps{
		Revisions: deps.Revisions, Blobs: deps.Blobs, Identity: deps.Identity, Crypto: deps.Crypto, Uploads: uploads, Random: deps.Random,
	}
	return syncvault.NewSyncVault(deps.Deps, syncvault.Steps{
		Rename: catalog.NewRenameRemoteVault(catalog.Deps{
			Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
			Random: deps.Random,
		}),
		Pull: pull.NewPullChanges(pull.Deps{
			Revisions: deps.Revisions, Blobs: deps.Blobs, Crypto: deps.Crypto, Host: deps.Host, State: deps.State,
			Downloads: downloads, Guard: guarded, Health: deps.Health,
		}),
		Scan:         localchanges.NewScanLocalChanges(localchanges.Deps{Host: deps.Host, State: deps.Progress, Clock: deps.Clock}),
		Detect:       localchanges.NewDetectMoves(localchanges.DetectDeps{Host: deps.Host, State: deps.Absences, Clock: deps.Clock, Delay: deps.RemovalDelay}),
		Commit:       commit.NewCommitRevisions(commitDeps),
		Move:         commit.NewCommitMoves(commitDeps),
		Remove:       commit.NewCommitRemovals(commitDeps),
		Guard:        guarded,
		Subscription: subscription.NewReadSubscription(subscription.Deps{Subscriptions: deps.Subscriptions}),
	})
}

// NewGuard returns the guard of removals the steps of a pass ask.
func NewGuard(deps Deps) interface {
	Execute(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, error)
} {
	return guard{use: deletions.NewGuardDeletions(newDeletionsDeps(deps))}
}
