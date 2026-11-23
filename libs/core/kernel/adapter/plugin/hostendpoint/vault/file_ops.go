package vault

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func readBaseline(stat *v1.FileStat) *domain.FileStat {
	if stat == nil {
		return nil
	}
	baseline := readFileStat(stat)
	return &baseline
}

func (e *filesCalls) MoveFile(
	ctx context.Context, req *connect.Request[v1.HostFileServiceMoveFileRequest],
) (*connect.Response[v1.HostFileServiceMoveFileResponse], error) {
	return serve(ctx, e.calls, req, manifest.PermissionVaultWrite,
		func(code v1.ErrorCode) *v1.HostFileServiceMoveFileResponse {
			return &v1.HostFileServiceMoveFileResponse{Error: code}
		},
		func(vault *Access) (*v1.HostFileServiceMoveFileResponse, error) {
			count, err := vault.Ops.MoveFile(ctx, req.Msg.GetFrom(), req.Msg.GetTo(), readBaseline(req.Msg.GetBaseline()))
			if err != nil {
				return nil, err
			}
			return &v1.HostFileServiceMoveFileResponse{MovedCount: int32(count)}, nil
		})
}

func (e *filesCalls) RemoveFile(
	ctx context.Context, req *connect.Request[v1.HostFileServiceRemoveFileRequest],
) (*connect.Response[v1.HostFileServiceRemoveFileResponse], error) {
	return serve(ctx, e.calls, req, manifest.PermissionVaultWrite,
		func(code v1.ErrorCode) *v1.HostFileServiceRemoveFileResponse {
			return &v1.HostFileServiceRemoveFileResponse{Error: code}
		},
		func(vault *Access) (*v1.HostFileServiceRemoveFileResponse, error) {
			count, err := vault.Ops.RemoveFile(ctx, req.Msg.GetPath(), readBaseline(req.Msg.GetBaseline()))
			if err != nil {
				return nil, err
			}
			return &v1.HostFileServiceRemoveFileResponse{RemovedCount: int32(count)}, nil
		})
}

func (e *filesCalls) ReadFileRange(
	ctx context.Context, req *connect.Request[v1.ReadFileRangeRequest],
) (*connect.Response[v1.ReadFileRangeResponse], error) {
	return serve(ctx, e.calls, req, manifest.PermissionVaultRead,
		func(code v1.ErrorCode) *v1.ReadFileRangeResponse { return &v1.ReadFileRangeResponse{Error: code} },
		func(vault *Access) (*v1.ReadFileRangeResponse, error) {
			content, stat, err := vault.Ranges.ReadFileRange(ctx, req.Msg.GetPath(), req.Msg.GetStart(), req.Msg.GetLength())
			if err != nil {
				return nil, err
			}
			return &v1.ReadFileRangeResponse{Content: content, Stat: writeFileStat(stat)}, nil
		})
}

func (e *filesCalls) PlaceFile(
	ctx context.Context, req *connect.Request[v1.PlaceFileRequest],
) (*connect.Response[v1.PlaceFileResponse], error) {
	pluginID, vault, err := e.beginAs(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.PlaceFileResponse{Error: noVault}), nil
	}
	stat, err := vault.Ops.PlaceFile(ctx, pluginID, req.Msg.GetName(), req.Msg.GetPath(), req.Msg.GetExpectedHash(), readBaseline(req.Msg.GetBaseline()))
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.PlaceFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.PlaceFileResponse{Stat: writeFileStat(stat)}), nil
}
