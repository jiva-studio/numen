package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestSessionPool(t *testing.T) {
	var destroyed atomic.Int32
	destroy := func(s int) {
		destroyed.Add(1)
	}

	sessions := []int{10, 20}
	pool := newSessionPool(sessions, 2, destroy)

	ctx := t.Context()
	s1, err := pool.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire s1 failed: %v", err)
	}
	s2, err := pool.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire s2 failed: %v", err)
	}
	if !((s1 == 10 && s2 == 20) || (s1 == 20 && s2 == 10)) {
		t.Errorf("expected sessions 10 and 20, got %d and %d", s1, s2)
	}

	pool.release(s1)
	pool.release(s2)

	if len(pool.idle) != 2 {
		t.Fatalf("expected 2 idle sessions before close, got %d", len(pool.idle))
	}

	pool.close()

	if len(pool.idle) != 0 {
		t.Errorf("expected idle channel drained on close, got %d", len(pool.idle))
	}

	if destroyed.Load() != 2 {
		t.Fatalf("expected 2 sessions destroyed, got %d", destroyed.Load())
	}

	_, err = pool.acquire(ctx)
	if !errors.Is(err, errClosed) {
		t.Fatalf("expected errClosed, got %v", err)
	}
}

func TestSessionPoolLanesFloor(t *testing.T) {
	sessions := []int{42}
	pool := newSessionPool(sessions, 0, func(int) {})
	if pool.lanes != 1 {
		t.Errorf("expected lanes clamped to 1, got %d", pool.lanes)
	}
	s, err := pool.acquire(t.Context())
	if err != nil || s != 42 {
		t.Errorf("acquire failed: %v, session: %d", err, s)
	}
	pool.release(s)
	pool.close()
}

func TestSessionPoolContextCancellation(t *testing.T) {
	sessions := []int{1}
	pool := newSessionPool(sessions, 1, func(int) {})

	s, err := pool.acquire(t.Context())
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = pool.acquire(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	pool.release(s)
	pool.close()
}
