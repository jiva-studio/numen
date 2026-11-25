// Package unlink ends the sync of a vault on this device and leaves its files where they are.
package unlink

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the use cases need.
type Deps struct {
	Links  ports.Links
	Forget ports.Forgetter
	Vaults ports.VaultList
	Cloud  CloudDeleter
	Locker ports.VaultLocker
}

// CloudDeleter deletes a synced vault from the cloud and forgets the links of this device to it.
type CloudDeleter interface {
	Execute(ctx context.Context, remoteID string) error
}

// StopSyncing ends the sync of a vault of this account on this device.
type StopSyncing struct{ deps Deps }

// NewStopSyncing returns the use case over the dependencies.
func NewStopSyncing(deps Deps) *StopSyncing { return &StopSyncing{deps: deps} }

// Execute forgets the link and what the sync kept of the vault, so the vault is not synced any more. The files stay, and
// so does the synced vault on the server, unless isDeleting is set: then the cloud copy is deleted first, and the vault
// stays linked when that fails.
func (u *StopSyncing) Execute(ctx context.Context, id domain.VaultID, isDeleting bool) error {
	unlock, err := u.deps.Locker.LockVault(ctx, id)
	if err != nil {
		return err
	}
	defer unlock()
	link, err := u.deps.Links.ReadLink(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return domainshare.ErrNotConnected
	}
	if err != nil {
		return err
	}
	if link.Kind == vaultlink.KindReceived {
		return domainshare.ErrReceivedStop
	}
	if isDeleting {
		return u.deps.Cloud.Execute(ctx, link.RemoteVaultID)
	}
	return u.deps.Forget.ForgetVault(ctx, id)
}

// Leave drops a share another person made with this account.
type Leave struct{ deps Deps }

// NewLeave returns the use case over the dependencies.
func NewLeave(deps Deps) *Leave { return &Leave{deps: deps} }

// Execute forgets the link and takes the vault of the share off the list of the installation, which makes its folder
// an ordinary folder. The files stay. The owner is not told: the share log has no entry a reader may sign, so the owner
// goes on listing this account as a reader until it removes it.
func (u *Leave) Execute(ctx context.Context, id domain.VaultID) error {
	unlock, err := u.deps.Locker.LockVault(ctx, id)
	if err != nil {
		return err
	}
	defer unlock()
	link, err := u.deps.Links.ReadLink(ctx, id)
	if err != nil && !errors.Is(err, vaultlink.ErrNotConnected) {
		return err
	}
	if err != nil || link.Kind != vaultlink.KindReceived {
		return domainshare.ErrNotReceived
	}
	if err := u.deps.Forget.ForgetVault(ctx, id); err != nil {
		return err
	}
	if _, err := u.deps.Vaults.RemoveVault(ctx, id); err != nil {
		link.ServerSeq = 0
		return errors.Join(err, u.deps.Links.WriteLink(ctx, id, link))
	}
	return nil
}
