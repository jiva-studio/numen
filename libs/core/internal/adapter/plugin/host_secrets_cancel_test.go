package plugin

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
)

// cancelSetup is a host whose secret operations wait for the lock, with what an operation would
// change and what it would send.
type cancelSetup struct {
	host    *Host
	probe   *configured
	store   *keys
	source  *listing
	changed atomic.Bool
}

// change is the secret operation: it renames the secret and replaces its value.
func (c *cancelSetup) change(ctx context.Context) func() ([]string, error) {
	return func() ([]string, error) {
		c.changed.Store(true)
		c.source.rename("Renamed")
		return []string{secretPluginID}, c.store.SetKey(ctx, "secret/"+secretID, "value-replaced")
	}
}

func newCancelSetup(t *testing.T) *cancelSetup {
	t.Helper()
	store := newKeys(map[string]string{"secret/" + secretID: fromKeychain})
	host, probe := newSecretHost(t, store)
	source := newListing("Work")
	host.SetSecretSource(source.source())
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	return &cancelSetup{host: host, probe: probe, store: store, source: source}
}

// holdLock keeps the lock of the secret operations busy with the activation of the plugin, whose
// Configure is held. It returns the function that lets the activation end and the channel that
// receives its result.
func (c *cancelSetup) holdLock(t *testing.T) (release func(), activation <-chan error) {
	t.Helper()
	activate := prepareActivation(t, c.host, true)
	entered, release := c.probe.setHold(t)
	result := make(chan error, 1)
	go func() { result <- activate() }()
	<-entered
	return release, result
}

// watchedContext signals through waitSignal that a callee began to wait for the end of the context, so a
// test can cancel it while the callee waits.
type watchedContext struct {
	context.Context
	once       sync.Once
	waitSignal chan struct{}
}

func newWatchedContext(parent context.Context) *watchedContext {
	return &watchedContext{Context: parent, waitSignal: make(chan struct{})}
}

func (w *watchedContext) Done() <-chan struct{} {
	w.once.Do(func() { close(w.waitSignal) })
	return w.Context.Done()
}

// getOperationNames returns the names of the keys of operations.
func getOperationNames() []string {
	return []string{"ApplySecretChange", "RefreshSecrets", "NotifyPluginConfig"}
}

// operations are the calls that wait for the lock of the secret operations.
func (c *cancelSetup) operations() map[string]func(ctx context.Context) error {
	return map[string]func(ctx context.Context) error{
		"ApplySecretChange": func(ctx context.Context) error {
			return c.host.ApplySecretChange(ctx, c.change(ctx))
		},
		"RefreshSecrets": func(ctx context.Context) error {
			return c.host.RefreshSecrets(ctx)
		},
		"NotifyPluginConfig": func(ctx context.Context) error {
			return c.host.NotifyPluginConfig(ctx, secretPluginID)
		},
	}
}

// assertNothingCommitted checks that the cancelled operation left the ledger and the key store as
// they were and sent nothing besides the Configure of the activation that held the lock.
func (c *cancelSetup) assertNothingCommitted(t *testing.T, calls int) {
	t.Helper()
	if c.changed.Load() {
		t.Error("the operation committed its change")
	}
	if name := c.source.current(); name != "Work" {
		t.Errorf("the ledger holds the name %q, want Work", name)
	}
	if got, err := c.store.GetKey(t.Context(), "secret/"+secretID); err != nil || got != fromKeychain {
		t.Errorf("the key store holds %q (%v), want the value it held", got, err)
	}
	if c.probe.count() != calls {
		t.Errorf("%d Configure calls, want %d", c.probe.count(), calls)
	}
}

