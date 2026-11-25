package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

// fileProvider serves the download methods of the backend to the core as the file provider.
type fileProvider struct{ runtime *runtime }

func newFileProvider(r *runtime) fileProvider { return fileProvider{runtime: r} }

func (p fileProvider) FetchFile(ctx context.Context, vaultID, path string) error {
	_, err := handle(ctx, p.runtime, api.MethodFetchFile, func(b *Backend) (api.FetchFileResponse, error) {
		return b.FetchFile(ctx, api.FetchFileRequest{VaultID: vaultID, Path: path})
	})
	return newProviderError(err)
}

func (p fileProvider) DropDownload(ctx context.Context, vaultID, path string) error {
	_, err := handle(ctx, p.runtime, api.MethodDropDownload, func(b *Backend) (api.DropDownloadResponse, error) {
		return b.DropDownload(ctx, api.DropDownloadRequest{VaultID: vaultID, Path: path})
	})
	return newProviderError(err)
}

func newProviderError(err error) error {
	failure, ok := errors.AsType[*server.APIError](err)
	if !ok {
		return err
	}
	return rpcerror.New(getRPCCode(failure.Code), errors.New(failure.Message))
}

func getRPCCode(code server.ErrorCode) rpcerror.Code {
	switch code {
	case server.CodeFailedPrecondition:
		return rpcerror.FailedPrecondition
	case server.CodeUnavailable:
		return rpcerror.Unavailable
	}
	return rpcerror.Internal
}
