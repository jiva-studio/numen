package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

func mountShareMethods(service *server.API, r *runtime) {
	server.Handle(service, api.MethodGetReaderCode, func(ctx context.Context, request api.GetReaderCodeRequest) (api.GetReaderCodeResponse, error) {
		return handle(r, func(b *Backend) (api.GetReaderCodeResponse, error) { return b.GetReaderCode(ctx, request) })
	})
	server.Handle(service, api.MethodReadReaderCode, func(ctx context.Context, request api.ReadReaderCodeRequest) (api.ReadReaderCodeResponse, error) {
		return handle(r, func(b *Backend) (api.ReadReaderCodeResponse, error) { return b.ReadReaderCode(ctx, request) })
	})
	server.Handle(service, api.MethodGetShare, func(ctx context.Context, request api.GetShareRequest) (api.GetShareResponse, error) {
		return handle(r, func(b *Backend) (api.GetShareResponse, error) { return b.GetShare(ctx, request) })
	})
	server.Handle(service, api.MethodAddReader, func(ctx context.Context, request api.AddReaderRequest) (api.AddReaderResponse, error) {
		return handle(r, func(b *Backend) (api.AddReaderResponse, error) { return b.AddReader(ctx, request) })
	})
	server.Handle(service, api.MethodRemoveReader, func(ctx context.Context, request api.RemoveReaderRequest) (api.RemoveReaderResponse, error) {
		return handle(r, func(b *Backend) (api.RemoveReaderResponse, error) { return b.RemoveReader(ctx, request) })
	})
	server.Handle(service, api.MethodStopShare, func(ctx context.Context, request api.StopShareRequest) (api.StopShareResponse, error) {
		return handle(r, func(b *Backend) (api.StopShareResponse, error) { return b.StopShare(ctx, request) })
	})
	server.Handle(service, api.MethodReadInvitation, func(ctx context.Context, request api.ReadInvitationRequest) (api.ReadInvitationResponse, error) {
		return handle(r, func(b *Backend) (api.ReadInvitationResponse, error) { return b.ReadInvitation(ctx, request) })
	})
	server.Handle(service, api.MethodAcceptInvitation, func(ctx context.Context, request api.AcceptInvitationRequest) (api.AcceptInvitationResponse, error) {
		return handle(r, func(b *Backend) (api.AcceptInvitationResponse, error) { return b.AcceptInvitation(ctx, request) })
	})
	server.Handle(service, api.MethodListReceived, func(ctx context.Context, request api.ListReceivedRequest) (api.ListReceivedResponse, error) {
		return handle(r, func(b *Backend) (api.ListReceivedResponse, error) { return b.ListReceived(ctx, request) })
	})
}
