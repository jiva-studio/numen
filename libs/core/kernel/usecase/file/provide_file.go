package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// FetchFile brings the bytes of an absent file onto the device by the file provider.
type FetchFile struct{ provider vaultport.FileProvider }

func NewFetchFile(provider vaultport.FileProvider) *FetchFile { return &FetchFile{provider: provider} }

// Execute returns once the file at path holds its bytes, or the failure of the provider.
func (u *FetchFile) Execute(ctx context.Context, v domain.Vault, path string) error {
	return u.provider.FetchFile(ctx, v, path)
}

// DropDownload takes the bytes of a downloaded file off the device by the file provider.
type DropDownload struct{ provider vaultport.FileProvider }

func NewDropDownload(provider vaultport.FileProvider) *DropDownload {
	return &DropDownload{provider: provider}
}

// Execute lists the file at path as absent again, or returns the failure of the provider.
func (u *DropDownload) Execute(ctx context.Context, v domain.Vault, path string) error {
	return u.provider.DropDownload(ctx, v, path)
}
