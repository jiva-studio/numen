// Package lifecycle keeps the sync of a phone to the time the application is on the screen.
package lifecycle

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Grace is how long Suspend waits for the last pass.
const Grace = 20 * time.Second

// Pass is one pass over one vault.
type Pass interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// Loop is the interval loop over the connected vaults. It runs until its context ends.
type Loop interface {
	Execute(ctx context.Context) error
}

// Stream is the stream of hints of the server. Watch returns when its context ends, and calls changed with
// the vault each hint names.
type Stream interface {
	Watch(ctx context.Context, changed func(domain.VaultID)) error
}

// Deps are what the use case needs.
type Deps struct {
	Pass   Pass
	Loop   Loop
	Stream Stream
	Host   ports.Host
	State  ports.Links
	Clock  ports.Clock
}

// Lifecycle starts and stops the loops of the sync as the application comes to the screen and leaves it.
type Lifecycle struct {
	deps Deps

	mu        sync.Mutex
	running   *running
	isFlushed bool
}

// running is the loop and the stream while the application is on the screen.
type running struct {
	stop context.CancelFunc
	done sync.WaitGroup
}

// NewLifecycle returns the use case over the dependencies.
func NewLifecycle(deps Deps) *Lifecycle { return &Lifecycle{deps: deps} }

// Resume starts the loop, which passes over every connected vault at once, and the stream. A second call
// while they run does nothing.
func (u *Lifecycle) Resume(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running != nil {
		return nil
	}
	runCtx, stop := context.WithCancel(context.WithoutCancel(ctx))
	started := &running{stop: stop}
	started.done.Add(2)
	go func() {
		defer started.done.Done()
		_ = u.deps.Loop.Execute(runCtx)
	}()
	go func() {
		defer started.done.Done()
		_ = u.deps.Stream.Watch(runCtx, func(id domain.VaultID) { _ = u.deps.Pass.Execute(runCtx, id) })
	}()
	u.running, u.isFlushed = started, false
	return nil
}

// Suspend stops the loop and the stream, and then runs one pass over every connected vault within Grace.
// A second call while they are stopped does nothing. It returns when nothing of the sync runs.
func (u *Lifecycle) Suspend(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running != nil {
		u.running.stop()
		u.running.done.Wait()
		u.running = nil
	} else if u.isFlushed {
		return nil
	}
	u.isFlushed = true
	return u.flush(context.WithoutCancel(ctx))
}

// flush passes over every connected vault and gives up when Grace has gone.
func (u *Lifecycle) flush(ctx context.Context) error {
	ids, err := u.listConnected(ctx)
	if err != nil {
		return err
	}
	flushCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	failures := make(chan error, len(ids))
	var passes sync.WaitGroup
	for _, id := range ids {
		passes.Go(func() { failures <- u.deps.Pass.Execute(flushCtx, id) })
	}
	ended := make(chan struct{})
	go func() {
		passes.Wait()
		close(ended)
	}()
	graceCtx, endGrace := context.WithCancel(ctx)
	defer endGrace()
	graced := make(chan struct{})
	go func() {
		defer close(graced)
		_ = u.deps.Clock.Wait(graceCtx, Grace)
	}()
	select {
	case <-ended:
	case <-graced:
		cancel()
		<-ended
	}
	close(failures)
	return joinFailures(failures)
}

func (u *Lifecycle) listConnected(ctx context.Context) ([]domain.VaultID, error) {
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	var ids []domain.VaultID
	for _, v := range vaults {
		if _, err := u.deps.State.ReadLink(ctx, v.ID); errors.Is(err, vaultlink.ErrNotConnected) {
			continue
		}
		ids = append(ids, v.ID)
	}
	return ids, nil
}

// joinFailures returns the failures that are not a server away or a pass cut short: the loop reports those.
func joinFailures(failures <-chan error) error {
	var joined []error
	for err := range failures {
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, syncprotocol.ErrUnreachable) ||
			errors.Is(err, sessiondomain.ErrUnreachable) {
			continue
		}
		joined = append(joined, err)
	}
	return errors.Join(joined...)
}
