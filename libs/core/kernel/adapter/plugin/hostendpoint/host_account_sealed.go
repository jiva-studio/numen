package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const (
	maxSealPlain  = 32 << 10
	maxOpenSealed = maxSealPlain + 256
	maxBoxBytes   = 1 << 10
)

var (
	errDataSize = errors.New("the data to seal has at most 32 KiB")
	errBoxSize  = errors.New("the box to open has at most 1 KiB")
)

var errLinkSize = errors.New("a link call carries 1 to 262144 bytes, a channel 16")

func (e *accountEndpoint) getAccountOf(header http.Header) (string, Account, error) {
	pluginID, err := e.auth.Authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	account := e.calls.Account
	if account == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	return pluginID, account, nil
}

func (e *accountEndpoint) getRestoreOf(header http.Header) (string, AccountRestore, error) {
	pluginID, err := e.auth.Authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	restore := e.calls.Restore
	if restore == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	return pluginID, restore, nil
}

func (e *accountEndpoint) SealData(
	ctx context.Context, req *connect.Request[v1.SealDataRequest],
) (*connect.Response[v1.SealDataResponse], error) {
	pluginID, account, err := e.getAccountOf(req.Header())
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetPlain()) > maxSealPlain {
		return nil, connect.NewError(connect.CodeInvalidArgument, errDataSize)
	}
	sealed, err := account.SealData(ctx, pluginID, req.Msg.GetContext(), req.Msg.GetPlain())
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.SealDataResponse{Sealed: sealed}), nil
}

func (e *accountEndpoint) OpenData(
	ctx context.Context, req *connect.Request[v1.OpenDataRequest],
) (*connect.Response[v1.OpenDataResponse], error) {
	pluginID, account, err := e.getAccountOf(req.Header())
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetSealed()) > maxOpenSealed {
		return nil, connect.NewError(connect.CodeInvalidArgument, errDataSize)
	}
	plain, err := account.OpenData(ctx, pluginID, req.Msg.GetContext(), req.Msg.GetSealed())
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.OpenDataResponse{Plain: plain}), nil
}

func (e *accountEndpoint) GetPendingRestore(
	ctx context.Context, req *connect.Request[v1.GetPendingRestoreRequest],
) (*connect.Response[v1.GetPendingRestoreResponse], error) {
	_, restore, err := e.getRestoreOf(req.Header())
	if err != nil {
		return nil, err
	}
	pending, err := restore.GetPendingRestore(ctx)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.GetPendingRestoreResponse{
		IsPending:     pending.IsPending,
		RootSignKey:   pending.RootSignKey,
		DeviceSignKey: pending.DeviceSignKey,
	}), nil
}

func (e *accountEndpoint) SignAsRecovery(
	ctx context.Context, req *connect.Request[v1.SignAsRecoveryRequest],
) (*connect.Response[v1.SignAsRecoveryResponse], error) {
	pluginID, restore, err := e.getRestoreOf(req.Header())
	if err != nil {
		return nil, err
	}
	message := req.Msg.GetMessage()
	if len(message) == 0 || len(message) > maxSignedMessage {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMessageSize)
	}
	signature, err := restore.SignAsRecovery(ctx, pluginID, message)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.SignAsRecoveryResponse{Signature: signature}), nil
}

func (e *accountEndpoint) CompleteRestore(
	ctx context.Context, req *connect.Request[v1.CompleteRestoreRequest],
) (*connect.Response[v1.CompleteRestoreResponse], error) {
	pluginID, restore, err := e.getRestoreOf(req.Header())
	if err != nil {
		return nil, err
	}
	if err := restore.CompleteRestore(ctx, pluginID, req.Msg.GetChain()); err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.CompleteRestoreResponse{}), nil
}

func (e *accountEndpoint) OpenSealedBox(
	ctx context.Context, req *connect.Request[v1.OpenSealedBoxRequest],
) (*connect.Response[v1.OpenSealedBoxResponse], error) {
	pluginID, account, err := e.getAccountOf(req.Header())
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetSealed()) > maxBoxBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, errBoxSize)
	}
	plain, err := account.OpenSealedBox(ctx, pluginID, req.Msg.GetContext(), req.Msg.GetSealed())
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.OpenSealedBoxResponse{Plain: plain}), nil
}
