package secret_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
)

// blockingStore stops each write of a value until the test releases it.
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
		plugin.Secret{ID: "s00000001", Name: "One", Group: "g"},
		plugin.Secret{ID: "s00000002", Name: "Two", Group: "g"},
	)
	store := blockingStore{keys: f.keys, entered: make(chan string, 2), release: make(chan struct{})}
	f.secrets = newSecrets(f.ledger, store, f.plugins, nil, nil, nil)

	var group sync.WaitGroup
	for _, id := range []string{"s00000001", "s00000002"} {
		group.Go(func() {
			if err := f.secrets.Items.Write.Execute(t.Context(), id, "value"); err != nil {
				t.Error(err)
			}
		})
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

// refreshBlocker stops each refresh until the test releases it.
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

func TestGateBlocksUntilPluginsAreConfigured(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "One", Group: "g"})
	gate := secret.NewGate()
	blocker := &refreshBlocker{entered: make(chan struct{}, 2), release: make(chan struct{})}
	f.secrets = newSecrets(f.ledger, f.keys, blocker, nil, nil, gate)

	first := make(chan error, 1)
	go func() { first <- f.secrets.Items.Rename.Execute(t.Context(), "s00000001", "Office") }()
	<-blocker.entered

	watch := make(chan error, 1)
	go func() { watch <- gate.Enter(t.Context()) }()
	second := make(chan error, 1)
	go func() { second <- f.secrets.Items.Move.Execute(t.Context(), "s00000001", "") }()
	select {
	case <-watch:
		t.Fatal("the file watch passed the gate while the plugins were being configured")
	case <-blocker.entered:
		t.Fatal("the second operation reached its refresh while the first one was configuring")
	case <-time.After(100 * time.Millisecond):
	}
	close(blocker.release)
	if err := <-first; err != nil {
		t.Error(err)
	}
	if err := <-watch; err != nil {
		t.Fatal(err)
	}
	gate.Leave()
	if err := <-second; err != nil {
		t.Error(err)
	}
}

func TestCanceledWaitOnGateChangesNothing(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "One", Group: "g"})
	gate := secret.NewGate()
	store := blockingStore{keys: f.keys, entered: make(chan string, 1), release: make(chan struct{})}
	f.secrets = newSecrets(f.ledger, store, f.plugins, nil, nil, gate)

	holder := make(chan error, 1)
	go func() { holder <- f.secrets.Items.Write.Execute(t.Context(), "s00000001", "value") }()
	<-store.entered

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := f.secrets.Items.Rename.Execute(ctx, "s00000001", "Office"); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want the context error", err)
	}
	close(store.release)
	if err := <-holder; err != nil {
		t.Error(err)
	}
	if f.ledger.list[0].Name != "One" {
		t.Errorf("name = %s, the canceled operation changed the list", f.ledger.list[0].Name)
	}
}

type failingLedger struct{ ledger }

var errDisk = errors.New("the disk is full")

func (*failingLedger) ChangeSecrets(func([]plugin.Secret) ([]plugin.Secret, error)) error {
	return errDisk
}

func TestCreateRollsBackValueWhenListWriteFailsAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	store := keys{}
	failing := &failingLedger{}
	canceling := cancelOnSet{keys: store, cancel: cancel}
	use := newSecrets(failing, canceling, &plugins{}, func() (string, error) { return "s00000001", nil }, nil, nil)

	if _, err := use.Items.Create.Execute(ctx, "g", "N", "value"); !errors.Is(err, errDisk) {
		t.Fatalf("error = %v", err)
	}
	if len(store) != 0 {
		t.Errorf("the store keeps %v after the list failed", store)
	}
}

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
	if err := f.secrets.Items.Write.Execute(t.Context(), "s00000042", "v"); !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("error = %v", err)
	}
	if len(f.plugins.readers) != 1 || f.plugins.readers[0] != "left-over" {
		t.Errorf("the plugins were refreshed: %v", f.plugins.readers)
	}
}
