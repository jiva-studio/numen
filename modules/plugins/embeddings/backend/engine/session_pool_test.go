package engine

import (
	"context"
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

	ctx := context.Background()
	s1, err := pool.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire s1 failed: %v", err)
	}
	s2, err := pool.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire s2 failed: %v", err)
	}

	pool.release(s1)
	pool.release(s2)

	pool.close()

	if destroyed.Load() != 2 {
		t.Fatalf("expected 2 sessions destroyed, got %d", destroyed.Load())
	}

	_, err = pool.acquire(ctx)
	if err != errClosed {
		t.Fatalf("expected errClosed, got %v", err)
	}
}
