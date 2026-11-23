package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

// keeper runs the sign-in against the server of the setting server, and runs it again when that address
// changes.
type keeper struct {
	deps session.Deps

	mu      sync.Mutex
	address string
	cancel  context.CancelFunc
	done    chan struct{}
}

func (k *keeper) configure(ctx context.Context, values map[string]string) {
	address := values[settingServer]
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
	deps.Server = newServer(address, deps.Clock.Now)
	k.address, k.cancel, k.done = address, cancel, done
	go func() {
		defer close(done)
		// Execute returns only the error of its context, which ends with the keeper.
		if err := session.NewKeep(deps).Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("the sign-in keeper ended", "error", err)
		}
	}()
}

func (k *keeper) stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.stopWithLock()
}

func (k *keeper) stopWithLock() {
	if k.cancel == nil {
		return
	}
	k.cancel()
	<-k.done
	k.address, k.cancel, k.done = "", nil, nil
}
