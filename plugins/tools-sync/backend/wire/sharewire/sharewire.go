// Package sharewire builds the use cases of sharing and receiving a folder from the dependencies of a pass, and
// answers the methods of the settings view that share a folder and receive one.
package sharewire

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/unlink"
)

// NewShareDeps returns the dependencies of the use cases of sharing over those of a pass.
func NewShareDeps(deps syncvault.Deps) share.Deps {
	return share.Deps{
		Shares: deps.Shares, Revisions: deps.Revisions, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity,
		Links: deps.State, States: deps.ShareStates, Clock: deps.Clock, Random: deps.Random,
	}
}

// NewReceiveDeps returns the dependencies of the use cases of receiving over those of a pass.
func NewReceiveDeps(deps syncvault.Deps) receive.Deps {
	return receive.Deps{
		Shares: deps.Shares, Blobs: deps.Blobs, Crypto: deps.Crypto, Box: deps.Box, Identity: deps.Identity, Host: deps.Host,
		Notes: deps.Notes, Links: deps.State, Records: deps.State, Statuses: deps.Progress, Vaults: deps.Vaults, Folders: deps.Folders, Clock: deps.Clock, Locker: deps.Locker,
	}
}

// Shares answers the methods of the settings view about the folders this account shares.
type Shares struct {
	host   ports.Host
	read   *share.ReadShares
	get    *share.GetShare
	remove *share.RemoveReader
	stop   *share.StopShare
}

// New returns the methods over the dependencies of a pass.
func New(deps syncvault.Deps) *Shares {
	sharing := NewShareDeps(deps)
	return &Shares{
		host: deps.Host, read: share.NewReadShares(sharing),
		get:    share.NewGetShare(sharing),
		remove: share.NewRemoveReader(sharing), stop: share.NewStopShare(sharing),
	}
}

// ReceivedShares answers the methods of the settings view about the folders other people share with this account.
type ReceivedShares struct {
	listing *receive.ListReceivedShares
	leave   *unlink.Leave
}

// NewReceivedShares returns the methods over the dependencies of a pass.
func NewReceivedShares(deps syncvault.Deps) *ReceivedShares {
	receiving := NewReceiveDeps(deps)
	return &ReceivedShares{
		listing: receive.NewListReceivedShares(receiving), leave: unlink.NewLeave(unlink.Deps{Links: deps.State, Forget: deps.Forget, Vaults: deps.Vaults, Cloud: nil, Locker: deps.Locker}),
	}
}

// GetShare returns the sharing of a folder.
func (s *Shares) GetShare(ctx context.Context, request api.GetShareRequest) (api.GetShareResponse, error) {
	overview, err := s.get.Execute(ctx, domain.VaultID(request.VaultID), request.Folder)
	if err != nil {
		return api.GetShareResponse{}, err
	}
	return api.GetShareResponse{State: api.ShareState(overview.State), Readers: getReaders(overview.Readers), Error: ""}, nil
}

func (s *Shares) RemoveReader(ctx context.Context, request api.RemoveReaderRequest) (api.RemoveReaderResponse, error) {
	return api.RemoveReaderResponse{}, s.remove.Execute(ctx, domain.VaultID(request.VaultID), request.Folder, request.Fingerprint)
}

// StopShare stops sharing the folder.
func (s *Shares) StopShare(ctx context.Context, request api.StopShareRequest) (api.StopShareResponse, error) {
	return api.StopShareResponse{}, s.stop.Execute(ctx, domain.VaultID(request.VaultID), request.Folder)
}

// LeaveShare stops pulling a received share and forgets it on this device; its files stay.
func (s *ReceivedShares) LeaveShare(ctx context.Context, request api.LeaveShareRequest) (api.LeaveShareResponse, error) {
	return api.LeaveShareResponse{}, s.leave.Execute(ctx, domain.VaultID(request.VaultID))
}

func (s *ReceivedShares) ListReceived(ctx context.Context, _ api.ListReceivedRequest) (api.ListReceivedResponse, error) {
	received, err := s.listing.Execute(ctx)
	if err != nil {
		return api.ListReceivedResponse{}, err
	}
	shares := make([]api.ReceivedShare, len(received))
	for i, one := range received {
		lastSynced := ""
		if !one.LastSyncedAt.IsZero() {
			lastSynced = one.LastSyncedAt.UTC().Format(time.RFC3339)
		}
		shares[i] = api.ReceivedShare{
			VaultID: string(one.VaultID), Name: one.Name, OwnerFingerprint: one.OwnerFingerprint, State: api.ReceivedState(one.State),
			LastSyncedAt: lastSynced,
		}
	}
	return api.ListReceivedResponse{Shares: shares}, nil
}

// ListShares returns the open shares of every vault of this account.
func (s *Shares) ListShares(ctx context.Context, _ api.ListSharesRequest) (api.ListSharesResponse, error) {
	vaults, err := s.host.GetVaults(ctx)
	if err != nil {
		return api.ListSharesResponse{}, err
	}
	shares := []api.SharedFolder{}
	for _, vault := range vaults {
		roots, err := s.read.Execute(ctx, vault.ID)
		if err != nil {
			return api.ListSharesResponse{}, err
		}
		for _, root := range roots {
			if root.IsClosed {
				continue
			}
			shares = append(shares, api.SharedFolder{
				VaultID: string(vault.ID), Folder: root.Folder, State: api.ShareState(root.GetState()), Readers: getReaders(root.Readers),
			})
		}
	}
	return api.ListSharesResponse{Shares: shares}, nil
}

func getReaders(readers []domainshare.Reader) []api.ShareReader {
	listed := make([]api.ShareReader, len(readers))
	for i, reader := range readers {
		listed[i] = api.ShareReader{Label: reader.Label, Fingerprint: reader.Fingerprint, State: api.ReaderState(reader.GetState())}
	}
	return listed
}
