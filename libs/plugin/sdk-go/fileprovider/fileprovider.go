// Package fileprovider serves a file provider over FileProviderService: the plugin that keeps the bytes of
// the files a vault lists as absent. The core calls it to fill an absent file and to take the bytes of a
// download off the device, and names no plugin to do so. A plugin that serves it declares the capability
// files.provider.
package fileprovider

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Handler is what a plugin implements to provide files. A failure is answered to the client with its code
// and message: a refusal a person reads is a failed precondition.
type Handler interface {
	// FetchFile downloads the absent file at path of the vault and fills it, and returns once the file
	// holds its bytes.
	FetchFile(ctx context.Context, vaultID, path string) error
	// DropDownload removes the download of the file at path of the vault. A file with changes the plugin
	// has not kept is a failed precondition, and its bytes stay.
	DropDownload(ctx context.Context, vaultID, path string) error
	// GetFetchProgress says how far the download at path of the vault has come. It is not known when no
	// download of that path in that vault runs.
	GetFetchProgress(ctx context.Context, vaultID, path string) (Progress, error)
}

// Progress is how far a download has come. Without a running download IsKnown is false and the sizes are
// zero.
type Progress struct {
	IsKnown    bool
	BytesDone  int64
	BytesTotal int64
}

type adapter struct {
	numenv1connect.UnimplementedFileProviderServiceHandler
	h Handler
}

func (a *adapter) FetchFile(
	ctx context.Context, req *connect.Request[v1.FileProviderServiceFetchFileRequest],
) (*connect.Response[v1.FileProviderServiceFetchFileResponse], error) {
	if err := a.h.FetchFile(ctx, req.Msg.GetVaultId(), req.Msg.GetPath()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.FileProviderServiceFetchFileResponse{}), nil
}

func (a *adapter) DropDownload(
	ctx context.Context, req *connect.Request[v1.FileProviderServiceDropDownloadRequest],
) (*connect.Response[v1.FileProviderServiceDropDownloadResponse], error) {
	if err := a.h.DropDownload(ctx, req.Msg.GetVaultId(), req.Msg.GetPath()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.FileProviderServiceDropDownloadResponse{}), nil
}

func (a *adapter) GetFetchProgress(
	ctx context.Context, req *connect.Request[v1.GetFetchProgressRequest],
) (*connect.Response[v1.GetFetchProgressResponse], error) {
	progress, err := a.h.GetFetchProgress(ctx, req.Msg.GetVaultId(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetFetchProgressResponse{
		IsKnown: progress.IsKnown, BytesDone: progress.BytesDone, BytesTotal: progress.BytesTotal,
	}), nil
}

// NewHandler returns the path and the handler of the file provider service over h.
func NewHandler(h Handler) (string, http.Handler) {
	return numenv1connect.NewFileProviderServiceHandler(&adapter{h: h})
}

// Mount registers the file provider service of h on s.
func Mount(s *server.Server, h Handler) {
	s.RegisterAuthenticated(NewHandler(h))
}
