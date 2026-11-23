// Package license serves a window's LicenseService: listing the licenses declared by plugins,
// activating and deactivating a machine license key, and streaming license updates.
package license

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	licenses "github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// UseCases are the use cases LicenseService is answered through.
type UseCases struct {
	List       *licenses.List
	Activate   *licenses.Activate
	Deactivate *licenses.Deactivate
	Watch      *licenses.Watch
}

// Handler serves LicenseService over the license use cases.
type Handler struct {
	licenses UseCases
}

// NewHandler returns the handler over the license use cases of this installation.
func NewHandler(useCases UseCases) *Handler {
	return &Handler{licenses: useCases}
}

// ListLicenses returns every license declared by the installed plugins with its current state on
// this machine.
func (h *Handler) ListLicenses(
	ctx context.Context,
	_ *connect.Request[v1.ListLicensesRequest],
) (*connect.Response[v1.ListLicensesResponse], error) {
	entries, err := h.licenses.List.Execute(ctx)
	if err != nil {
		return nil, wire.NewInternalError(err)
	}
	return connect.NewResponse(&v1.ListLicensesResponse{
		Licenses: writeLicenses(entries),
	}), nil
}

// ActivateLicense activates a license key on this machine.
func (h *Handler) ActivateLicense(
	ctx context.Context,
	r *connect.Request[v1.ActivateLicenseRequest],
) (*connect.Response[v1.ActivateLicenseResponse], error) {
	info, err := h.licenses.Activate.Execute(ctx, licenses.ActivateRequest{
		LicenseID: r.Msg.GetLicenseId(),
		Key:       r.Msg.GetKey(),
	})
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, wire.NewInternalError(err)
		}
		return connect.NewResponse(&v1.ActivateLicenseResponse{
			Error:  code,
			Detail: wire.GetErrorDetail(err, detailed...),
		}), nil
	}
	return connect.NewResponse(&v1.ActivateLicenseResponse{
		License: writeLicense(info),
	}), nil
}

// DeactivateLicense releases this machine's activation slot and removes the stored key once
// confirmed by the provider.
func (h *Handler) DeactivateLicense(
	ctx context.Context,
	r *connect.Request[v1.DeactivateLicenseRequest],
) (*connect.Response[v1.DeactivateLicenseResponse], error) {
	info, err := h.licenses.Deactivate.Execute(ctx, r.Msg.GetLicenseId())
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, wire.NewInternalError(err)
		}
		return connect.NewResponse(&v1.DeactivateLicenseResponse{
			Error:  code,
			Detail: wire.GetErrorDetail(err, detailed...),
		}), nil
	}
	return connect.NewResponse(&v1.DeactivateLicenseResponse{
		License: writeLicense(info),
	}), nil
}

// WatchLicenses streams the full list of licenses when the stream opens and again whenever a
// license or the set of plugins changes.
func (h *Handler) WatchLicenses(
	ctx context.Context,
	_ *connect.Request[v1.WatchLicensesRequest],
	out *connect.ServerStream[v1.WatchLicensesResponse],
) error {
	return wire.RunWatch(ctx, out, wire.ResendInterval, wire.GetLastOrEmpty, func(ctx context.Context, send func(*v1.WatchLicensesResponse) error) error {
		return h.licenses.Watch.Execute(ctx, func(entries []license.Info) error {
			return send(&v1.WatchLicensesResponse{Licenses: writeLicenses(entries)})
		})
	})
}

var detailed = []error{
	license.ErrNotFound,
	license.ErrInvalidKey,
	license.ErrActivationLimitReached,
	license.ErrExpired,
	license.ErrDisabled,
	license.ErrProductMismatch,
	license.ErrProviderUnavailable,
	license.ErrKeyStore,
}

func getErrorCode(err error) (v1.LicenseErrorCode, bool) {
	switch {
	case errors.Is(err, license.ErrNotFound):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, license.ErrInvalidKey):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_INVALID_KEY, true
	case errors.Is(err, license.ErrActivationLimitReached):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_ACTIVATION_LIMIT_REACHED, true
	case errors.Is(err, license.ErrExpired):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_EXPIRED, true
	case errors.Is(err, license.ErrDisabled):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_DISABLED, true
	case errors.Is(err, license.ErrProductMismatch):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_PRODUCT_MISMATCH, true
	case errors.Is(err, license.ErrProviderUnavailable):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_PROVIDER_UNAVAILABLE, true
	case errors.Is(err, license.ErrKeyStore):
		return v1.LicenseErrorCode_LICENSE_ERROR_CODE_KEY_STORE_FAILED, true
	}
	return v1.LicenseErrorCode_LICENSE_ERROR_CODE_UNSPECIFIED, false
}
