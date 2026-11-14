package vaultcount_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/vaultcount"
)

type indexer struct {
	state host.IndexState
	err   error
}

func (i indexer) EnsureIndexed(context.Context) (host.IndexState, error) { return i.state, i.err }

type counter struct {
	counts review.CardsDue
	err    error
}

func (c counter) Execute(context.Context, domain.Vault) (review.CardsDue, error) {
	return c.counts, c.err
}

func TestCountReportsAVaultStillBeingIndexedWithoutCountingIt(t *testing.T) {
	got := vaultcount.Count(t.Context(), indexer{state: host.IndexState{Status: host.Indexing}}, counter{err: errors.New("unreached")}, domain.Vault{ID: "a"})
	if !got.IsIndexing || got.Unread != "" {
		t.Errorf("a vault being indexed came back as %+v", got)
	}
}

func TestCountReturnsTheUnreadReason(t *testing.T) {
	failed := indexer{state: host.IndexState{Status: host.IndexFailed, Reason: "walk failed"}}
	if got := vaultcount.Count(t.Context(), failed, counter{}, domain.Vault{}); got.Unread != "walk failed" {
		t.Errorf("a vault that failed to index came back as %+v", got)
	}
	if got := vaultcount.Count(t.Context(), indexer{err: errors.New("ensure boom")}, counter{}, domain.Vault{}); got.Unread != "ensure boom" {
		t.Errorf("a vault that could not be ensured came back as %+v", got)
	}
	ready := indexer{state: host.IndexState{Status: host.Indexed}}
	if got := vaultcount.Count(t.Context(), ready, counter{err: errors.New("due boom")}, domain.Vault{}); got.Unread != "due boom" {
		t.Errorf("a vault that could not be counted came back as %+v", got)
	}
}

func TestCountCarriesTheCountOfAnIndexedVault(t *testing.T) {
	ready := indexer{state: host.IndexState{Status: host.Indexed}}
	got := vaultcount.Count(t.Context(), ready, counter{counts: review.CardsDue{Due: 3}}, domain.Vault{ID: "a"})
	if got.Counts.Due != 3 || got.Unread != "" || got.IsIndexing || got.Vault.ID != "a" {
		t.Errorf("an indexed vault came back as %+v", got)
	}
}

func TestStartCountsEveryVaultAndThenCloses(t *testing.T) {
	var vaults []domain.Vault
	for _, id := range []domain.VaultID{"a", "b", "c", "d", "e", "f", "g"} {
		vaults = append(vaults, domain.Vault{ID: id})
	}
	var seen []string
	for id := range vaultcount.Start(t.Context(), vaults, func(_ context.Context, v domain.Vault) string { return string(v.ID) }) {
		seen = append(seen, id)
	}
	slices.Sort(seen)
	if !slices.Equal(seen, []string{"a", "b", "c", "d", "e", "f", "g"}) {
		t.Errorf("counted %v", seen)
	}
}

func TestStartStopsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	counted := vaultcount.Start(ctx, []domain.Vault{{ID: "a"}, {ID: "b"}}, func(context.Context, domain.Vault) int { return 1 })
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("the counters did not stop")
	case <-drain(counted):
	}
}

func drain(counted <-chan int) <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		for range counted {
		}
	}()
	return stop
}

func TestForwardStopsOnTheFirstCallbackError(t *testing.T) {
	counted := make(chan int, 3)
	counted <- 1
	counted <- 2
	counted <- 3
	close(counted)
	var got []int
	err := vaultcount.Forward(counted, nil, func(n int) error {
		got = append(got, n)
		if n == 2 {
			return errors.New("page closed")
		}
		return nil
	}, func() error { return nil })
	if err == nil || !slices.Equal(got, []int{1, 2}) {
		t.Errorf("forwarded %v with %v", got, err)
	}
}

func TestForwardCallsIdleWhileNothingArrives(t *testing.T) {
	idle := make(chan time.Time, 1)
	idle <- time.Time{}
	counted := make(chan int)
	idles := 0
	err := vaultcount.Forward(counted, idle, func(int) error { return nil }, func() error {
		idles++
		close(counted)
		return nil
	})
	if err != nil || idles != 1 {
		t.Errorf("idle ran %d times, ended with %v", idles, err)
	}
}
