package search

import (
	"context"
	"io"
)

func (r *concurrentVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return r.Open(ctx, path)
}

func (r *fakeVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return r.Open(ctx, path)
}
