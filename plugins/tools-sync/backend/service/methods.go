package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
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
		return handle(r, func(b *Backend) (api.PreviewConnectResponse, error) { return b.PreviewConnect(ctx, request) })
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
}

func handle[T any](r *runtime, call func(b *Backend) (T, error)) (T, error) {
	backend, err := r.getBackend()
	if err != nil {
		var none T
		return none, server.NewAPIError(server.CodeUnavailable, err.Error())
	}
	result, err := call(backend)
	if err != nil {
		return result, server.NewAPIError(getCode(err), getText(err))
	}
	return result, nil
}

func getCode(err error) server.ErrorCode {
	switch {
	case errors.Is(err, syncprotocol.ErrUnreachable), errors.Is(err, sessiondomain.ErrUnreachable):
		return server.CodeUnavailable
	case errors.Is(err, syncprotocol.ErrAlreadySubscribed), errors.Is(err, syncprotocol.ErrSamePlan):
		return server.CodeAlreadyExists
	case errors.Is(err, syncprotocol.ErrUnknownPlan), errors.Is(err, share.ErrBadCode):
		return server.CodeInvalidArgument
	case errors.Is(err, share.ErrUnknownShare):
		return server.CodeNotFound
	case isRefusal(err):
		return server.CodeFailedPrecondition
	case isAccountDeleting(err):
		return server.CodeFailedPrecondition
	case errors.Is(err, sessiondomain.ErrNoIdentity), errors.Is(err, syncprotocol.ErrBillingUnavailable), errors.Is(err, syncprotocol.ErrNoActiveSubscription):
		return server.CodeFailedPrecondition
	}
	return server.CodeInternal
}

func getRefusals() []error {
	return []error{
		share.ErrWholeVault, share.ErrInsideShare, share.ErrHoldsShare, share.ErrNotConnected, share.ErrReceivedVault,
		share.ErrReceivedNotSynced, share.ErrOwnIdentity, share.ErrAlreadyReader, share.ErrChangedIdentity, share.ErrReaderNotAdded,
		share.ErrNotForThisAccount, share.ErrAlreadyReceived, share.ErrOwnerChanged, share.ErrEnded, share.ErrDamagedLog, share.ErrNotReady,
		share.ErrUnknownReader, share.ErrNotShared, share.ErrRolledBack,
	}
}

func isAccountDeleting(err error) bool {
	return errors.Is(err, sessiondomain.AccountDeletingError{})
}

func isRefusal(err error) bool { return findRefusal(err) != nil }

func findRefusal(err error) error {
	for _, refusal := range getRefusals() {
		if errors.Is(err, refusal) {
			return refusal
		}
	}
	return nil
}

func getText(err error) string {
	refusal := findRefusal(err)
	if refusal == nil {
		return err.Error()
	}
	text := refusal.Error()
	return strings.ToUpper(text[:1]) + text[1:]
}
