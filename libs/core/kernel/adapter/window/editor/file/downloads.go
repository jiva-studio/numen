package file

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Downloads fills the absent files of the vault and takes their bytes off the device again, by the file
// provider.
type Downloads struct {
	vault    Vault
	fetch    *file.FetchFile
	drop     *file.DropDownload
	progress *file.GetDownloadProgress
}

// NewDownloads is the part of FileService that fills and drops the files the provider keeps.
func NewDownloads(
	window Vault, fetch *file.FetchFile, drop *file.DropDownload, progress *file.GetDownloadProgress,
) *Downloads {
	return &Downloads{vault: window, fetch: fetch, drop: drop, progress: progress}
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

// GetDownloadProgress says how far the download of the file at the path has come. It is not known when no
// plugin provides files or the provider runs no download of the file.
func (h *Downloads) GetDownloadProgress(
	ctx context.Context, r *connect.Request[v1.GetDownloadProgressRequest],
) (*connect.Response[v1.GetDownloadProgressResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	progress, err := h.progress.Execute(ctx, showing, r.Msg.GetPath())
	if errors.Is(err, domain.ErrUnavailable) {
		return connect.NewResponse(&v1.GetDownloadProgressResponse{}), nil
	}
	if err != nil {
		return nil, newDownloadError(err)
	}
	return connect.NewResponse(&v1.GetDownloadProgressResponse{
		IsKnown: progress.IsKnown, BytesDone: progress.BytesDone, BytesTotal: progress.BytesTotal,
	}), nil
}

// interruptedError is what the person reads when a download stops: a sentence, which an error string may not be.
type interruptedError struct{}

func (interruptedError) Error() string { return "The download was interrupted. Try again." }

var errDownloadInterrupted error = interruptedError{}

func newDownloadError(err error) error {
	if answered, ok := errors.AsType[*connect.Error](err); ok {
		if isInterruption(answered.Code()) {
			return connect.NewError(connect.CodeUnavailable, errDownloadInterrupted)
		}
		return answered
	}
	if errors.Is(err, domain.ErrUnavailable) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeUnavailable, errDownloadInterrupted)
	}
	return wire.NewInternalError(err)
}

func isInterruption(code connect.Code) bool {
	return code == connect.CodeUnknown || code == connect.CodeCanceled || code == connect.CodeDeadlineExceeded
}
