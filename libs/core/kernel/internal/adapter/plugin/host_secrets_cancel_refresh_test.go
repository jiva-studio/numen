package plugin

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
)

func getOperationNames() []string {
	return []string{"Refresh", "NotifyPluginConfig"}
}

func (c *cancelSetup) operations() map[string]func(ctx context.Context) error {
	return map[string]func(ctx context.Context) error{
		"Refresh": func(ctx context.Context) error {
			return c.host.Config.Refresh(ctx)
		},
		"NotifyPluginConfig": func(ctx context.Context) error {
			return c.host.Config.NotifyPluginConfig(ctx, secretPluginID)
		},
	}
}

func (c *cancelSetup) assertNothingSent(t *testing.T, calls int) {
	t.Helper()
	if c.probe.count() != calls {
		t.Errorf("%d Configure calls, want %d", c.probe.count(), calls)
	}
}

// A call whose context ends while it waits for the lock returns the context error at once, sends
// nothing, and the lock stays usable for the call after it.
func TestACallCancelledWhileItWaitsForTheLockReturnsAndSendsNothing(t *testing.T) {
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
				t.Fatalf("the call returned %v without waiting for its context", err)
			case <-time.After(delivery.ConfigureTimeout):
				t.Fatal("the call does not wait for the end of its context")
			}
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("the call returned %v, want the context error", err)
				}
			case <-time.After(delivery.ConfigureTimeout):
				t.Fatal("the call still waits for the lock after its context ended")
			}

			release()
			if err := <-activation; err != nil {
				t.Fatal(err)
			}
			c.assertNothingSent(t, calls+1)
			if err := applyChange(c.host, t.Context(), c.source.renameBy("Later")); err != nil {
				t.Fatalf("the lock cannot be taken after the canceled call: %v", err)
			}
			if got := deliveredName(t, c.probe); got != "Later" {
				t.Errorf("the plugin holds the list %q, want the one of the call after", got)
			}
		})
	}
}

// A call whose context ended before it returns the context error and sends nothing, whether the
// lock is free or busy.
func TestACallCancelledBeforeItStartsReturnsAndSendsNothing(t *testing.T) {
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
						t.Errorf("the call returned %v, want the context error", err)
					}
				case <-time.After(delivery.ConfigureTimeout):
					t.Fatal("the call still waits for the lock after its context ended")
				}

				release()
				if isLockBusy {
					if err := <-activation; err != nil {
						t.Fatal(err)
					}
				}
				c.assertNothingSent(t, calls)
			})
		}
	}
}

// An activation whose context ends while it waits for the lock returns the context error, sends
// nothing and no longer awaits its Configure, so the next refresh gives the plugin its list.
func TestAnActivationCancelledWhileItWaitsForTheLockSendsNothingAndTheNextRefreshConfiguresThePlugin(t *testing.T) {
	c := newCancelSetup(t)
	calls := c.probe.count()
	if err := c.host.Config.Delivery().Lock(t.Context()); err != nil {
		t.Fatal(err)
	}
	isLocked := true
	unlock := func() {
		if isLocked {
			isLocked = false
			c.host.Config.Delivery().Unlock()
		}
	}
	defer unlock()
	activate := prepareActivationWith(t, c.host, true)

	ctx, cancel := context.WithCancel(t.Context())
	activation := make(chan error, 1)
	go func() { activation <- activate(ctx) }()
	wiretest.WaitUntil(t, "the activation waits for its Configure", func() bool {
		return c.host.Config.Delivery().IsAwaitingFirstConfigure(secretPluginID)
	})
	cancel()
	select {
	case err := <-activation:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the activation returned %v, want the context error", err)
		}
	case <-time.After(delivery.ConfigureTimeout):
		t.Fatal("the activation still waits for the lock after its context ended")
	}
	if c.host.Config.Delivery().IsAwaitingFirstConfigure(secretPluginID) {
		t.Error("the plugin still awaits a Configure that will not be sent")
	}

	unlock()
	c.assertNothingSent(t, calls)
	if err := c.host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	c.assertNothingSent(t, calls+1)
}

// An operation commits its change and then its context ends before the plugin is configured. The
// next refresh gives the plugin the list the change made.
func TestAnOperationCancelledAfterItsCommitIsRepairedByTheNextRefresh(t *testing.T) {
	c := newCancelSetup(t)
	calls := c.probe.count()
	c.source.rename("Renamed")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := c.host.Config.Refresh(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("the refresh returned %v, want the context error", err)
	}
	c.assertNothingSent(t, calls)
	if err := c.host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	c.assertNothingSent(t, calls+1)
	if got := deliveredName(t, c.probe); got != "Renamed" {
		t.Errorf("the plugin holds the list %q, want the one the change made", got)
	}
}

// A write replaces a value and leaves the list as it is, so the plugin is due only because the
// operation named it. When the context ends after the commit the plugin still gets the new value at
// the next refresh.
func TestAWriteCancelledAfterItsCommitIsRepairedByTheNextRefresh(t *testing.T) {
	c := newCancelSetup(t)
	calls := c.probe.count()
	if err := c.store.SetKey(t.Context(), "secret/"+secretID, "value-replaced"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := c.host.Config.Refresh(ctx, secretPluginID); !errors.Is(err, context.Canceled) {
		t.Fatalf("the refresh returned %v, want the context error", err)
	}
	c.assertNothingSent(t, calls)
	if err := c.host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	c.assertNothingSent(t, calls+1)
	if got := listed(t, c.probe, declared); len(got) != 1 || got[0].GetValue() != "value-replaced" {
		t.Errorf("the plugin holds %v, want the replaced value", got)
	}
}

// A context that ends while the plugin is being configured says nothing about the plugin. The list
// is not recorded as held and the next refresh sends it again.
func TestAConfigureEndedByTheContextOfTheCallerIsSentAgainByTheNextRefresh(t *testing.T) {
	c := newCancelSetup(t)
	calls := c.probe.count()
	entered, release := c.probe.setHold(t)
	c.source.rename("Renamed")
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- c.host.Config.Refresh(ctx) }()
	<-entered
	cancel()
	if err := <-result; err == nil {
		t.Fatal("the refresh succeeded though its context ended during the send")
	}
	release()
	c.assertNothingSent(t, calls)

	if err := c.host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := deliveredName(t, c.probe); got != "Renamed" {
		t.Errorf("the plugin holds the list %q, want the renamed one", got)
	}
}
