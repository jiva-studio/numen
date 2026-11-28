package identity_test

import (
	"context"
	"sync"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// chainFile is the chain file of one device kept in memory.
type chainFile struct {
	mu    sync.Mutex
	chain []byte
}

func (f *chainFile) Read(context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), f.chain...), nil
}

func (f *chainFile) Write(_ context.Context, chain []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chain = append([]byte(nil), chain...)
	return nil
}

func (f *chainFile) Delete(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chain = nil
	return nil
}
