package vault_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// heldWalk is a vault whose walk says when it began and then waits to be let
// go, so that two scans of it can be caught overlapping.
type heldWalk struct {
	began chan struct{}
	let   chan struct{}
}

func newHeldWalk() heldWalk {
	return heldWalk{began: make(chan struct{}, 2), let: make(chan struct{})}
}

func (h heldWalk) Open(context.Context, string) (io.ReadSeekCloser, error) { return nil, nil }

func (h heldWalk) Walk(ctx context.Context, _ func(domain.Fingerprint) error) error {
	h.began <- struct{}{}
	select {
	case <-h.let:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (heldWalk) List(context.Context, string) ([]domain.Entry, error) { return nil, nil }
func (heldWalk) Read(context.Context, string) ([]byte, error)         { return nil, nil }

func (heldWalk) Stat(context.Context, string) (domain.Fingerprint, error) {
	return domain.Fingerprint{}, nil
}

// hasWalkBegun says whether a walk began before the wait ran out.
func (h heldWalk) hasWalkBegun() bool {
	select {
	case <-h.began:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// heldWalks opens that one vault, whichever vault is asked for.
type heldWalks struct{ reader heldWalk }

func (h heldWalks) Open(domain.Vault) (port.VaultReader, error) { return h.reader, nil }

// One index is walked a vault at a time: a second scan writing into it waits
// for the first to finish.
func TestOneWalkOfAVaultRunsAtATime(t *testing.T) {
	t.Parallel()
	held := newHeldWalk()
	readers := heldWalks{reader: held}
	db := openIndex(t)
	v := domain.Vault{ID: "one", Path: t.TempDir()}

	done := make(chan error, 2)
	first, second := scanner(readers, db), scanner(readers, db)
	go func() { _, err := first.Execute(t.Context(), v); done <- err }()
	if !held.hasWalkBegun() {
		t.Fatal("the first walk never began")
	}
	go func() { _, err := second.Execute(t.Context(), v); done <- err }()

	select {
	case <-held.began:
		t.Fatal("a second walk of the vault began while the first was still going")
	case <-time.After(100 * time.Millisecond):
	}
	close(held.let)
	if !held.hasWalkBegun() {
		t.Fatal("the second walk never took its turn")
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

// Two indexes are two sets of turns. Nothing process-wide stands between them,
// so a window, a terminal and a test suite do not queue behind each other for
// a vault they each keep their own index of.
func TestWalksIntoTwoIndexesDoNotWaitOnEachOther(t *testing.T) {
	t.Parallel()
	held := newHeldWalk()
	readers := heldWalks{reader: held}
	v := domain.Vault{ID: "one", Path: t.TempDir()}

	done := make(chan error, 2)
	for _, db := range []*container.Index{openIndex(t), openIndex(t)} {
		scan := scanner(readers, db)
		go func() { _, err := scan.Execute(t.Context(), v); done <- err }()
	}
	for range 2 {
		if !held.hasWalkBegun() {
			t.Fatal("a walk of one index waited on a walk of another")
		}
	}
	close(held.let)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
