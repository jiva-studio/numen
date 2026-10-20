package agentrun

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestModelCacheKeepsWhatWasListedWhileTheKeyStays(t *testing.T) {
	var cache ModelCache[string]
	asked := 0
	list := func(context.Context) ([]string, error) {
		asked++
		return []string{"a"}, nil
	}
	for range 2 {
		if got := cache.Get(t.Context(), "k", list); len(got) != 1 || got[0] != "a" {
			t.Fatalf("the models are %v", got)
		}
	}
	if asked != 1 {
		t.Errorf("the program was asked %d times, want 1", asked)
	}
	cache.Get(t.Context(), "other", list)
	if asked != 2 {
		t.Errorf("a new key asked %d times in all, want 2", asked)
	}
}

func TestModelCacheLeavesAFailedKeyAloneForAWhile(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := ModelCache[string]{Now: func() time.Time { return now }}
	failing := true
	asked := 0
	list := func(context.Context) ([]string, error) {
		asked++
		if failing {
			return nil, errors.New("down")
		}
		return []string{"a"}, nil
	}

	if got := cache.Get(t.Context(), "k", list); got != nil {
		t.Fatalf("a failed listing gave %v", got)
	}
	failing = false
	if got := cache.Get(t.Context(), "k", list); got != nil || asked != 1 {
		t.Fatalf("a key that just failed was asked again: %v after %d asks", got, asked)
	}
	now = now.Add(ModelFailureBackoff)
	if got := cache.Get(t.Context(), "k", list); len(got) != 1 {
		t.Errorf("the next listing gave %v", got)
	}
}

func TestModelCacheKeepsNothingFromAnEndedRequest(t *testing.T) {
	var cache ModelCache[string]
	list := func(ctx context.Context) ([]string, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []string{"a"}, nil
	}
	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if got := cache.Get(ended, "k", list); got != nil {
		t.Errorf("an ended request got %v", got)
	}
	if got := cache.Get(t.Context(), "k", list); len(got) != 1 {
		t.Errorf("the next request got %v, a failure of the ended one was kept", got)
	}
}
