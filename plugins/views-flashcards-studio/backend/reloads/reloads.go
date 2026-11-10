// Package reloads tells a window that something changed in the vaults, for as long as it listens.
package reloads

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// Moves is where the changes to the files of the vaults are reported.
type Moves interface {
	WatchMoves(ctx context.Context) (<-chan host.Move, error)
}

// UnavailableError is a failure to follow the changes, as against a listener that has gone.
type UnavailableError struct{ Err error }

func (e *UnavailableError) Error() string { return e.Err.Error() }

func (e *UnavailableError) Unwrap() error { return e.Err }

// Watch signals that nothing changed at once and whenever idle fires, and that something changed on
// each move, until ctx ends or the moves stop. The change itself is not carried: a window shows
// counts and the cards behind them, and asks for them again after any change. Invalidate runs
// before each signal of a change. A signal that fails ends the watch with its error.
func Watch(ctx context.Context, moves Moves, idle <-chan time.Time, invalidate func(), signal func(reload bool) error) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	moved, err := moves.WatchMoves(ctx)
	if err != nil {
		return &UnavailableError{Err: err}
	}

	// The stream sends an initial message before any change. A stream silent until a file changes
	// is indistinguishable from one that never opened.
	if err := signal(false); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-idle:
			// No change, which the opening message also signals. A closed page fails the write.
			if err := signal(false); err != nil {
				return err
			}
		case move, open := <-moved:
			if !open {
				return nil
			}
			if move.Err != nil {
				return &UnavailableError{Err: move.Err}
			}
			invalidate()
			if err := signal(true); err != nil {
				return err
			}
		}
	}
}
