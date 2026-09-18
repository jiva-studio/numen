package vault

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Walks is the vaults being walked, a turn each.
//
// It belongs to whatever the walks are written into: two indexes are two sets
// of turns, and a set goes when the thing holding it goes.
type Walks struct {
	mu    sync.Mutex
	turns map[domain.VaultID]chan struct{}
}

// turn is this vault's turn, made where the set does not hold one yet.
func (w *Walks) turn(vaultID domain.VaultID) chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.turns == nil {
		w.turns = map[domain.VaultID]chan struct{}{}
	}
	held, found := w.turns[vaultID]
	if !found {
		held = make(chan struct{}, 1)
		w.turns[vaultID] = held
	}
	return held
}

// startOne takes the vault's turn and answers with the release of it. Walks of
// different vaults do not wait on each other, and neither do walks that share
// no turns.
func (w *Walks) startOne(ctx context.Context, vaultID domain.VaultID) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if w == nil {
		return func() {}, nil
	}
	turn := w.turn(vaultID)
	select {
	case turn <- struct{}{}:
		return func() { <-turn }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
