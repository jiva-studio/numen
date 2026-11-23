// Package service is the composition root of Numen Sync: it builds the adapters and the use cases, keeps the sign-in
// and the sync running, and answers the methods of the settings view.
package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

const (
	keyNoIdentity = "no-identity"
	keyConflicts  = "conflicts"
)

// Backend answers the methods of the settings view.
type Backend struct {
	deps    syncvault.Deps
	catalog catalog.Deps
	pending ports.PausedStore
	resolve *deletions.ResolveDeletions
}

// NewBackend returns the backend over the dependencies of a pass.
func NewBackend(deps syncvault.Deps) *Backend {
	return &Backend{
		deps: deps,
		catalog: catalog.Deps{
			Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
			Journals: deps.Restore.Mirror,
			Random:   deps.Random,
		},
		pending: nil, resolve: nil,
	}
}

// ListVaults lists every vault of the installation with its state.
func (b *Backend) ListVaults(ctx context.Context, _ api.ListVaultsRequest) (api.ListVaultsResponse, error) {
	vaults, err := b.deps.Host.GetVaults(ctx)
	if err != nil {
		return api.ListVaultsResponse{Vaults: nil}, err
	}
	slices.SortStableFunc(vaults, func(a, c domain.Vault) int {
		return strings.Compare(a.Name+"\x00"+string(a.ID), c.Name+"\x00"+string(c.ID))
	})
	listed := make([]api.Vault, 0, len(vaults))
	holders := map[string]bool{}
	for _, vault := range vaults {
		one, err := b.getVault(ctx, vault, holders)
		if err != nil {
			return api.ListVaultsResponse{Vaults: nil}, err
		}
		listed = append(listed, one)
	}
	return api.ListVaultsResponse{Vaults: listed}, nil
}

func (b *Backend) getVault(ctx context.Context, vault domain.Vault, holders map[string]bool) (api.Vault, error) {
	one := api.Vault{ID: string(vault.ID), Name: vault.Name, Path: vault.Path, State: api.StateNotConnected, LastSyncedAt: "", ConflictCount: 0, PausedDeletions: nil}
	link, err := b.deps.State.ReadLink(ctx, vault.ID)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return one, nil
	}
	if err != nil {
		return one, err
	}
	if holders[link.RemoteVaultID] {
		one.State = api.StateCopy
		return one, nil
	}
	holders[link.RemoteVaultID] = true
	one.State = api.StateConnected
	status, err := b.deps.Progress.ReadStatus(ctx, vault.ID)
	if err != nil {
		return one, err
	}
	if !status.LastSyncedAt.IsZero() {
		one.LastSyncedAt = status.LastSyncedAt.UTC().Format(time.RFC3339)
	}
	one.ConflictCount = status.ConflictCount
	one.PausedDeletions, err = b.getPausedRemovals(ctx, vault.ID)
	return one, err
}

// ListRemoteVaults lists the synced vaults of the account.
func (b *Backend) ListRemoteVaults(ctx context.Context, _ api.ListRemoteVaultsRequest) (api.ListRemoteVaultsResponse, error) {
	remote, err := catalog.NewListRemoteVaults(b.catalog).Execute(ctx)
	if err != nil {
		return api.ListRemoteVaultsResponse{Vaults: nil}, err
	}
	listed := make([]api.RemoteVault, len(remote))
	for i, one := range remote {
		listed[i] = api.RemoteVault{RemoteID: one.RemoteID, Name: one.Name, IsConnected: one.IsConnected}
	}
	return api.ListRemoteVaultsResponse{Vaults: listed}, nil
}

func (b *Backend) ConnectVault(ctx context.Context, request api.ConnectVaultRequest) (api.ConnectVaultResponse, error) {
	input := catalog.ConnectInput{VaultID: domain.VaultID(request.VaultID), Mode: 0, RemoteID: request.RemoteID}
	switch request.Mode {
	case api.ModeNew:
		input.Mode = catalog.ModeNew
	case api.ModeExisting:
		input.Mode = catalog.ModeExisting
	}
	if !vaultlink.IsRemoteID(request.RemoteID) {
		return api.ConnectVaultResponse{}, catalog.ErrUnknownRemoteVault
	}
	return api.ConnectVaultResponse{}, catalog.NewConnectVault(b.catalog).Execute(ctx, input)
}

// GetStatus returns whether this device has an identity, whether the server answers and what warns.
func (b *Backend) GetStatus(ctx context.Context, _ api.GetStatusRequest) (api.GetStatusResponse, error) {
	status := api.GetStatusResponse{HasIdentity: true, IsReachable: false, Items: nil}
	_, err := b.deps.Identity.GetPublicIdentity(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		status.HasIdentity = false
		status.Items = append(status.Items, api.StatusItem{Key: keyNoIdentity, Text: "No identity on this device"})
		return status, nil
	}
	if err != nil {
		return status, err
	}
	_, err = b.deps.Chain.GetAccountChain(ctx)
	status.IsReachable = !errors.Is(err, syncprotocol.ErrUnreachable) && !errors.Is(err, sessiondomain.ErrUnreachable)
	count, err := b.countConflicts(ctx)
	if err != nil {
		return status, err
	}
	if count > 0 {
		status.Items = append(status.Items, api.StatusItem{Key: keyConflicts, Text: syncvault.GetConflictText(count)})
	}
	return status, nil
}

func (b *Backend) countConflicts(ctx context.Context) (int, error) {
	vaults, err := b.deps.Host.GetVaults(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, vault := range vaults {
		status, err := b.deps.Progress.ReadStatus(ctx, vault.ID)
		if err != nil {
			return 0, err
		}
		total += status.ConflictCount
	}
	return total, nil
}
