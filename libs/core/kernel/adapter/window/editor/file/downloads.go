package file

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// Downloads fills the absent files of the vault and takes their bytes off the device again, by the file
// provider.
type Downloads struct {
	vault Vault
	fetch *file.FetchFile
	drop  *file.DropDownload
}

// NewDownloads is the part of FileService that fills and drops the files the provider keeps.
func NewDownloads(window Vault, fetch *file.FetchFile, drop *file.DropDownload) *Downloads {
	return &Downloads{vault: window, fetch: fetch, drop: drop}
}

// FetchFile fills the absent file at the path.
func (h *Downloads) FetchFile(
	ctx context.Context, r *connect.Request[v1.FetchFileRequest],
) (*connect.Response[v1.FetchFileResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	if err := h.fetch.Execute(ctx, showing, r.Msg.GetPath()); err != nil {
		return nil, newDownloadError(err)
	}
	return connect.NewResponse(&v1.FetchFileResponse{}), nil
}

// DropDownload takes the bytes of the file at the path off the device.
func (h *Downloads) DropDownload(
	ctx context.Context, r *connect.Request[v1.DropDownloadRequest],
) (*connect.Response[v1.DropDownloadResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	if err := h.drop.Execute(ctx, showing, r.Msg.GetPath()); err != nil {
		return nil, newDownloadError(err)
	}
	return connect.NewResponse(&v1.DropDownloadResponse{}), nil
}

func newDownloadError(err error) error {
	var answered *connect.Error
	if errors.As(err, &answered) {
		return answered
	}
	var unavailable *plugindomain.UnavailableError
	if errors.As(err, &unavailable) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return wire.NewInternalError(err)
}
