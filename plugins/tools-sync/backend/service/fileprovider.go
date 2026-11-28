package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/fileprovider"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/providerwire"
)

// fileProvider serves the download methods of the backend to the core as the file provider.
type fileProvider struct{ runtime *runtime }

func newFileProvider(r *runtime) fileProvider { return fileProvider{runtime: r} }

func (p fileProvider) FetchFile(ctx context.Context, vaultID, path string) error {
	_, err := handle(ctx, p.runtime, api.MethodFetchFile, func(b *Backend) (api.FetchFileResponse, error) {
		return b.FetchFile(ctx, api.FetchFileRequest{VaultID: vaultID, Path: path})
	})
	return providerwire.NewError(err)
}

func (p fileProvider) DropDownload(ctx context.Context, vaultID, path string) error {
	_, err := handle(ctx, p.runtime, api.MethodDropDownload, func(b *Backend) (api.DropDownloadResponse, error) {
		return b.DropDownload(ctx, api.DropDownloadRequest{VaultID: vaultID, Path: path})
	})
	return providerwire.NewError(err)
}

func (p fileProvider) GetFetchProgress(ctx context.Context, vaultID, path string) (fileprovider.Progress, error) {
	answer, err := handle(ctx, p.runtime, api.MethodGetFetchProgress, func(b *Backend) (api.GetFetchProgressResponse, error) {
		return b.GetFetchProgress(ctx, api.GetFetchProgressRequest{VaultID: vaultID, Path: path})
	})
	return fileprovider.Progress{IsKnown: answer.IsKnown, BytesDone: answer.BytesDone, BytesTotal: answer.BytesTotal}, providerwire.NewError(err)
}
