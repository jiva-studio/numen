package hostendpoint

import (
	"context"
	"errors"
	"io/fs"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/vaultaccess"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

const noStoreVault = v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_NO_VAULT

// storeEndpoint answers HostStoreService: each plugin's own folder in the store of a vault.
type storeEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostStoreServiceHandler = (*storeEndpoint)(nil)

// open authenticates a call and opens the caller's folder in the vault it works. The folder is the
// caller's own, so it takes no vault permission. A host with no vault to offer answers NO_VAULT.
func (e *storeEndpoint) open(ctx context.Context, header http.Header) (pluginport.PluginStore, v1.HostStoreErrorCode, error) {
	pluginID, err := e.endpoint.authenticate(header.Get("Authorization"))
	if err != nil {
		return nil, 0, err
	}
	vault, err := e.endpoint.getVault(header)
	if err != nil {
		return nil, 0, err
	}
	if vault == nil {
		return nil, noStoreVault, nil
	}
	store, err := vault.OpenStore(pluginID)
	if err != nil {
		code, err := answerStore(ctx, err)
		return nil, code, err
	}
	return store, 0, nil
}

// answerStore is the code a refusal of the store is answered with. The end of the caller's context
// is cancelled, and anything else is internal.
func answerStore(ctx context.Context, err error) (v1.HostStoreErrorCode, error) {
	if err == nil {
		return v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, vaultaccess.ErrNoVault):
		return noStoreVault, nil
	case errors.Is(err, vaultport.ErrHeldByAnother):
		return v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_HELD_BY_ANOTHER, nil
	case errors.Is(err, vaultport.ErrOutside):
		return v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_OUTSIDE, nil
	case errors.Is(err, fs.ErrNotExist):
		return v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

func (e *storeEndpoint) ReadStoreFile(
	ctx context.Context, req *connect.Request[v1.ReadStoreFileRequest],
) (*connect.Response[v1.ReadStoreFileResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.ReadStoreFileResponse{Error: code}), err
	}
	content, err := store.Read(ctx, req.Msg.GetName())
	code, err = answerStore(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadStoreFileResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadStoreFileResponse{Content: content}), nil
}

func (e *storeEndpoint) AppendStoreFile(
	ctx context.Context, req *connect.Request[v1.AppendStoreFileRequest],
) (*connect.Response[v1.AppendStoreFileResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.AppendStoreFileResponse{Error: code}), err
	}
	code, err = answerStore(ctx, store.Append(ctx, req.Msg.GetName(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.AppendStoreFileResponse{Error: code}), err
}

func (e *storeEndpoint) ListStoreFiles(
	ctx context.Context, req *connect.Request[v1.ListStoreFilesRequest],
) (*connect.Response[v1.ListStoreFilesResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.ListStoreFilesResponse{Error: code}), err
	}
	found, err := store.List(ctx)
	code, err = answerStore(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListStoreFilesResponse{Error: code}), err
	}
	files := make([]*v1.HostStoreFile, 0, len(found))
	for _, one := range found {
		files = append(files, &v1.HostStoreFile{Name: one.Name, Size: one.Size})
	}
	return connect.NewResponse(&v1.ListStoreFilesResponse{Files: files}), nil
}

func (e *storeEndpoint) WriteStoreFile(
	ctx context.Context, req *connect.Request[v1.WriteStoreFileRequest],
) (*connect.Response[v1.WriteStoreFileResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.WriteStoreFileResponse{Error: code}), err
	}
	code, err = answerStore(ctx, store.Write(ctx, req.Msg.GetName(), req.Msg.GetContent()))
	return connect.NewResponse(&v1.WriteStoreFileResponse{Error: code}), err
}

func (e *storeEndpoint) DeleteStoreFile(
	ctx context.Context, req *connect.Request[v1.DeleteStoreFileRequest],
) (*connect.Response[v1.DeleteStoreFileResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.DeleteStoreFileResponse{Error: code}), err
	}
	code, err = answerStore(ctx, store.Delete(ctx, req.Msg.GetName()))
	return connect.NewResponse(&v1.DeleteStoreFileResponse{Error: code}), err
}

func (e *storeEndpoint) RenameStoreFile(
	ctx context.Context, req *connect.Request[v1.RenameStoreFileRequest],
) (*connect.Response[v1.RenameStoreFileResponse], error) {
	store, code, err := e.open(ctx, req.Header())
	if err != nil || store == nil {
		return connect.NewResponse(&v1.RenameStoreFileResponse{Error: code}), err
	}
	code, err = answerStore(ctx, store.Rename(ctx, req.Msg.GetFrom(), req.Msg.GetTo()))
	return connect.NewResponse(&v1.RenameStoreFileResponse{Error: code}), err
}
