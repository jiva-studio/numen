package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

const (
	maxSignedMessage = 4096
	noIdentityReason = "NO_IDENTITY"
	noIdentityDomain = "numen.v1.HostAccountService"
)

var (
	errNoAccount      = errors.New("the host keeps no identity for plugins")
	errMessageSize    = errors.New("the message to sign has 1 to 4096 bytes")
	errAccountFailure = errors.New("the identity of this device could not be read")
)

// accountEndpoint answers the calls of HostAccountService. The plugin is the one the token names; the
// request names none. Both calls need the permission identity.
type accountEndpoint struct {
	numenv1connect.UnimplementedHostAccountServiceHandler

	endpoint *Endpoint
}

func (e *accountEndpoint) SignAsDevice(
	ctx context.Context, req *connect.Request[v1.SignAsDeviceRequest],
) (*connect.Response[v1.SignAsDeviceResponse], error) {
	pluginID, err := e.endpoint.Authorize(req.Header().Get("Authorization"), manifest.PermissionIdentity)
	if err != nil {
		return nil, err
	}
	account := e.endpoint.callers.Account
	if account == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	message := req.Msg.GetMessage()
	if len(message) == 0 || len(message) > maxSignedMessage {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMessageSize)
	}
	signature, err := account.SignAsDevice(ctx, pluginID, message)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.SignAsDeviceResponse{Signature: signature}), nil
}

func (e *accountEndpoint) GetPublicIdentity(
	ctx context.Context, req *connect.Request[v1.GetPublicIdentityRequest],
) (*connect.Response[v1.GetPublicIdentityResponse], error) {
	if _, err := e.endpoint.Authorize(req.Header().Get("Authorization"), manifest.PermissionIdentity); err != nil {
		return nil, err
	}
	account := e.endpoint.callers.Account
	if account == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	public, err := account.GetPublicIdentity(ctx)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.GetPublicIdentityResponse{
		AccountFingerprint: public.AccountFingerprint,
		DeviceFingerprint:  public.DeviceFingerprint,
		DeviceSignKey:      public.DeviceSignKey,
		Chain:              public.Chain,
	}), nil
}

func getAccountError(err error) error {
	if errors.Is(err, identitydomain.ErrNoIdentity) {
		failure := connect.NewError(connect.CodeFailedPrecondition, identitydomain.ErrNoIdentity)
		if detail, detailErr := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: noIdentityReason, Domain: noIdentityDomain}); detailErr == nil {
			failure.AddDetail(detail)
		}
		return failure
	}
	return connect.NewError(connect.CodeInternal, errAccountFailure)
}
