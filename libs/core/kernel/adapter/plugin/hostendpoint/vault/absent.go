package vault

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

var errNoAbsentFiles = errors.New("this host lists no absent files")

// AbsentFiles lists the files a vault shows and does not hold, on behalf of the plugin named, which fills
// them and drops their bytes. A host whose files are all held has none.
type AbsentFiles interface {
	PutAbsentFiles(ctx context.Context, pluginID string, files []vaultdomain.AbsentFile) error
	ListAbsentFiles(ctx context.Context) ([]vaultdomain.AbsentFile, error)
	FillFile(ctx context.Context, pluginID, path, storeName string) error
	EvictFile(ctx context.Context, pluginID, path, expectedHash string) error
}

func serveAbsent[Q, R any](
	ctx context.Context, c *calls, req *connect.Request[Q], permission string,
	withCode func(code v1.ErrorCode) *R, work func(pluginID string, absent AbsentFiles) (*R, error),
) (*connect.Response[R], error) {
	pluginID, access, err := c.beginAs(req.Header(), permission)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return connect.NewResponse(withCode(noVault)), nil
	}
	if access.Absent == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoAbsentFiles)
	}
	out, err := work(pluginID, access.Absent)
	if err == nil {
		return connect.NewResponse(out), nil
	}
	code, failure := answer(ctx, err)
	if failure != nil {
		return nil, failure
	}
	return connect.NewResponse(withCode(code)), nil
}

func (e *absentCalls) PutAbsentFiles(
	ctx context.Context, req *connect.Request[v1.PutAbsentFilesRequest],
) (*connect.Response[v1.PutAbsentFilesResponse], error) {
	return serveAbsent(ctx, e.calls, req, manifest.PermissionVaultWrite,
		func(code v1.ErrorCode) *v1.PutAbsentFilesResponse { return &v1.PutAbsentFilesResponse{Error: code} },
		func(pluginID string, absent AbsentFiles) (*v1.PutAbsentFilesResponse, error) {
			files := make([]vaultdomain.AbsentFile, len(req.Msg.GetFiles()))
			for i, file := range req.Msg.GetFiles() {
				files[i] = vaultdomain.AbsentFile{
					Path: file.GetPath(), Size: file.GetSize(), Mtime: time.UnixMilli(file.GetModifiedUnixMs()), Hash: file.GetHash(),
				}
			}
			return &v1.PutAbsentFilesResponse{}, absent.PutAbsentFiles(ctx, pluginID, files)
		})
}

func (e *absentCalls) ListAbsentFiles(
	ctx context.Context, req *connect.Request[v1.ListAbsentFilesRequest],
) (*connect.Response[v1.ListAbsentFilesResponse], error) {
	return serveAbsent(ctx, e.calls, req, manifest.PermissionVaultRead,
		func(code v1.ErrorCode) *v1.ListAbsentFilesResponse { return &v1.ListAbsentFilesResponse{Error: code} },
		func(_ string, absent AbsentFiles) (*v1.ListAbsentFilesResponse, error) {
			listed, err := absent.ListAbsentFiles(ctx)
			if err != nil {
				return nil, err
			}
			files := make([]*v1.HostAbsentFile, len(listed))
			for i, file := range listed {
				files[i] = &v1.HostAbsentFile{Path: file.Path, Size: file.Size, ModifiedUnixMs: file.Mtime.UnixMilli(), Hash: file.Hash}
			}
			return &v1.ListAbsentFilesResponse{Files: files}, nil
		})
}

func (e *absentCalls) FillFile(
	ctx context.Context, req *connect.Request[v1.FillFileRequest],
) (*connect.Response[v1.FillFileResponse], error) {
	return serveAbsent(ctx, e.calls, req, manifest.PermissionVaultWrite,
		func(code v1.ErrorCode) *v1.FillFileResponse { return &v1.FillFileResponse{Error: code} },
		func(pluginID string, absent AbsentFiles) (*v1.FillFileResponse, error) {
			return &v1.FillFileResponse{}, absent.FillFile(ctx, pluginID, req.Msg.GetPath(), req.Msg.GetStoreName())
		})
}

func (e *absentCalls) EvictFile(
	ctx context.Context, req *connect.Request[v1.EvictFileRequest],
) (*connect.Response[v1.EvictFileResponse], error) {
	return serveAbsent(ctx, e.calls, req, manifest.PermissionVaultWrite,
		func(code v1.ErrorCode) *v1.EvictFileResponse { return &v1.EvictFileResponse{Error: code} },
		func(pluginID string, absent AbsentFiles) (*v1.EvictFileResponse, error) {
			return &v1.EvictFileResponse{}, absent.EvictFile(ctx, pluginID, req.Msg.GetPath(), req.Msg.GetExpectedHash())
		})
}