// An operation whose context ends while it waits for the lock returns the context error at once and
// commits nothing, and the lock stays usable for the operation after it.
func TestAnOperationCancelledWhileItWaitsForTheLockReturnsAndCommitsNothing(t *testing.T) {
	for _, name := range getOperationNames() {
		t.Run(name, func(t *testing.T) {
			c := newCancelSetup(t)
			release, activation := c.holdLock(t)
			calls := c.probe.count()

			cancellable, cancel := context.WithCancel(t.Context())
			ctx := newWatchedContext(cancellable)
			result := make(chan error, 1)
			go func() { result <- c.operations()[name](ctx) }()
			select {
			case <-ctx.waitSignal:
			case err := <-result:
				t.Fatalf("the operation returned %v without waiting for its context", err)
			case <-time.After(configureCallTimeout):
				t.Fatal("the operation does not wait for the end of its context")
			}
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("the operation returned %v, want the context error", err)
				}
			case <-time.After(configureCallTimeout):
				t.Fatal("the operation still waits for the lock after its context ended")
			}

			release()
			if err := <-activation; err != nil {
				t.Fatal(err)
			}
			c.assertNothingCommitted(t, calls+1)
			if err := c.host.ApplySecretChange(t.Context(), c.source.renameBy("Later")); err != nil {
				t.Fatalf("the lock cannot be taken after the cancelled operation: %v", err)
			}
			if got := deliveredName(t, c.probe); got != "Later" {
				t.Errorf("the plugin holds the list %q, want the one of the operation after", got)
			}
		})
	}
}

// An operation whose context ended before the call returns the context error and commits nothing,
// whether the lock is free or busy.
func TestAnOperationCancelledBeforeTheCallReturnsAndCommitsNothing(t *testing.T) {
	for _, name := range getOperationNames() {
		for _, isLockBusy := range []bool{false, true} {
			label := name + " with a free lock"
			if isLockBusy {
				label = name + " with a busy lock"
			}
			t.Run(label, func(t *testing.T) {
				c := newCancelSetup(t)
				calls := c.probe.count()
				release := func() {}
				var activation <-chan error
				if isLockBusy {
					release, activation = c.holdLock(t)
					calls++
				}
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				result := make(chan error, 1)
				go func() { result <- c.operations()[name](ctx) }()
				select {
				case err := <-result:
					if !errors.Is(err, context.Canceled) {
						t.Errorf("the operation returned %v, want the context error", err)
					}
				case <-time.After(configureCallTimeout):
					t.Fatal("the operation still waits for the lock after its context ended")
				}

				release()
				if isLockBusy {
					if err := <-activation; err != nil {
						t.Fatal(err)
					}
				}
				c.assertNothingCommitted(t, calls)
			})
		}
	}
}

// An activation whose context ends while it waits for the lock returns the context error, sends
// nothing and no longer awaits its Configure, so the next refresh gives the plugin its list.
func TestAnActivationCancelledWhileItWaitsForTheLockSendsNothingAndTheNextRefreshConfiguresThePlugin(t *testing.T) {
	c := newCancelSetup(t)
	calls := c.probe.count()
	gate, entered := make(chan struct{}), make(chan struct{})
	operation := make(chan error, 1)
	go func() {
		operation <- c.host.ApplySecretChange(t.Context(), func() ([]string, error) {
			close(entered)
			<-gate
			return nil, nil
		})
	}()
	<-entered
	activate := prepareActivationWith(t, c.host, true)

	ctx, cancel := context.WithCancel(t.Context())
	activation := make(chan error, 1)
	go func() { activation <- activate(ctx) }()
	plugintest.WaitUntil(t, "the activation waits for its Configure", func() bool {
		return c.host.isAwaitingFirstConfigure(secretPluginID)
	})
	cancel()
	select {
	case err := <-activation:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the activation returned %v, want the context error", err)
		}
	case <-time.After(configureCallTimeout):
		t.Fatal("the activation still waits for the lock after its context ended")
	}
	if c.host.isAwaitingFirstConfigure(secretPluginID) {
		t.Error("the plugin still awaits a Configure that will not be sent")
	}

	close(gate)
	if err := <-operation; err != nil {
		t.Fatal(err)
	}
	if c.probe.count() != calls {
		t.Errorf("%d Configure calls, want %d", c.probe.count(), calls)
	}
	if err := c.host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if c.probe.count() != calls+1 {
		t.Errorf("%d Configure calls after the refresh, want %d", c.probe.count(), calls+1)
	}
}
