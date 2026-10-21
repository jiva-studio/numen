package agentrun

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestModelCachePreservesWhatWasListedWhileTheKeyStays(t *testing.T) {
	var cache ModelCache[string]
	request := 0
	list := func(context.Context) ([]string, error) {
		request++
		return []string{"a"}, nil
	}
	for range 2 {
		if got := cache.Get(t.Context(), "k", list); len(got) != 1 || got[0] != "a" {
			t.Fatalf("the models are %v", got)
		}
	}
	if request != 1 {
		t.Errorf("the program was asked %d times, want 1", request)
	}
	cache.Get(t.Context(), "other", list)
	if request != 2 {
		t.Errorf("a new key asked %d times in all, want 2", request)
	}
}

func TestModelCachePreservesAFailedKeyAloneForAWhile(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := ModelCache[string]{Now: func() time.Time { return now }}
	failing := true
	request := 0
	list := func(context.Context) ([]string, error) {
		request++
		if failing {
			return nil, errors.New("down")
		}
		return []string{"a"}, nil
	}

	if got := cache.Get(t.Context(), "k", list); got != nil {
		t.Fatalf("a failed listing gave %v", got)
	}
	failing = false
	if got := cache.Get(t.Context(), "k", list); got != nil || request != 1 {
		t.Fatalf("a key that just failed was asked again: %v after %d asks", got, request)
	}
	now = now.Add(ModelFailureBackoff)
	if got := cache.Get(t.Context(), "k", list); len(got) != 1 {
		t.Errorf("the next listing gave %v", got)
	}
}

func TestModelCachePreservesNothingFromAnEndedRequest(t *testing.T) {
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
