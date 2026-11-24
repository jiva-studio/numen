package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/downloadwire"
)

// WithDownloads gives the backend the methods of the downloads of absent files.
func (b *Backend) WithDownloads(downloads *downloadwire.Downloads) *Backend {
	b.downloads = downloads
	return b
}

// FetchFile downloads the absent file at the path of a vault.
func (b *Backend) FetchFile(ctx context.Context, request api.FetchFileRequest) (api.FetchFileResponse, error) {
	return b.downloads.FetchFile(ctx, request)
}

// DropDownload removes the download of the file at the path of a vault.
func (b *Backend) DropDownload(ctx context.Context, request api.DropDownloadRequest) (api.DropDownloadResponse, error) {
	return b.downloads.DropDownload(ctx, request)
}

// GetDownloads lists the downloads that run.
func (b *Backend) GetDownloads(ctx context.Context, request api.GetDownloadsRequest) (api.GetDownloadsResponse, error) {
	return b.downloads.GetDownloads(ctx, request)
}
