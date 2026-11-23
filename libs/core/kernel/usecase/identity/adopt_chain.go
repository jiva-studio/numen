package identity

import (
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// AdoptChain takes a chain the server holds.
type AdoptChain struct{ deps Deps }

// NewAdoptChain returns the use case that adopts a chain.
func NewAdoptChain(d Deps) *AdoptChain { return &AdoptChain{deps: d} }

// Execute takes the chain when it is the chain of this device or a longer one that extends it (identitylib.ErrNotExtension otherwise).
// When the epoch grew the record takes the keys the chain wraps for this device; when the chain revokes this device the record stays as it is
// and the device is revoked. Taking the same chain again changes nothing.
func (u *AdoptChain) Execute(ctx context.Context, chainBytes []byte) error {
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	state, err := u.deps.readState(ctx)
	if err != nil {
		return err
	}
	_, err = u.deps.adoptState(ctx, state, chain)
	return err
}

func (d Deps) adoptState(ctx context.Context, state identitylib.State, chain identitylib.Chain) (identitylib.State, error) {
	next, err := state.Adopt(chain)
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	if next.Chain.Len() == state.Chain.Len() {
		return next, nil
	}
	if err := d.writeAdoptedState(ctx, next); err != nil {
		return identitylib.State{}, err
	}
	return next, nil
}
