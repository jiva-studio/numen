// Package sharewire builds the use cases of sharing and receiving a folder from the dependencies of a pass, and
// answers the methods of the settings view that share a folder and receive one.
package sharewire

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
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
		Notes: deps.Notes, Links: deps.State, Records: deps.State, Statuses: deps.Progress, Vaults: deps.Vaults, Folders: deps.Folders, Clock: deps.Clock,
	}
}

// Shares answers the methods of the settings view about the folders this account shares.
type Shares struct {
	get    *share.GetShare
	add    *share.ShareFolder
	remove *share.RemoveReader
	stop   *share.StopShare
}

// New returns the methods over the dependencies of a pass.
func New(deps syncvault.Deps) *Shares {
	sharing := NewShareDeps(deps)
	return &Shares{
		get: share.NewGetShare(sharing), add: share.NewShareFolder(sharing),
		remove: share.NewRemoveReader(sharing), stop: share.NewStopShare(sharing),
	}
}

// Received answers the methods of the settings view about the folders other people share with this account.
type Received struct {
	code    *receive.GetReaderCode
	read    *receive.ReadInvitation
	accept  *receive.AcceptInvitation
	listing *receive.ListReceived
}

// NewReceived returns the methods over the dependencies of a pass.
func NewReceived(deps syncvault.Deps) *Received {
	receiving := NewReceiveDeps(deps)
	return &Received{
		code: receive.NewGetReaderCode(receiving), read: receive.NewReadInvitation(receiving),
		accept: receive.NewAcceptInvitation(receiving), listing: receive.NewListReceived(receiving),
	}
}

// GetReaderCode returns the code of this account and its fingerprint.
func (s *Received) GetReaderCode(ctx context.Context, _ api.GetReaderCodeRequest) (api.GetReaderCodeResponse, error) {
	code, err := s.code.Execute(ctx)
	if err != nil {
		return api.GetReaderCodeResponse{}, err
	}
	return api.GetReaderCodeResponse{Code: code.Code, Fingerprint: code.Fingerprint}, nil
}

// ReadReaderCode returns the account fingerprint of the reader a code names.
func (s *Received) ReadReaderCode(_ context.Context, request api.ReadReaderCodeRequest) (api.ReadReaderCodeResponse, error) {
	chain, err := domainshare.ParseReaderCode(request.Code)
	if err != nil {
		return api.ReadReaderCodeResponse{}, err
	}
	return api.ReadReaderCodeResponse{Fingerprint: chain.Root().Fingerprint().String(), Label: ""}, nil
}

// GetShare returns the sharing of a folder.
func (s *Shares) GetShare(ctx context.Context, request api.GetShareRequest) (api.GetShareResponse, error) {
	overview, err := s.get.Execute(ctx, domain.VaultID(request.VaultID), request.Folder)
	if err != nil {
		return api.GetShareResponse{}, err
	}
	readers := make([]api.ShareReader, len(overview.Readers))
	for i, reader := range overview.Readers {
		readers[i] = api.ShareReader{Label: reader.Label, Fingerprint: reader.Fingerprint, State: api.ReaderState(reader.GetState())}
	}
	return api.GetShareResponse{State: api.ShareState(overview.State), Readers: readers, Error: ""}, nil
}

// AddReader shares the folder with the reader of the code and returns the invitation.
func (s *Shares) AddReader(ctx context.Context, request api.AddReaderRequest) (api.AddReaderResponse, error) {
	invited, err := s.add.Execute(ctx, share.AddReaderInput{
		VaultID: domain.VaultID(request.VaultID), Folder: request.Folder, Code: request.Code, Label: request.Label, IsConfirmed: request.ConfirmChange,
	})
	if err != nil {
		return api.AddReaderResponse{}, err
	}
	return api.AddReaderResponse{Invitation: invited.Text, Fingerprint: invited.Fingerprint}, nil
}

func (s *Shares) RemoveReader(ctx context.Context, request api.RemoveReaderRequest) (api.RemoveReaderResponse, error) {
	return api.RemoveReaderResponse{}, s.remove.Execute(ctx, domain.VaultID(request.VaultID), request.Folder, request.Fingerprint)
}

// StopShare stops sharing the folder.
func (s *Shares) StopShare(ctx context.Context, request api.StopShareRequest) (api.StopShareResponse, error) {
	return api.StopShareResponse{}, s.stop.Execute(ctx, domain.VaultID(request.VaultID), request.Folder)
}

// ReadInvitation shows what an invitation offers.
func (s *Received) ReadInvitation(ctx context.Context, request api.ReadInvitationRequest) (api.ReadInvitationResponse, error) {
	preview, err := s.read.Execute(ctx, request.Text)
	if err != nil {
		return api.ReadInvitationResponse{}, err
	}
	return api.ReadInvitationResponse{
		ShareName: preview.ShareName, OwnerFingerprint: preview.OwnerFingerprint, IsForThisAccount: preview.IsForThisAccount,
		AlreadyReceived: preview.IsReceived,
	}, nil
}

// AcceptInvitation receives the folder of an invitation as a vault.
func (s *Received) AcceptInvitation(ctx context.Context, request api.AcceptInvitationRequest) (api.AcceptInvitationResponse, error) {
	_, err := s.accept.Execute(ctx, request.Text)
	return api.AcceptInvitationResponse{}, err
}

func (s *Received) ListReceived(ctx context.Context, _ api.ListReceivedRequest) (api.ListReceivedResponse, error) {
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
