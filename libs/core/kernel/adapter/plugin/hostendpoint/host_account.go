package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

const (
	maxSignedMessage = 4096
	noIdentityReason = "NO_IDENTITY"
	noIdentityDomain = "numen.v1.HostAccountService"

	cannotOpenReason       = "CANNOT_OPEN"
	noPendingRestoreReason = "NO_PENDING_RESTORE"
	invalidChainReason     = "INVALID_CHAIN"
	identityExistsReason   = "IDENTITY_EXISTS"
	noLinkReason           = "NO_LINK"
)

var (
	errNoAccount      = errors.New("the host keeps no identity for plugins")
	errMessageSize    = errors.New("the message to sign has 1 to 4096 bytes")
	errAccountFailure = errors.New("the identity of this device could not be read")
)

// accountEndpoint answers the calls of HostAccountService. The plugin is the one the token names; the
// request names none. Every call needs the permission identity, or identity:link for the link, chain and restore calls.
type accountEndpoint struct {
	numenv1connect.UnimplementedHostAccountServiceHandler

	auth  authority
	calls AccountCalls
}

// authority tells which plugin made a call.
type authority interface {
	Authenticate(header string) (string, error)
}

func (e *accountEndpoint) SignAsDevice(
	ctx context.Context, req *connect.Request[v1.SignAsDeviceRequest],
) (*connect.Response[v1.SignAsDeviceResponse], error) {
	pluginID, err := e.auth.Authenticate(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	account := e.calls.Account
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
	if _, err := e.auth.Authenticate(req.Header().Get("Authorization")); err != nil {
		return nil, err
	}
	account := e.calls.Account
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
		KeyEpoch:           public.KeyEpoch,
		ChainLength:        uint32(public.ChainLength),
		IsRevoked:          public.IsRevoked,
	}), nil
}

type accountRefusal struct {
	target error
	code   connect.Code
	reason string
}

func getAccountRefusals() []accountRefusal {
	return []accountRefusal{
		{identitydomain.ErrNoIdentity, connect.CodeFailedPrecondition, noIdentityReason},
		{identitydomain.ErrNoChain, connect.CodeFailedPrecondition, noIdentityReason},
		{identitydomain.ErrCannotOpen, connect.CodeFailedPrecondition, cannotOpenReason},
		{identitydomain.ErrNoPendingRestore, connect.CodeFailedPrecondition, noPendingRestoreReason},
		{identitydomain.ErrInvalidChain, connect.CodeInvalidArgument, invalidChainReason},
		{identitydomain.ErrIdentityExists, connect.CodeFailedPrecondition, identityExistsReason},
		{identitydomain.ErrNoLink, connect.CodeFailedPrecondition, noLinkReason},
	}
}

func getAccountError(err error) error {
	for _, refusal := range getAccountRefusals() {
		if errors.Is(err, refusal.target) {
			failure := connect.NewError(refusal.code, refusal.target)
			if detail, detailErr := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: refusal.reason, Domain: noIdentityDomain}); detailErr == nil {
				failure.AddDetail(detail)
			}
			return failure
		}
	}
	return connect.NewError(connect.CodeInternal, errAccountFailure)
}
