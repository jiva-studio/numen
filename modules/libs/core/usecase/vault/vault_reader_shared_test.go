package vault_test

import (
	"context"
	"io"
)

func (r *scanVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return r.Open(ctx, path)
}

func (r heldWalk) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return r.Open(ctx, path)
}
