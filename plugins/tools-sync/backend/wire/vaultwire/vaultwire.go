// Package vaultwire answers the methods of the settings view about the vaults of the installation and of the account:
// listing, connecting, previewing a connection, the status and the answer to the pause before many removals.
package vaultwire

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
	domaintransfer "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/preview"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/unlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/sharewire"
)

const (
	keyNoIdentity = "no-identity"
	keyConflicts  = "conflicts"
)

// Vaults answers the methods about the vaults, as the three parts below.
type Vaults struct {
	*Listing
	*Connections
	*Removals
}

// Listing answers the methods about the state of the vaults of the installation.
type Listing struct {
	deps    syncvault.Deps
	pending ports.PausedStore
	resolve *deletions.ResolveDeletions
}

// Connections answers the methods about connecting a vault to a synced vault.
type Connections struct {
	deps    syncvault.Deps
	catalog catalog.Deps
	connect *catalog.ConnectVault
	heads   ports.Heads
}

// Removals answers the methods that end the sync of a vault or delete a synced vault.
type Removals struct {
	stop   *unlink.StopSyncing
	delete *catalog.DeleteCloudVault
}

// New returns the methods over the dependencies of a pass and the heads of a synced vault.
func New(deps syncvault.Deps, heads ports.Heads) *Vaults {
	catalogDeps := catalog.Deps{
		Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State,
		Journals: deps.Restore.Mirror, Cloud: deps.Cloud, Forget: deps.Forget,
		Random: deps.Random,
	}
	deleting := catalog.NewDeleteCloudVault(catalogDeps, share.NewCloseShares(sharewire.NewShareDeps(deps)))
	return &Vaults{
		Listing:     &Listing{deps: deps, pending: nil, resolve: nil},
		Connections: &Connections{deps: deps, catalog: catalogDeps, connect: catalog.NewConnectVault(catalogDeps), heads: heads},
		Removals: &Removals{
			stop:   unlink.NewStopSyncing(unlink.Deps{Links: deps.State, Forget: deps.Forget, Vaults: deps.Vaults, Cloud: deleting}),
			delete: deleting,
		},
	}
}

// SetDeletions gives the methods the answer to the pause before many removals.
func (v *Listing) SetDeletions(pending ports.PausedStore, resolve *deletions.ResolveDeletions) {
	v.pending, v.resolve = pending, resolve
}

// ListVaults lists every vault of the installation with its state.
func (v *Listing) ListVaults(ctx context.Context, _ api.ListVaultsRequest) (api.ListVaultsResponse, error) {
	vaults, err := v.deps.Host.GetVaults(ctx)
	if err != nil {
		return api.ListVaultsResponse{Vaults: nil}, err
	}
	slices.SortStableFunc(vaults, func(a, c domain.Vault) int {
		return strings.Compare(a.Name+"\x00"+string(a.ID), c.Name+"\x00"+string(c.ID))
	})
	listed := make([]api.Vault, 0, len(vaults))
	holders := map[string]bool{}
	for _, vault := range vaults {
		one, err := v.getVault(ctx, vault, holders)
		if err != nil {
			return api.ListVaultsResponse{Vaults: nil}, err
		}
		listed = append(listed, one)
	}
	return api.ListVaultsResponse{Vaults: listed}, nil
}

func (v *Listing) getVault(ctx context.Context, vault domain.Vault, holders map[string]bool) (api.Vault, error) {
	one := api.Vault{ID: string(vault.ID), Name: vault.Name, Path: vault.Path, State: api.StateNotConnected, LastSyncedAt: "", ConflictCount: 0, PausedDeletions: nil, RemoteID: ""}
	link, err := v.deps.State.ReadLink(ctx, vault.ID)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return one, nil
	}
	if err != nil {
		return one, err
	}
	switch {
	case link.Kind == vaultlink.KindReceived:
		one.State = api.StateReceived
	case holders[link.RemoteVaultID]:
		one.State = api.StateCopy
		return one, nil
	default:
		holders[link.RemoteVaultID] = true
		one.State = api.StateConnected
		one.RemoteID = link.RemoteVaultID
	}
	status, err := v.deps.Progress.ReadStatus(ctx, vault.ID)
	if err != nil {
		return one, err
	}
	if !status.LastSyncedAt.IsZero() {
		one.LastSyncedAt = status.LastSyncedAt.UTC().Format(time.RFC3339)
	}
	one.ConflictCount = status.ConflictCount
	one.PausedDeletions, err = v.getPausedRemovals(ctx, vault.ID)
	return one, err
}

func (v *Listing) getPausedRemovals(ctx context.Context, id domain.VaultID) (*api.PausedDeletions, error) {
	if v.pending == nil {
		return nil, nil
	}
	state, err := v.pending.ReadPending(ctx, id)
	if err != nil || state == nil {
		return nil, err
	}
	return &api.PausedDeletions{Count: len(state.Files), Tracked: state.Tracked, Direction: api.Direction(state.Direction)}, nil
}

