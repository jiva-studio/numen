package reloads_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/reloads"
)

type moves struct {
	channel <-chan host.Move
	err     error
}

func (m moves) WatchMoves(context.Context) (<-chan host.Move, error) { return m.channel, m.err }

func TestWatchSignalsNothingChangedAtOnceAndAReloadOnEachMove(t *testing.T) {
	channel := make(chan host.Move, 2)
	channel <- host.Move{}
	close(channel)
	var signals []bool
	invalidated := 0

	err := reloads.Watch(t.Context(), moves{channel: channel}, nil, func() { invalidated++ }, func(reload bool) error {
		signals = append(signals, reload)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 2 || signals[0] || !signals[1] || invalidated != 1 {
		t.Errorf("signals %v with %d invalidations, want [false true] and one", signals, invalidated)
	}
}

func TestWatchSignalsNothingChangedWhenIdle(t *testing.T) {
	idle := make(chan time.Time, 1)
	idle <- time.Time{}
	count := 0

	err := reloads.Watch(t.Context(), moves{channel: make(chan host.Move)}, idle, func() {}, func(bool) error {
		count++
		if count == 2 {
			return errors.New("page closed")
		}
		return nil
	})
	if err == nil || err.Error() != "page closed" {
		t.Fatalf("expected the failure of the signal, got %v", err)
	}
}

func TestWatchIsUnavailableWhenTheMovesCannotBeFollowed(t *testing.T) {
	err := reloads.Watch(t.Context(), moves{err: errors.New("watch boom")}, nil, func() {}, func(bool) error { return nil })
	if !errors.As(err, new(*reloads.UnavailableError)) {
		t.Fatalf("expected an UnavailableError, got %v", err)
	}

	channel := make(chan host.Move, 1)
	channel <- host.Move{Err: errors.New("move boom")}
	err = reloads.Watch(t.Context(), moves{channel: channel}, nil, func() {}, func(bool) error { return nil })
	if !errors.As(err, new(*reloads.UnavailableError)) {
		t.Fatalf("expected an UnavailableError for a failed move, got %v", err)
	}
}

func TestWatchEndsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	err := reloads.Watch(ctx, moves{channel: make(chan host.Move)}, nil, func() {}, func(bool) error {
		cancel()
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil when the context ends, got %v", err)
	}
}
