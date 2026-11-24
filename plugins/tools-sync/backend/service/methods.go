package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
)

func mountMethods(service *server.API, r *runtime) {
	server.Handle(service, api.MethodListVaults, func(ctx context.Context, request api.ListVaultsRequest) (api.ListVaultsResponse, error) {
		return handle(r, func(b *Backend) (api.ListVaultsResponse, error) { return b.ListVaults(ctx, request) })
	})
	server.Handle(service, api.MethodListRemoteVaults, func(ctx context.Context, request api.ListRemoteVaultsRequest) (api.ListRemoteVaultsResponse, error) {
		return handle(r, func(b *Backend) (api.ListRemoteVaultsResponse, error) { return b.ListRemoteVaults(ctx, request) })
	})
	server.Handle(service, api.MethodConnectVault, func(ctx context.Context, request api.ConnectVaultRequest) (api.ConnectVaultResponse, error) {
		return handle(r, func(b *Backend) (api.ConnectVaultResponse, error) { return b.ConnectVault(ctx, request) })
	})
	server.Handle(service, api.MethodPreviewConnect, func(ctx context.Context, request api.PreviewConnectRequest) (api.PreviewConnectResponse, error) {
		return handle(r, func(b *Backend) (api.PreviewConnectResponse, error) { return b.GetConnectPreview(ctx, request) })
	})
	server.Handle(service, api.MethodGetStatus, func(ctx context.Context, request api.GetStatusRequest) (api.GetStatusResponse, error) {
		return handle(r, func(b *Backend) (api.GetStatusResponse, error) { return b.GetStatus(ctx, request) })
	})
	server.Handle(service, api.MethodGetSubscription, func(ctx context.Context, request api.GetSubscriptionRequest) (api.GetSubscriptionResponse, error) {
		return handle(r, func(b *Backend) (api.GetSubscriptionResponse, error) { return b.GetSubscription(ctx, request) })
	})
	server.Handle(service, api.MethodListPlans, func(ctx context.Context, request api.ListPlansRequest) (api.ListPlansResponse, error) {
		return handle(r, func(b *Backend) (api.ListPlansResponse, error) { return b.ListPlans(ctx, request) })
	})
	server.Handle(service, api.MethodResolveDeletions, func(ctx context.Context, request api.ResolveDeletionsRequest) (api.ResolveDeletionsResponse, error) {
		return handle(r, func(b *Backend) (api.ResolveDeletionsResponse, error) { return b.ResolveDeletions(ctx, request) })
	})
	server.Handle(service, api.MethodCreateCheckout, func(ctx context.Context, request api.CreateCheckoutRequest) (api.CreateCheckoutResponse, error) {
		return handle(r, func(b *Backend) (api.CreateCheckoutResponse, error) { return b.CreateCheckout(ctx, request) })
	})
	server.Handle(service, api.MethodChangePlan, func(ctx context.Context, request api.ChangePlanRequest) (api.ChangePlanResponse, error) {
		return handle(r, func(b *Backend) (api.ChangePlanResponse, error) { return b.ChangePlan(ctx, request) })
	})
	server.Handle(service, api.MethodCreatePortalSession, func(ctx context.Context, request api.CreatePortalSessionRequest) (api.CreatePortalSessionResponse, error) {
		return handle(r, func(b *Backend) (api.CreatePortalSessionResponse, error) { return b.CreatePortalSession(ctx, request) })
	})
	server.Handle(service, api.MethodGetAccountState, func(ctx context.Context, request api.GetAccountStateRequest) (api.GetAccountStateResponse, error) {
		return handle(r, func(b *Backend) (api.GetAccountStateResponse, error) { return b.GetAccountState(ctx, request) })
	})
	server.Handle(service, api.MethodDeleteCloudData, func(ctx context.Context, request api.DeleteCloudDataRequest) (api.DeleteCloudDataResponse, error) {
		return handle(r, func(b *Backend) (api.DeleteCloudDataResponse, error) { return b.DeleteCloudData(ctx, request) })
	})
	server.Handle(service, api.MethodRestoreAccount, func(ctx context.Context, request api.RestoreAccountRequest) (api.RestoreAccountResponse, error) {
		return handle(r, func(b *Backend) (api.RestoreAccountResponse, error) { return b.RestoreAccount(ctx, request) })
	})
	server.Handle(service, api.MethodExportAccountData, func(ctx context.Context, request api.ExportAccountDataRequest) (api.ExportAccountDataResponse, error) {
		return handle(r, func(b *Backend) (api.ExportAccountDataResponse, error) { return b.ExportAccountData(ctx, request) })
	})
	mountShareMethods(service, r)
	server.Handle(service, api.MethodSuspend, func(ctx context.Context, request api.SuspendRequest) (api.SuspendResponse, error) {
		return handle(r, func(b *Backend) (api.SuspendResponse, error) { return b.Suspend(ctx, request) })
	})
	server.Handle(service, api.MethodFetchFile, func(ctx context.Context, request api.FetchFileRequest) (api.FetchFileResponse, error) {
		return handle(r, func(b *Backend) (api.FetchFileResponse, error) { return b.FetchFile(ctx, request) })
	})
	server.Handle(service, api.MethodDropDownload, func(ctx context.Context, request api.DropDownloadRequest) (api.DropDownloadResponse, error) {
		return handle(r, func(b *Backend) (api.DropDownloadResponse, error) { return b.DropDownload(ctx, request) })
	})
	server.Handle(service, api.MethodGetDownloads, func(ctx context.Context, request api.GetDownloadsRequest) (api.GetDownloadsResponse, error) {
		return handle(r, func(b *Backend) (api.GetDownloadsResponse, error) { return b.GetDownloads(ctx, request) })
	})
	server.Handle(service, api.MethodResume, func(ctx context.Context, request api.ResumeRequest) (api.ResumeResponse, error) {
		return handle(r, func(b *Backend) (api.ResumeResponse, error) { return b.Resume(ctx, request) })
	})
}

func handle[T any](r *runtime, call func(b *Backend) (T, error)) (T, error) {
	backend, err := r.getBackend()
	if err != nil {
		var none T
		return none, server.NewAPIError(server.CodeUnavailable, err.Error())
	}
	result, err := call(backend)
	if err != nil {
		return result, server.NewAPIError(errorwire.GetCode(err), errorwire.GetText(err))
	}
	return result, nil
}
