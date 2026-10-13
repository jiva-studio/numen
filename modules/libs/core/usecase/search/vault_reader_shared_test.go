package search

import (
	"context"
	"io"
)

func (c *concurrentVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return c.Open(ctx, path)
}

func (f *fakeVault) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return f.Open(ctx, path)
}