// ResolveDeletions takes the answer of the person to the removals of a vault that wait for it.
func (v *Listing) ResolveDeletions(ctx context.Context, request api.ResolveDeletionsRequest) (api.ResolveDeletionsResponse, error) {
	if v.resolve == nil {
		return api.ResolveDeletionsResponse{}, deletions.ErrNothingPending
	}
	_, err := v.resolve.Execute(ctx, domain.VaultID(request.VaultID), domaintransfer.Choice(request.Choice))
	return api.ResolveDeletionsResponse{}, err
}

// StopSyncingVault ends the sync of the vault on this device. Its files and its synced vault on the server stay.
func (v *Removals) StopSyncingVault(ctx context.Context, request api.StopSyncingVaultRequest) (api.StopSyncingVaultResponse, error) {
	return api.StopSyncingVaultResponse{}, v.stop.Execute(ctx, domain.VaultID(request.VaultID), request.DeleteCloudCopy)
}

// DeleteCloudVault deletes a synced vault of the account from the cloud. The files of the vaults that were synced stay.
func (v *Removals) DeleteCloudVault(ctx context.Context, request api.DeleteCloudVaultRequest) (api.DeleteCloudVaultResponse, error) {
	return api.DeleteCloudVaultResponse{}, v.delete.Execute(ctx, request.RemoteID)
}

// ListRemoteVaults lists the synced vaults of the account.
func (v *Connections) ListRemoteVaults(ctx context.Context, _ api.ListRemoteVaultsRequest) (api.ListRemoteVaultsResponse, error) {
	remote, err := catalog.NewListRemoteVaults(v.catalog).Execute(ctx)
	if err != nil {
		return api.ListRemoteVaultsResponse{Vaults: nil}, err
	}
	listed := make([]api.RemoteVault, len(remote))
	for i, one := range remote {
		listed[i] = api.RemoteVault{RemoteID: one.RemoteID, Name: one.Name, IsConnected: one.IsConnected, SizeBytes: one.SizeBytes}
	}
	return api.ListRemoteVaultsResponse{Vaults: listed}, nil
}

// ConnectVault connects a vault of the installation to a synced vault, or to a new one.
func (v *Connections) ConnectVault(ctx context.Context, request api.ConnectVaultRequest) (api.ConnectVaultResponse, error) {
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
	return api.ConnectVaultResponse{}, v.connect.Execute(ctx, input)
}

// GetConnectPreview counts the notes of a vault that have another text than the note at the same path of a synced vault.
func (v *Connections) GetConnectPreview(ctx context.Context, request api.PreviewConnectRequest) (api.PreviewConnectResponse, error) {
	if !vaultlink.IsRemoteID(request.RemoteID) {
		return api.PreviewConnectResponse{}, catalog.ErrUnknownRemoteVault
	}
	use := preview.NewPreviewConnect(preview.Deps{
		Heads: v.heads,
		Host:  v.deps.Host,
	})
	found, err := use.Execute(ctx, preview.Input{VaultID: domain.VaultID(request.VaultID), RemoteID: request.RemoteID})
	return api.PreviewConnectResponse{DifferCount: len(found.Differ)}, err
}

// GetStatus returns whether this device has an identity, whether the server answers and what warns.
func (v *Listing) GetStatus(ctx context.Context, _ api.GetStatusRequest) (api.GetStatusResponse, error) {
	status := api.GetStatusResponse{HasIdentity: true, IsReachable: false, Items: nil}
	_, err := v.deps.Identity.GetPublicIdentity(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		status.HasIdentity = false
		status.Items = append(status.Items, api.StatusItem{Key: keyNoIdentity, Text: "No identity on this device"})
		return status, nil
	}
	if err != nil {
		return status, err
	}
	_, err = v.deps.Chain.GetAccountChain(ctx)
	status.IsReachable = !errors.Is(err, syncprotocol.ErrUnreachable) && !errors.Is(err, sessiondomain.ErrUnreachable)
	count, err := v.countConflicts(ctx)
	if err != nil {
		return status, err
	}
	if count > 0 {
		status.Items = append(status.Items, api.StatusItem{Key: keyConflicts, Text: syncvault.GetConflictText(count)})
	}
	return status, nil
}

func (v *Listing) countConflicts(ctx context.Context) (int, error) {
	vaults, err := v.deps.Host.GetVaults(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, vault := range vaults {
		status, err := v.deps.Progress.ReadStatus(ctx, vault.ID)
		if err != nil {
			return 0, err
		}
		total += status.ConflictCount
	}
	return total, nil
}
