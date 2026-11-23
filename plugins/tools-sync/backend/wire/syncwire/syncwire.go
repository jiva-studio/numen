// Package syncwire builds the pass over a synced vault from the use cases it runs.
package syncwire

import (
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/sharewire"
)

// NewSyncVault returns the pass over one vault with the use cases of its steps over the dependencies.
func NewSyncVault(deps syncvault.Deps) *syncvault.SyncVault {
	sharing := sharewire.NewShareDeps(deps)
	return syncvault.NewSyncVault(deps, syncvault.Steps{
		Rename: catalog.NewRenameRemoteVault(catalog.Deps{
			Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
			Random: deps.Random,
		}),
		Pull: pull.NewPullChanges(pull.Deps{
			Revisions: deps.Revisions, Blobs: deps.Blobs, Crypto: deps.Crypto, Host: deps.Host, State: deps.State,
		}),
		Scan:         localchanges.NewScanLocalChanges(localchanges.Deps{Host: deps.Host, State: deps.Progress, Clock: deps.Clock}),
		Commit:       commit.NewCommitRevisions(commit.Deps{Revisions: deps.Revisions, Blobs: deps.Blobs, Identity: deps.Identity, Random: deps.Random}),
		Subscription: subscription.NewReadSubscription(subscription.Deps{Subscriptions: deps.Subscriptions}),
		Shares:       share.NewReadShares(sharing),
		Rewrap:       share.NewRewrapFolder(sharing),
		Receive:      receive.NewPullShare(sharewire.NewReceiveDeps(deps)),
	})
}
