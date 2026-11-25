package jitter_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/jitter"
)

func TestRandom_PicksWithinTheLimitAndSpreadsOverIt(t *testing.T) {
	t.Parallel()
	source := jitter.NewRandom(rand.New(rand.NewPCG(1, 2)))
	limit := 1500 * time.Millisecond
	seen := map[time.Duration]bool{}
	var lowest, highest = limit, time.Duration(0)

	for range 1000 {
		got := source.Pick(limit)
		if got < 0 || got > limit {
			t.Fatalf("pick = %v, want within 0 and %v", got, limit)
		}
		seen[got] = true
		lowest, highest = min(lowest, got), max(highest, got)
	}

	if len(seen) < 500 {
		t.Errorf("%d different picks in 1000, want them spread", len(seen))
	}
	if lowest > limit/20 || highest < limit-limit/20 {
		t.Errorf("picks reach from %v to %v, want both ends of 0 to %v", lowest, highest, limit)
	}
}

func TestRandom_PicksNothingForALimitOfNothing(t *testing.T) {
	t.Parallel()
	source := jitter.NewRandom(rand.New(rand.NewPCG(1, 2)))

	if got := source.Pick(0); got != 0 {
		t.Errorf("pick = %v, want 0", got)
	}
}
