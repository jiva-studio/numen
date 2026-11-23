package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

// GetReaderCode returns the code of this account for the owner of a share.
func (b *Backend) GetReaderCode(ctx context.Context, request api.GetReaderCodeRequest) (api.GetReaderCodeResponse, error) {
	return b.shares.GetReaderCode(ctx, request)
}

// ReadReaderCode returns the account fingerprint a reader code names.
func (b *Backend) ReadReaderCode(ctx context.Context, request api.ReadReaderCodeRequest) (api.ReadReaderCodeResponse, error) {
	return b.shares.ReadReaderCode(ctx, request)
}

// GetShare returns the sharing of a folder.
func (b *Backend) GetShare(ctx context.Context, request api.GetShareRequest) (api.GetShareResponse, error) {
	return b.shares.GetShare(ctx, request)
}

// AddReader shares a folder with a reader and returns the invitation.
func (b *Backend) AddReader(ctx context.Context, request api.AddReaderRequest) (api.AddReaderResponse, error) {
	return b.shares.AddReader(ctx, request)
}

func (b *Backend) RemoveReader(ctx context.Context, request api.RemoveReaderRequest) (api.RemoveReaderResponse, error) {
	return b.shares.RemoveReader(ctx, request)
}

// StopShare stops sharing a folder.
func (b *Backend) StopShare(ctx context.Context, request api.StopShareRequest) (api.StopShareResponse, error) {
	return b.shares.StopShare(ctx, request)
}

// ReadInvitation shows what an invitation offers.
func (b *Backend) ReadInvitation(ctx context.Context, request api.ReadInvitationRequest) (api.ReadInvitationResponse, error) {
	return b.shares.ReadInvitation(ctx, request)
}

// AcceptInvitation receives the folder of an invitation as a vault.
func (b *Backend) AcceptInvitation(ctx context.Context, request api.AcceptInvitationRequest) (api.AcceptInvitationResponse, error) {
	return b.shares.AcceptInvitation(ctx, request)
}

// ListReceived lists the shares this computer received.
func (b *Backend) ListReceived(ctx context.Context, request api.ListReceivedRequest) (api.ListReceivedResponse, error) {
	return b.shares.ListReceived(ctx, request)
}
