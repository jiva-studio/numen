// Package docfile opens the file of a document that is held open while its
// pages are drawn.
package docfile

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Open opens a document's file to be held open while it is drawn. A reader
// that can open a file shared is asked to.
func Open(ctx context.Context, reader port.VaultReader, path string) (io.ReadSeekCloser, error) {
	if shared, ok := reader.(port.SharedOpener); ok {
		return shared.OpenShared(ctx, path)
	}
	return reader.Open(ctx, path)
}
