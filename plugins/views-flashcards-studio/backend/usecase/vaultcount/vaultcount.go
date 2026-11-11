// Package vaultcount counts the cards due in every vault of the installation: a vault at a time,
// several of them alongside each other, each sent as it is finished.
package vaultcount

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

// Interval is the interval at which a stream with no counts sends a message, so a page can tell an
// open stream from a closed one.
const Interval = time.Second

// atOnce is how many vaults are counted alongside each other.
//
// Counting one reads every deck it holds and replays its whole answer log, so each of them holds a
// vault's worth of cards while it runs.
const atOnce = 4

// Indexer brings the index of one vault up to date.
type Indexer interface {
	EnsureIndexed(ctx context.Context) (host.IndexState, error)
}

// DueCounter counts the cards of a vault by what is due.
type DueCounter interface {
	Execute(ctx context.Context, v domain.Vault) (flashcards.CardsDue, error)
}

// Result is how the count of one vault ended. A vault that was not counted says why: it is being
// indexed, or Unread holds the reason.
type Result struct {
	Vault      domain.Vault
	IsIndexing bool
	Unread     string
	Counts     flashcards.CardsDue
}

// Count counts one vault. The vault is brought up to date before it is counted. Nothing is counted
// from a walk half done, and the numbers arrive with the count that the finished walk wakes.
func Count(ctx context.Context, index Indexer, due DueCounter, v domain.Vault) Result {
	one := Result{Vault: v, IsIndexing: false, Unread: "", Counts: flashcards.CardsDue{ReviewCards: 0, Due: 0, New: 0, Decks: nil, Presets: nil}}
	state, err := index.EnsureIndexed(ctx)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	switch state.Status {
	case host.Indexing:
		one.IsIndexing = true
		return one
	case host.IndexFailed:
		one.Unread = state.Reason
		return one
	case host.Indexed:
	}
	counts, err := due.Execute(ctx, v)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	one.Counts = counts
	return one
}

// Start counts the vaults, at most atOnce at a time, and sends each result on the channel returned,
// which is closed when every vault is counted or ctx ends.
func Start[T any](ctx context.Context, vaults []domain.Vault, count func(context.Context, domain.Vault) T) <-chan T {
	counted := make(chan T)
	go func() {
		defer close(counted)
		var running sync.WaitGroup
		room := make(chan struct{}, atOnce)
		for _, one := range vaults {
			v := one
			select {
			case room <- struct{}{}:
			case <-ctx.Done():
				running.Wait()
				return
			}
			running.Go(func() {
				defer func() { <-room }()
				select {
				case counted <- count(ctx, v):
				case <-ctx.Done():
				}
			})
		}
		running.Wait()
	}()
	return counted
}

// Forward hands each result to onCounted until the channel is closed, and calls onIdle whenever idle
// fires first. A failure of either ends it.
func Forward[T any](counted <-chan T, idle <-chan time.Time, onCounted func(T) error, onIdle func() error) error {
	for counted != nil {
		select {
		case one, open := <-counted:
			if !open {
				counted = nil
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="Setting counted to nil exits the loop on either continue or break."
				continue
			}
			if err := onCounted(one); err != nil {
				return err
			}
		case <-idle:
			if err := onIdle(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Watch counts each vault in vaults and calls onCounted as each finishes, calling onIdle initially
// and whenever Interval elapses without a count completing.
func Watch[T any](
	ctx context.Context,
	vaults []domain.Vault,
	count func(context.Context, domain.Vault) T,
	onCounted func(T) error,
	onIdle func() error,
) error {
	if err := onIdle(); err != nil {
		return err
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	repeat := clock.NewTicker(Interval)
	defer repeat.Stop()

	return Forward(Start(ctx, vaults, count), repeat.C, func(one T) error {
		if err := onCounted(one); err != nil {
			return err
		}
		repeat.Reset(Interval)
		return nil
	}, onIdle)
}
