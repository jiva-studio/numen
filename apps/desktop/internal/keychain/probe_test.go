package keychain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// blockingBackend answers nothing until it is released.
type blockingBackend struct {
	release chan struct{}
	probes  int
	mu      sync.Mutex
}

func newBlockingBackend() *blockingBackend { return &blockingBackend{release: make(chan struct{})} }

func (b *blockingBackend) Get(string) (string, error) {
	b.mu.Lock()
	b.probes++
	b.mu.Unlock()
	<-b.release
	return "", nil
}
func (b *blockingBackend) Set(string, string) error { <-b.release; return nil }
func (b *blockingBackend) Delete(string) error      { <-b.release; return nil }

func TestACallWaitingForTheKeychainReturnsWhenItsContextEnds(t *testing.T) {
	backend := newBlockingBackend()
	store := newTestStore(t.TempDir())
	store.backend = backend
	t.Cleanup(func() { close(backend.release) })

	for name, call := range map[string]func(context.Context) error{
		"set": func(ctx context.Context) error { return store.SetKey(ctx, entryName, valueOne) },
		"get": func(ctx context.Context) error { _, err := store.GetKey(ctx, entryName); return err },
		"del": func(ctx context.Context) error { return store.DeleteKey(ctx, entryName) },
	} {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		returned := make(chan error, 1)
		go func() { returned <- call(ctx) }()
		select {
		case err := <-returned:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("%s: error = %v, want the deadline", name, err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("%s: the call waits for the keychain after its context ended", name)
		}
		cancel()
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.probes != 1 {
		t.Errorf("the keychain was probed %d times by callers that waited together, want 1", backend.probes)
	}
}

func TestACallWaitingForASharedProbeReturnsWithItsOwnContext(t *testing.T) {
	backend := newBlockingBackend()
	store := newTestStore(t.TempDir())
	store.backend = backend
	t.Cleanup(func() { close(backend.release) })

	ctx, cancel := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() { _, err := store.GetKey(ctx, entryName); first <- err }()
	time.Sleep(50 * time.Millisecond)

	second, stop := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer stop()
	_, err := store.GetKey(second, entryName)
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Errorf("the first caller returned %v, want its cancellation", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a second caller returned %v, want the deadline of its own context", err)
	}
}
