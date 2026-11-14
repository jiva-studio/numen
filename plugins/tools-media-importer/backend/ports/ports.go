// Package ports declares the interfaces the use cases of the plugin share.
package ports

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

// Fetcher retrieves the content published at an address.
type Fetcher interface {
	CanCopy(address string) bool
	GetDownloadModel(address string) fetch.Model
	GetMetadata(ctx context.Context, address string) (fetch.Metadata, error)
	GetText(ctx context.Context, address string, want fetch.PreferredCaptions) (fetch.Text, error)
	Download(ctx context.Context, address string, media fetch.Media, into io.Writer) (fetch.Copy, error)
}

// Entropy fills the random bits of the identifiers a use case mints.
type Entropy interface {
	Fill(b []byte)
}
