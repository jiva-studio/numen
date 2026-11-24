// Package service is the composition root of Numen Sync: it builds the adapters and the use cases, keeps the sign-in
// and the sync running, and answers the methods of the settings view.
package service

import (
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/accountwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/downloadwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/sharewire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/subscriptionwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/vaultwire"
)

// Backend answers the methods of the settings view.
type Backend struct {
	life *lifecycle.Lifecycle
	*vaultwire.Vaults
	*downloadwire.Downloads
	*sharewire.Shares
	*sharewire.Received
	*subscriptionwire.Subscription
	*accountwire.Account
}

// NewBackend returns the backend over the dependencies of a pass.
func NewBackend(deps syncvault.Deps) *Backend {
	return &Backend{
		life: nil, Downloads: nil,
		Vaults:       vaultwire.New(deps),
		Shares:       sharewire.New(deps),
		Received:     sharewire.NewReceived(deps),
		Subscription: subscriptionwire.New(deps),
		Account:      accountwire.New(),
	}
}

// WithAccountData gives the backend the deletion, the restore and the export of the account.
func (b *Backend) WithAccountData(deps accountdata.Deps) *Backend {
	b.Use(deps)
	return b
}

// WithDeletions gives the backend the answer to the pause before many removals.
func (b *Backend) WithDeletions(pending ports.PausedStore, resolve *deletions.ResolveDeletions) *Backend {
	b.SetDeletions(pending, resolve)
	return b
}

// WithDownloads gives the backend the methods of the downloads of absent files.
func (b *Backend) WithDownloads(downloads *downloadwire.Downloads) *Backend {
	b.Downloads = downloads
	return b
}
