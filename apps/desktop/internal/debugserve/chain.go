//go:build numen_debug

package debugserve

import (
	"context"
	"sync"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// ChainStore keeps the device chain in memory for the life of the process, next to the in-memory key store, so a test
// server never reads or writes the chain file of the machine.
type ChainStore struct {
	mu    sync.Mutex
	chain []byte
}

// NewChainStore returns a ChainStore that holds no chain.
func NewChainStore() *ChainStore { return &ChainStore{} }

// Read returns the chain, or ErrNoChain when none was written.
func (s *ChainStore) Read(_ context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), s.chain...), nil
}

// Write replaces the chain.
func (s *ChainStore) Write(_ context.Context, chain []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chain = append([]byte(nil), chain...)
	return nil
}

func (s *ChainStore) Delete(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chain = nil
	return nil
}
