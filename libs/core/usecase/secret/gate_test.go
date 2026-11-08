package secret_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// blockingStore stops each write of a value until the test lets it go.
type blockingStore struct {
	keys
	entered chan string
	release chan struct{}
}

func (b blockingStore) SetKey(ctx context.Context, name, value string) error {
	b.entered <- name
	<-b.release
	return b.keys.SetKey(ctx, name, value)
}

func TestTwoOperationsNeverInterleaveTheirChanges(t *testing.T) {
	f := newFixture(
		secret.Secret{ID: "s00000001", Name: "One", Group: "g"},
		secret.Secret{ID: "s00000002", Name: "Two", Group: "g"},
	)
	store := blockingStore{keys: f.keys, entered: make(chan string, 2), release: make(chan struct{})}
	f.secrets = secret.NewSecrets(f.ledger, store, f.plugins, nil, nil, nil)

	var group sync.WaitGroup
	for _, id := range []string{"s00000001", "s00000002"} {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := f.secrets.Write.Execute(t.Context(), id, "value"); err != nil {
				t.Error(err)
			}
		}()
	}
	<-store.entered
	select {
	case name := <-store.entered:
		t.Fatalf("the second operation wrote %s while the first one was still changing", name)
	case <-time.After(100 * time.Millisecond):
	}
	close(store.release)
	group.Wait()
}

// refreshBlocker stops each refresh until the test lets it go.
type refreshBlocker struct {
	plugins
	entered chan struct{}
	release chan struct{}
}

func (r *refreshBlocker) Refresh(context.Context, ...string) error {
	r.entered <- struct{}{}
	<-r.release
	return nil
}

func TestTheSendToPluginsDoesNotHoldBackTheNextOperation(t *testing.T) {
	f := newFixture(secret.Secret{ID: "s00000001", Name: "One", Group: "g"})
	blocker := &refreshBlocker{entered: make(chan struct{}, 1), release: make(chan struct{})}
	f.secrets = secret.NewSecrets(f.ledger, f.keys, blocker, nil, nil, nil)

	first := make(chan error, 1)
	go func() { first <- f.secrets.Rename.Execute(t.Context(), "s00000001", "Office") }()
	<-blocker.entered

	second := make(chan error, 1)
	go func() { second <- f.secrets.Move.Execute(t.Context(), "s00000001", "") }()
	select {
	case <-blocker.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the second operation did not reach its own refresh while the first one was sending")
	}
	close(blocker.release)
	if err := <-first; err != nil {
		t.Error(err)
	}
	if err := <-second; err != nil {
		t.Error(err)
	}
}

func TestAnOperationWhoseContextEndedWhileItWaitsForTheGateChangesNothing(t *testing.T) {
	f := newFixture(secret.Secret{ID: "s00000001", Name: "One", Group: "g"})
	gate := secret.NewGate()
	store := blockingStore{keys: f.keys, entered: make(chan string, 1), release: make(chan struct{})}
	f.secrets = secret.NewSecrets(f.ledger, store, f.plugins, nil, nil, gate)

	holder := make(chan error, 1)
	go func() { holder <- f.secrets.Write.Execute(t.Context(), "s00000001", "value") }()
	<-store.entered

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := f.secrets.Rename.Execute(ctx, "s00000001", "Office"); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want the context error", err)
	}
	close(store.release)
	if err := <-holder; err != nil {
		t.Error(err)
	}
	if f.ledger.list[0].Name != "One" {
		t.Errorf("name = %s, the cancelled operation changed the list", f.ledger.list[0].Name)
	}
}

// failingLedger refuses to change the list.
type failingLedger struct{ ledger }

var errDisk = errors.New("the disk is full")

func (*failingLedger) ChangeSecrets(func([]secret.Secret) ([]secret.Secret, error)) error {
	return errDisk
}

func TestCreateRemovesTheValueAlthoughTheContextEndedBeforeTheListFailed(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	store := keys{}
	failing := &failingLedger{}
	cancelling := cancelOnSet{keys: store, cancel: cancel}
	use := secret.NewSecrets(failing, cancelling, &plugins{}, func() (string, error) { return "s00000001", nil }, nil, nil)

	if _, err := use.Create.Execute(ctx, "g", "N", "value"); !errors.Is(err, errDisk) {
		t.Fatalf("error = %v", err)
	}
	if len(store) != 0 {
		t.Errorf("the store keeps %v after the list failed", store)
	}
}

// cancelOnSet ends the context of the caller after it stores a value, and refuses to delete with it.
type cancelOnSet struct {
	keys
	cancel context.CancelFunc
}

func (c cancelOnSet) SetKey(ctx context.Context, name, value string) error {
	err := c.keys.SetKey(ctx, name, value)
	c.cancel()
	return err
}

func (c cancelOnSet) DeleteKey(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.keys.DeleteKey(ctx, name)
}

func TestAFailedChangeRefreshesNoPlugin(t *testing.T) {
	f := newFixture()
	f.plugins.readers = []string{"left-over"}
	if err := f.secrets.Write.Execute(t.Context(), "s00000042", "v"); !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("error = %v", err)
	}
	if len(f.plugins.readers) != 1 || f.plugins.readers[0] != "left-over" {
		t.Errorf("the plugins were refreshed: %v", f.plugins.readers)
	}
}
