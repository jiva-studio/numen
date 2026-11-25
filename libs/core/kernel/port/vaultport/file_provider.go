package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// FileProvider is the plugin that keeps the bytes of the files a vault lists as absent.
type FileProvider interface {
	// FetchFile fills the absent file at path and returns once it holds its bytes.
	FetchFile(ctx context.Context, v domain.Vault, path string) error
	// DropDownload takes the bytes of the file at path off the device and lists the file as absent again.
	DropDownload(ctx context.Context, v domain.Vault, path string) error
}
