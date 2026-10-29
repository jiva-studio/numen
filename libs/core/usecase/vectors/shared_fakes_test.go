package vectors

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// fakeClaim is the hold on a vault's index, in memory.
type fakeClaim struct {
	mu        sync.Mutex
	isIndexed bool
}

func (c *fakeClaim) ClaimIndex(context.Context) (func() error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isIndexed {
		return nil, vaultport.ErrClaimed
	}
	c.isIndexed = true
	return c.release, nil
}

func (c *fakeClaim) release() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isIndexed = false
	return nil
}

const bookPath = "library/book.epub"
