// Package keeper runs the sign-in against one server at a time and runs it again when the address changes.
package keeper

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

// NewServer returns the server at the address, checking its challenges against now.
type NewServer func(address string, now func() time.Time) session.Server

// Keeper runs the sign-in against the server at an address, and runs it again when that address changes.
type Keeper struct {
	deps      session.Deps
	newServer NewServer

	mu      sync.Mutex
	address string
	cancel  context.CancelFunc
	done    chan struct{}
	last    *session.Keep
}

// New returns a keeper that runs no sign-in until Configure names an address.
func New(deps session.Deps, newServer NewServer) *Keeper {
	return &Keeper{deps: deps, newServer: newServer, mu: sync.Mutex{}, address: "", cancel: nil, done: nil, last: nil}
}

// Configure signs in to the server at the address until ctx ends or the address changes. An empty address
// signs in to none.
func (k *Keeper) Configure(ctx context.Context, address string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if address == k.address && k.cancel != nil {
		return
	}
	k.stopWithLock()
	if address == "" {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	deps := k.deps
	deps.Server = k.newServer(address, deps.Clock.Now)
	k.address, k.cancel, k.done = address, cancel, done
	keep := session.NewKeep(deps)
	if k.last != nil {
		keep.Inherit(k.last)
	}
	k.last = keep
	go func() {
		defer close(done)
		// Execute returns only when ctx ends, with its error.
		err := keep.Execute(ctx)
		if ctx.Err() == nil {
			slog.Warn("the sign-in keeper ended", "error", err)
		}
	}()
}

// Stop ends the sign-in and waits for it to return.
func (k *Keeper) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.stopWithLock()
}

func (k *Keeper) stopWithLock() {
	if k.cancel == nil {
		return
	}
	k.cancel()
	<-k.done
	k.address, k.cancel, k.done = "", nil, nil
}
