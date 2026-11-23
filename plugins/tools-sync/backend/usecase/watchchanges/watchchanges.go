// Package watchchanges keeps the stream of change hints open and wakes the sync of each vault a hint names.
package watchchanges

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	retryWait    = 3 * time.Second
	silenceBeats = 3
	minHeartbeat = time.Second
)

// Deps are the ports WatchChanges needs.
type Deps struct {
	Hints ports.Hints
	Host  ports.Host
	Links ports.Links
	Waker ports.Waker
	Clock ports.Clock
}

// WatchChanges reads the stream of hints until its context ends.
type WatchChanges struct {
	deps Deps
}

// NewWatchChanges returns the use case over deps.
func NewWatchChanges(deps Deps) *WatchChanges { return &WatchChanges{deps: deps} }

// Execute opens the stream, wakes the vaults it names and opens it again after each end, until ctx ends. It
// writes nothing and raises no health item: a stream that does not work only costs the speed, and the pass at
// the interval reports a server that is away.
func (u *WatchChanges) Execute(ctx context.Context) error {
	var closeErr error
	for {
		closeErr = u.runOnce(ctx)
		if err := ctx.Err(); err != nil {
			return errors.Join(err, closeErr)
		}
		if err := u.deps.Clock.Wait(ctx, retryWait); err != nil {
			return err
		}
	}
}

func (u *WatchChanges) runOnce(ctx context.Context) (closeErr error) {
	watch, err := u.deps.Hints.Watch(ctx)
	if err != nil {
		return nil
	}
	defer func() { closeErr = watch.Close() }()
	u.read(ctx, watch)
	return nil
}

type received struct {
	event ports.HintEvent
	err   error
}

func (u *WatchChanges) read(ctx context.Context, watch ports.Watch) {
	readCtx, stop := context.WithCancel(ctx)
	var group sync.WaitGroup
	defer func() {
		stop()
		group.Wait()
	}()
	events := make(chan received)
	group.Go(func() { pump(readCtx, watch, events) })
	// Only the clock of the plugin can tell a silence of silenceBeats heartbeats from a quiet but healthy stream.
	beat := silenceBeats * max(watch.GetHeartbeat(), minHeartbeat)
	silence := u.startSilence(readCtx, &group, beat)
	for {
		select {
		case <-ctx.Done():
			return
		case <-silence.fired:
			return
		case got := <-events:
			if got.err != nil {
				return
			}
			silence.cancel()
			silence = u.startSilence(readCtx, &group, beat)
			u.wakeFor(ctx, got.event)
		}
	}
}

func pump(ctx context.Context, watch ports.Watch, events chan<- received) {
	for {
		event, err := watch.Receive(ctx)
		select {
		case events <- received{event: event, err: err}:
		case <-ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

type silence struct {
	fired  <-chan struct{}
	cancel context.CancelFunc
}

func (u *WatchChanges) startSilence(ctx context.Context, group *sync.WaitGroup, d time.Duration) silence {
	silenceCtx, cancel := context.WithCancel(ctx)
	fired := make(chan struct{})
	group.Go(func() {
		if u.deps.Clock.Wait(silenceCtx, d) != nil {
			return
		}
		select {
		case fired <- struct{}{}:
		case <-silenceCtx.Done():
		}
	})
	return silence{fired: fired, cancel: cancel}
}

func (u *WatchChanges) wakeFor(ctx context.Context, event ports.HintEvent) {
	if event.IsHeartbeat() {
		return
	}
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return
	}
	for _, vault := range vaults {
		link, err := u.deps.Links.ReadLink(ctx, vault.ID)
		if err != nil {
			continue
		}
		if link.RemoteVaultID == event.RemoteVaultID && link.ServerSeq < event.Seq {
			u.deps.Waker.Wake(vault.ID)
		}
	}
}
