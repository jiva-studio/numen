// Package syncwire builds the pass over a synced vault from the use cases it runs.
package syncwire

import (
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
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
	Uploads         ports.Uploads
	UploadEntries   ports.UploadEntries
	DownloadBlobs   ports.BlobReader
	DownloadEntries ports.DownloadEntries
	Pending         ports.PausedStore
	Approvals       ports.Approvals
	Absences        localchanges.DetectState
	RemovalDelay    func() time.Duration
	Digests         ports.Digests
	Conflicts       ports.ConflictResolver
	LazyHeavy       bool
}

// NewResolveDeletions returns the use case that takes the answer of the person to a pause.
func NewResolveDeletions(deps Deps) *deletions.ResolveDeletions {
	return deletions.NewResolveDeletions(newDeletionsDeps(deps))
}

// NewGuard returns the guard of removals the steps of a pass ask.
func NewGuard(deps Deps) *deletions.GuardDeletions {
	return deletions.NewGuardDeletions(newDeletionsDeps(deps))
}

// NewDownloads returns the use case that stages the files the pull brings down.
func NewDownloads(deps Deps) *download.DownloadFile {
	return download.NewDownloadFile(download.Deps{Blobs: deps.DownloadBlobs, Entries: deps.DownloadEntries, Host: deps.Host, Digests: deps.Digests})
}

func newDeletionsDeps(deps Deps) deletions.Deps {
	return deletions.Deps{Pending: deps.Pending, Approvals: deps.Approvals, Health: deps.Health, Clock: deps.Clock}
}

// NewSyncVault returns the pass over one vault with the use cases of its steps over the dependencies.
func NewSyncVault(deps Deps) *syncvault.SyncVault {
	downloads := NewDownloads(deps)
	uploads := upload.NewUploadFile(upload.Deps{
		Uploads: deps.Uploads, Entries: deps.UploadEntries, Crypto: deps.Crypto, Host: deps.Host, Random: deps.Random, Digests: deps.Digests,
	})
	guard := NewGuard(deps)
	commitDeps := commit.Deps{
		Revisions: deps.Revisions, Blobs: deps.Blobs, Identity: deps.Identity, Crypto: deps.Crypto, Uploads: uploads, Random: deps.Random,
	}
	return syncvault.NewSyncVault(deps.Deps, syncvault.Steps{
		Rename: catalog.NewRenameRemoteVault(catalog.Deps{
			Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
			Random: deps.Random,
		}),
		Sweep: downloads,
		Pull: pull.NewPullChanges(pull.Deps{
			Revisions: deps.Revisions, Blobs: deps.Blobs, Crypto: deps.Crypto, Host: deps.Host, State: deps.State,
			Downloads: downloads, Guard: guard, Health: deps.Health,
			Conflicts: deps.Conflicts, Clock: deps.Clock, LazyHeavy: deps.LazyHeavy,
		}),
		Scan:         localchanges.NewScanLocalChanges(localchanges.Deps{Host: deps.Host, State: deps.Progress, Clock: deps.Clock}),
		Detect:       localchanges.NewDetectMoves(localchanges.DetectDeps{Host: deps.Host, State: deps.Absences, Clock: deps.Clock, Delay: deps.RemovalDelay}),
		Commit:       commit.NewCommitRevisions(commitDeps),
		Move:         commit.NewCommitMoves(commitDeps),
		Remove:       commit.NewCommitRemovals(commitDeps),
		Guard:        guard,
		Subscription: subscription.NewReadSubscription(subscription.Deps{Subscriptions: deps.Subscriptions}),
	})
}
