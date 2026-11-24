package hostendpoint

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const (
	maxLinkWaitSeconds = 20
	maxLinkBlob        = 256 << 10
	linkChannelSize    = 16
)

func (e *accountEndpoint) getLinkOf(header http.Header) (AccountLink, error) {
	if _, err := e.auth.Authorize(header.Get("Authorization"), manifest.PermissionIdentityLink); err != nil {
		return nil, err
	}
	link := e.calls.Link
	if link == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	return link, nil
}

func (e *accountEndpoint) getChainOf(header http.Header) (AccountChain, error) {
	if _, err := e.auth.Authorize(header.Get("Authorization"), manifest.PermissionIdentityLink); err != nil {
		return nil, err
	}
	chain := e.calls.Chain
	if chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoAccount)
	}
	return chain, nil
}

func (e *accountEndpoint) GetLinkWork(
	ctx context.Context, req *connect.Request[v1.GetLinkWorkRequest],
) (*connect.Response[v1.GetLinkWorkResponse], error) {
	link, err := e.getLinkOf(req.Header())
	if err != nil {
		return nil, err
	}
	wait := min(req.Msg.GetWaitSeconds(), maxLinkWaitSeconds)
	work, err := link.GetLinkWork(ctx, req.Msg.GetSinceVersion(), time.Duration(wait)*time.Second)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.GetLinkWorkResponse{
		Version:      work.Version,
		IsActive:     work.IsActive,
		WantsChannel: work.WantsChannel,
		ChannelId:    work.ChannelID,
		Server:       work.Server,
		Outbound:     work.Outbound,
		IsClosed:     work.IsClosed,
	}), nil
}

func (e *accountEndpoint) SetLinkChannel(
	ctx context.Context, req *connect.Request[v1.SetLinkChannelRequest],
) (*connect.Response[v1.SetLinkChannelResponse], error) {
	link, err := e.getLinkOf(req.Header())
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetChannelId()) != linkChannelSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSize)
	}
	if err := link.SetLinkChannel(ctx, req.Msg.GetChannelId()); err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.SetLinkChannelResponse{}), nil
}

func (e *accountEndpoint) PutLinkInbound(
	ctx context.Context, req *connect.Request[v1.PutLinkInboundRequest],
) (*connect.Response[v1.PutLinkInboundResponse], error) {
	link, err := e.getLinkOf(req.Header())
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetBlob()) == 0 || len(req.Msg.GetBlob()) > maxLinkBlob {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSize)
	}
	if err := link.PutLinkInbound(ctx, req.Msg.GetBlob()); err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.PutLinkInboundResponse{}), nil
}

func (e *accountEndpoint) GetChainPublish(
	ctx context.Context, req *connect.Request[v1.GetChainPublishRequest],
) (*connect.Response[v1.GetChainPublishResponse], error) {
	chain, err := e.getChainOf(req.Header())
	if err != nil {
		return nil, err
	}
	pending, err := chain.GetChainPublish(ctx)
	if err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.GetChainPublishResponse{IsPending: pending.IsPending, Chain: pending.Chain}), nil
}

func (e *accountEndpoint) PutChainPublished(
	ctx context.Context, req *connect.Request[v1.PutChainPublishedRequest],
) (*connect.Response[v1.PutChainPublishedResponse], error) {
	chain, err := e.getChainOf(req.Header())
	if err != nil {
		return nil, err
	}
	// A revoke the core ended because the device to revoke added a device meanwhile is the answer to the courier, not a failure of it.
	err = chain.PutChainPublished(ctx, req.Msg.GetAccepted(), req.Msg.GetLatest())
	switch {
	case errors.Is(err, identitydomain.ErrRevokeRaced):
		slog.WarnContext(ctx, "the server did not take the chain and the revoke ended", "accepted", false, "error", err)
	case err != nil:
		slog.WarnContext(ctx, "could not take the answer of the server to the chain", "accepted", req.Msg.GetAccepted(), "error", err)
		return nil, getAccountError(err)
	default:
		slog.InfoContext(ctx, "the server answered to the chain", "accepted", req.Msg.GetAccepted())
	}
	return connect.NewResponse(&v1.PutChainPublishedResponse{}), nil
}

func (e *accountEndpoint) AdoptChain(
	ctx context.Context, req *connect.Request[v1.AdoptChainRequest],
) (*connect.Response[v1.AdoptChainResponse], error) {
	chain, err := e.getChainOf(req.Header())
	if err != nil {
		return nil, err
	}
	if err := chain.AdoptChain(ctx, req.Msg.GetChain()); err != nil {
		return nil, getAccountError(err)
	}
	return connect.NewResponse(&v1.AdoptChainResponse{}), nil
}
