package vault

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (e *cacheCalls) ReadCache(
	ctx context.Context, req *connect.Request[v1.ReadCacheRequest],
) (*connect.Response[v1.ReadCacheResponse], error) {
	pluginID, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ReadCacheResponse{Error: noVault}), nil
	}
	content, found, err := vault.Cache.ReadCache(ctx, pluginID, req.Msg.GetName())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadCacheResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadCacheResponse{IsFound: found, Content: content}), nil
}

func (e *cacheCalls) WriteCache(
	ctx context.Context, req *connect.Request[v1.WriteCacheRequest],
) (*connect.Response[v1.WriteCacheResponse], error) {
	pluginID, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteCacheResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Cache.WriteCache(ctx, pluginID, req.Msg.GetName(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.WriteCacheResponse{Error: code}), err
}
