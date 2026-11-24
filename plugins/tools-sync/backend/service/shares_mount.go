package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

func mountShareMethods(service *server.API, r *runtime) {
	server.Handle(service, api.MethodGetShare, func(ctx context.Context, request api.GetShareRequest) (api.GetShareResponse, error) {
		return handle(ctx, r, api.MethodGetShare, func(b *Backend) (api.GetShareResponse, error) { return b.GetShare(ctx, request) })
	})
	server.Handle(service, api.MethodListShares, func(ctx context.Context, request api.ListSharesRequest) (api.ListSharesResponse, error) {
		return handle(ctx, r, api.MethodListShares, func(b *Backend) (api.ListSharesResponse, error) { return b.ListShares(ctx, request) })
	})
	server.Handle(service, api.MethodRemoveReader, func(ctx context.Context, request api.RemoveReaderRequest) (api.RemoveReaderResponse, error) {
		return handle(ctx, r, api.MethodRemoveReader, func(b *Backend) (api.RemoveReaderResponse, error) { return b.RemoveReader(ctx, request) })
	})
	server.Handle(service, api.MethodStopShare, func(ctx context.Context, request api.StopShareRequest) (api.StopShareResponse, error) {
		return handle(ctx, r, api.MethodStopShare, func(b *Backend) (api.StopShareResponse, error) { return b.StopShare(ctx, request) })
	})
	server.Handle(service, api.MethodListReceived, func(ctx context.Context, request api.ListReceivedRequest) (api.ListReceivedResponse, error) {
		return handle(ctx, r, api.MethodListReceived, func(b *Backend) (api.ListReceivedResponse, error) { return b.ListReceived(ctx, request) })
	})
	server.Handle(service, api.MethodLeaveShare, func(ctx context.Context, request api.LeaveShareRequest) (api.LeaveShareResponse, error) {
		return handle(ctx, r, api.MethodLeaveShare, func(b *Backend) (api.LeaveShareResponse, error) { return b.LeaveShare(ctx, request) })
	})
}

func mountInviteMethods(service *server.API, r *runtime) {
	server.Handle(service, api.MethodStartShareInvitation, func(ctx context.Context, request api.StartShareInvitationRequest) (api.StartShareInvitationResponse, error) {
		return handle(ctx, r, api.MethodStartShareInvitation, func(b *Backend) (api.StartShareInvitationResponse, error) {
			return b.StartShareInvitation(ctx, request)
		})
	})
	server.Handle(service, api.MethodListShareInvitations, func(ctx context.Context, request api.ListShareInvitationsRequest) (api.ListShareInvitationsResponse, error) {
		return handle(ctx, r, api.MethodListShareInvitations, func(b *Backend) (api.ListShareInvitationsResponse, error) {
			return b.ListShareInvitations(ctx, request)
		})
	})
	server.Handle(service, api.MethodRevokeShareInvitation, func(ctx context.Context, request api.RevokeShareInvitationRequest) (api.RevokeShareInvitationResponse, error) {
		return handle(ctx, r, api.MethodRevokeShareInvitation, func(b *Backend) (api.RevokeShareInvitationResponse, error) {
			return b.RevokeShareInvitation(ctx, request)
		})
	})
	server.Handle(service, api.MethodListShareRequests, func(ctx context.Context, request api.ListShareRequestsRequest) (api.ListShareRequestsResponse, error) {
		return handle(ctx, r, api.MethodListShareRequests, func(b *Backend) (api.ListShareRequestsResponse, error) { return b.ListShareRequests(ctx, request) })
	})
	server.Handle(service, api.MethodApproveShareRequest, func(ctx context.Context, request api.ApproveShareRequestRequest) (api.ApproveShareRequestResponse, error) {
		return handle(ctx, r, api.MethodApproveShareRequest, func(b *Backend) (api.ApproveShareRequestResponse, error) { return b.ApproveShareRequest(ctx, request) })
	})
	server.Handle(service, api.MethodDeclineShareRequest, func(ctx context.Context, request api.DeclineShareRequestRequest) (api.DeclineShareRequestResponse, error) {
		return handle(ctx, r, api.MethodDeclineShareRequest, func(b *Backend) (api.DeclineShareRequestResponse, error) { return b.DeclineShareRequest(ctx, request) })
	})
	server.Handle(service, api.MethodPreviewShareInvitation, func(ctx context.Context, request api.PreviewShareInvitationRequest) (api.PreviewShareInvitationResponse, error) {
		return handle(ctx, r, api.MethodPreviewShareInvitation, func(b *Backend) (api.PreviewShareInvitationResponse, error) {
			return b.GetShareInvitationPreview(ctx, request)
		})
	})
	server.Handle(service, api.MethodAcceptShareInvitation, func(ctx context.Context, request api.AcceptShareInvitationRequest) (api.AcceptShareInvitationResponse, error) {
		return handle(ctx, r, api.MethodAcceptShareInvitation, func(b *Backend) (api.AcceptShareInvitationResponse, error) {
			return b.AcceptShareInvitation(ctx, request)
		})
	})
	server.Handle(service, api.MethodGetShareInvitationStatus, func(ctx context.Context, request api.GetShareInvitationStatusRequest) (api.GetShareInvitationStatusResponse, error) {
		return handle(ctx, r, api.MethodGetShareInvitationStatus, func(b *Backend) (api.GetShareInvitationStatusResponse, error) {
			return b.GetShareInvitationStatus(ctx, request)
		})
	})
}
