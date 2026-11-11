package plugin_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

// cancelSetup is a host whose sends wait for the lock, with the store and the list a change would touch.
type cancelSetup struct {
	host   *pluginhost.Host
	probe  *configured
	store  *keys
	source *listing
}

func newCancelSetup(t *testing.T) *cancelSetup {
	t.Helper()
	store := newKeys(map[string]string{"secret/" + secretID: fromKeychain})
	host, probe := newSecretHost(t, store)
	source := newListing("Work")
	host.Config.SetSecretSource(source.source())
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	return &cancelSetup{host: host, probe: probe, store: store, source: source}
}

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
