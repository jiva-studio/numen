package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
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
	server.Handle(service, api.MethodGetStatus, func(ctx context.Context, request api.GetStatusRequest) (api.GetStatusResponse, error) {
		return handle(r, func(b *Backend) (api.GetStatusResponse, error) { return b.GetStatus(ctx, request) })
	})
	server.Handle(service, api.MethodGetSubscription, func(ctx context.Context, request api.GetSubscriptionRequest) (api.GetSubscriptionResponse, error) {
		return handle(r, func(b *Backend) (api.GetSubscriptionResponse, error) { return b.GetSubscription(ctx, request) })
	})
	server.Handle(service, api.MethodCreateCheckout, func(ctx context.Context, request api.CreateCheckoutRequest) (api.CreateCheckoutResponse, error) {
		return handle(r, func(b *Backend) (api.CreateCheckoutResponse, error) { return b.CreateCheckout(ctx, request) })
	})
	server.Handle(service, api.MethodSuspend, func(ctx context.Context, request api.SuspendRequest) (api.SuspendResponse, error) {
		return handle(r, func(b *Backend) (api.SuspendResponse, error) { return b.Suspend(ctx, request) })
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
		return result, server.NewAPIError(getCode(err), err.Error())
	}
	return result, nil
}

func getCode(err error) server.ErrorCode {
	switch {
	case errors.Is(err, syncprotocol.ErrUnreachable), errors.Is(err, sessiondomain.ErrUnreachable):
		return server.CodeUnavailable
	case errors.Is(err, syncprotocol.ErrAlreadySubscribed):
		return server.CodeAlreadyExists
	case errors.Is(err, host.ErrChanged):
		return server.CodeFailedPrecondition
	case errors.Is(err, sessiondomain.ErrNoIdentity), errors.Is(err, syncprotocol.ErrBillingUnavailable):
		return server.CodeFailedPrecondition
	}
	return server.CodeInternal
}

func getWarning(err error) string {
	switch {
	case errors.Is(err, syncprotocol.ErrUnreachable), errors.Is(err, sessiondomain.ErrUnreachable):
		return "Could not reach the sync server"
	case errors.Is(err, fetchfile.ErrBlobMissing):
		return "Could not download the file"
	case errors.Is(err, host.ErrChanged):
		return "The file has changes that are not sent"
	}
	return ""
}
