package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

func mountMethods(service *server.API, r *runtime) {
	mountCall(service, r, api.MethodListVaults, (*Backend).ListVaults)
	mountCall(service, r, api.MethodListRemoteVaults, (*Backend).ListRemoteVaults)
	server.Handle(service, api.MethodConnectVault, func(ctx context.Context, request api.ConnectVaultRequest) (api.ConnectVaultResponse, error) {
		return handle(ctx, r, api.MethodConnectVault, func(b *Backend) (api.ConnectVaultResponse, error) {
			response, err := b.ConnectVault(ctx, request)
			if err == nil {
				r.wake(domain.VaultID(request.VaultID))
			}
			return response, err
		})
	})
	mountCall(service, r, api.MethodStopSyncingVault, (*Backend).StopSyncingVault)
	mountCall(service, r, api.MethodDeleteCloudVault, (*Backend).DeleteCloudVault)
	mountCall(service, r, api.MethodPreviewConnect, (*Backend).GetConnectPreview)
	mountCall(service, r, api.MethodGetStatus, (*Backend).GetStatus)
	mountCall(service, r, api.MethodGetSubscription, (*Backend).GetSubscription)
	mountCall(service, r, api.MethodListPlans, (*Backend).ListPlans)
	mountCall(service, r, api.MethodResolveDeletions, (*Backend).ResolveDeletions)
	mountCall(service, r, api.MethodCreateCheckout, (*Backend).CreateCheckout)
	mountCall(service, r, api.MethodChangePlan, (*Backend).ChangePlan)
	mountCall(service, r, api.MethodCreatePortalSession, (*Backend).CreatePortalSession)
	mountCall(service, r, api.MethodGetAccountState, (*Backend).GetAccountState)
	mountCall(service, r, api.MethodDeleteCloudData, (*Backend).DeleteCloudData)
	mountCall(service, r, api.MethodRestoreAccount, (*Backend).RestoreAccount)
	mountCall(service, r, api.MethodExportAccountData, (*Backend).ExportAccountData)
	mountShareMethods(service, r)
	mountInviteMethods(service, r)
	mountCall(service, r, api.MethodSuspend, (*Backend).Suspend)
	mountCall(service, r, api.MethodFetchFile, (*Backend).FetchFile)
	mountCall(service, r, api.MethodDropDownload, (*Backend).DropDownload)
	mountCall(service, r, api.MethodGetDownloads, (*Backend).GetDownloads)
	mountCall(service, r, api.MethodResume, (*Backend).Resume)
}

func mountCall[Q, R any](service *server.API, r *runtime, method string, call func(*Backend, context.Context, Q) (R, error)) {
	server.Handle(service, method, func(ctx context.Context, request Q) (R, error) {
		return handle(ctx, r, method, func(b *Backend) (R, error) { return call(b, ctx, request) })
	})
}

func handle[T any](ctx context.Context, r *runtime, operation string, call func(b *Backend) (T, error)) (T, error) {
	backend, err := r.getBackend()
	if err == nil {
		var result T
		if result, err = call(backend); err == nil {
			return result, nil
		}
	}
	var none T
	code, text := r.failures.Describe(ctx, operation, err)
	return none, server.NewAPIError(code, text)
}

func mountShareMethods(service *server.API, r *runtime) {
	mountCall(service, r, api.MethodGetShare, (*Backend).GetShare)
	mountCall(service, r, api.MethodListShares, (*Backend).ListShares)
	mountCall(service, r, api.MethodRemoveReader, (*Backend).RemoveReader)
	mountCall(service, r, api.MethodStopShare, (*Backend).StopShare)
	mountCall(service, r, api.MethodListReceived, (*Backend).ListReceived)
	mountCall(service, r, api.MethodLeaveShare, (*Backend).LeaveShare)
}

func mountInviteMethods(service *server.API, r *runtime) {
	mountCall(service, r, api.MethodStartShareInvitation, (*Backend).StartShareInvitation)
	mountCall(service, r, api.MethodListShareInvitations, (*Backend).ListShareInvitations)
	mountCall(service, r, api.MethodRevokeShareInvitation, (*Backend).RevokeShareInvitation)
	mountCall(service, r, api.MethodListShareRequests, (*Backend).ListShareRequests)
	mountCall(service, r, api.MethodApproveShareRequest, (*Backend).ApproveShareRequest)
	mountCall(service, r, api.MethodDeclineShareRequest, (*Backend).DeclineShareRequest)
	mountCall(service, r, api.MethodPreviewShareInvitation, (*Backend).GetShareInvitationPreview)
	mountCall(service, r, api.MethodAcceptShareInvitation, (*Backend).AcceptShareInvitation)
	mountCall(service, r, api.MethodGetShareInvitationStatus, (*Backend).GetShareInvitationStatus)
}
