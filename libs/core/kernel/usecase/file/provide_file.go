package file

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// FileProvider is the plugin that keeps the bytes of the files a vault lists as absent.
type FileProvider interface {
	// FetchFile fills the absent file at path and returns once it holds its bytes.
	FetchFile(ctx context.Context, v domain.Vault, path string) error
	// DropDownload takes the bytes of the file at path off the device and lists the file as absent again.
	DropDownload(ctx context.Context, v domain.Vault, path string) error
	// GetFetchProgress says how far the download of the file at path has come.
	GetFetchProgress(ctx context.Context, v domain.Vault, path string) (vaultdomain.DownloadProgress, error)
}

// FetchFile brings the bytes of an absent file onto the device by the file provider.
type FetchFile struct{ provider FileProvider }

func NewFetchFile(provider FileProvider) *FetchFile { return &FetchFile{provider: provider} }

// Execute returns once the file at path holds its bytes, or the failure of the provider.
func (u *FetchFile) Execute(ctx context.Context, v domain.Vault, path string) error {
	return u.provider.FetchFile(ctx, v, path)
}

// DropDownload takes the bytes of a downloaded file off the device by the file provider.
type DropDownload struct{ provider FileProvider }

func NewDropDownload(provider FileProvider) *DropDownload {
	return &DropDownload{provider: provider}
}

// Execute lists the file at path as absent again, or returns the failure of the provider.
func (u *DropDownload) Execute(ctx context.Context, v domain.Vault, path string) error {
	return u.provider.DropDownload(ctx, v, path)
}

// GetDownloadProgress reads how far the download of a file has come from the file provider.
type GetDownloadProgress struct{ provider FileProvider }

func NewGetDownloadProgress(provider FileProvider) *GetDownloadProgress {
	return &GetDownloadProgress{provider: provider}
}

// Execute returns the progress of the download of the file at path, or the failure of the provider.
func (u *GetDownloadProgress) Execute(ctx context.Context, v domain.Vault, path string) (vaultdomain.DownloadProgress, error) {
	return u.provider.GetFetchProgress(ctx, v, path)
}
