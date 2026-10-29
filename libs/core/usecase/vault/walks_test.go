package vault_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// blockedWalk is a vault whose walk signals when it begins and then blocks until
// released, so two overlapping scans can be detected.
type blockedWalk struct {
	began chan struct{}
	let   chan struct{}
}

func newBlockedWalk() blockedWalk {
	return blockedWalk{began: make(chan struct{}, 2), let: make(chan struct{})}
}

func (h blockedWalk) Open(context.Context, string) (io.ReadSeekCloser, error) { return nil, nil }

func (h blockedWalk) WalkFiles(ctx context.Context, _ func(domain.FileStat) error) error {
	h.began <- struct{}{}
	select {
	case <-h.let:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (blockedWalk) List(context.Context, string) ([]domain.Entry, error) { return nil, nil }
func (blockedWalk) Read(context.Context, string) ([]byte, error)         { return nil, nil }

func (blockedWalk) StatFile(context.Context, string) (domain.FileStat, error) {
	return domain.FileStat{}, nil
}

// hasWalkBegun reports whether a walk began before the wait ran out.
func (h blockedWalk) hasWalkBegun() bool {
	select {
	case <-h.began:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// blockedWalks opens that one vault, whichever vault is requested.
type blockedWalks struct{ reader blockedWalk }

func (h blockedWalks) Open(domain.Vault) (vaultport.VaultReader, error) { return h.reader, nil }

// One index is walked a vault at a time: a second scan writing into it waits
// for the first to finish.
func TestOneWalkOfAVaultRunsAtATime(t *testing.T) {
	t.Parallel()
	target := newBlockedWalk()
	readers := blockedWalks{reader: target}
	db := openIndex(t)
	v := domain.Vault{ID: "one", Path: t.TempDir()}

	isDone := make(chan error, 2)
	first, second := scanner(readers, db), scanner(readers, db)
	go func() { _, err := first.Execute(t.Context(), v); isDone <- err }()
	if !target.hasWalkBegun() {
		t.Fatal("the first walk never began")
	}
	go func() { _, err := second.Execute(t.Context(), v); isDone <- err }()

	select {
	case <-target.began:
		t.Fatal("a second walk of the vault began while the first was still going")
	case <-time.After(100 * time.Millisecond):
	}
	close(target.let)
	if !target.hasWalkBegun() {
		t.Fatal("the second walk never took its turn")
	}
	for range 2 {
		if err := <-isDone; err != nil {
			t.Fatal(err)
		}
	}
}

// so a window, a terminal and a test suite do not queue behind each other for
// a vault they each keep their own index of.
func TestWalksIntoTwoIndexesDoNotWaitOnEachOther(t *testing.T) {
	t.Parallel()
	target := newBlockedWalk()
	readers := blockedWalks{reader: target}
	v := domain.Vault{ID: "one", Path: t.TempDir()}

	isDone := make(chan error, 2)
	for _, db := range []*container.Index{openIndex(t), openIndex(t)} {
		scan := scanner(readers, db)
		go func() { _, err := scan.Execute(t.Context(), v); isDone <- err }()
	}
	for range 2 {
		if !target.hasWalkBegun() {
			t.Fatal("a walk of one index waited on a walk of another")
		}
	}
	close(target.let)
	for range 2 {
		if err := <-isDone; err != nil {
			t.Fatal(err)
		}
	}
}
