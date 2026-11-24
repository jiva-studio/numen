package identity

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// changeSignal tells the listeners that something the identity of this device is made of changed: the record, the
// restore that waits for a chain, or the link in progress.
type changeSignal struct {
	mu        sync.Mutex
	listeners []func()
}

func (s *changeSignal) subscribe(listener func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *changeSignal) notify() {
	// Called under the lock of a store: a listener returns at once and takes none.
	s.mu.Lock()
	listeners := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, listener := range listeners {
		listener()
	}
}

// watchedKeys notifies the signal when the record of the identity is written or deleted.
type watchedKeys struct {
	pluginport.KeyStore
	signal *changeSignal
}

func (k watchedKeys) SetKey(ctx context.Context, name, key string) error {
	err := k.KeyStore.SetKey(ctx, name, key)
	if err == nil && name == keyName {
		k.signal.notify()
	}
	return err
}

func (k watchedKeys) DeleteKey(ctx context.Context, name string) error {
	err := k.KeyStore.DeleteKey(ctx, name)
	if err == nil && name == keyName {
		k.signal.notify()
	}
	return err
}
