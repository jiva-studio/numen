package vault_test

import (
	"context"
	"io"
)

func (m *scanVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return m.Open(ctx, path)
}

func (h heldWalk) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return h.Open(ctx, path)
}
