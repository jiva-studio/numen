package search

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// waitingSource is a search source that answers only when the context of its call ends, as a plugin
// that is still embedding the query.
type waitingSource struct {
	started chan struct{}
	ended   chan error
}

func newWaitingSource() waitingSource {
	return waitingSource{started: make(chan struct{}), ended: make(chan error, 1)}
}

func (s waitingSource) Rank(ctx context.Context, _ domain.Vault, _ search.Query) (search.Ranking, error) {
	close(s.started)
	<-ctx.Done()
	s.ended <- ctx.Err()
	return search.Ranking{}, ctx.Err()
}

// waitingNames is a name index that answers only when the context of its call ends.
type waitingNames struct {
	started chan struct{}
	ended   chan error
}

func (n waitingNames) GetNameMatches(ctx context.Context, _ domain.VaultID, _ string, _ int) ([]domain.NameMatch, error) {
	close(n.started)
	<-ctx.Done()
	n.ended <- ctx.Err()
	return nil, ctx.Err()
}

func newWaitingHandler(t *testing.T, source waitingSource, clock *testsupport.ManualClock) *Handler {
	t.Helper()
	failed := func(err error) { t.Errorf("a cancelled search was reported as a failure of its source: %v", err) }
	finds := search.New(nil, nil, nil, []search.Participant{source}, failed)
	return NewHandler(
		shownVault{v: domain.Vault{ID: "v1", Name: "Garden"}},
		search.NewFindNames(names{}, tabKinds{}),
		search.NewFindPassages(finds, titles{}, types{}, tabKinds{}),
		clock,
	)
}

func searchMeaning(handler *Handler, ctx context.Context, id string) <-chan error {
	answered := make(chan error, 1)
	go func() {
		_, err := handler.SearchPassages(ctx, connect.NewRequest(&v1.SearchPassagesRequest{
			Query: "seeds in winter", Mode: v1.SearchMode_SEARCH_MODE_MEANING, SearchId: id,
		}))
		answered <- err
	}()
	return answered
}

func cancelSearch(t *testing.T, handler *Handler, id string) {
	t.Helper()
	if _, err := handler.CancelSearch(t.Context(), connect.NewRequest(&v1.CancelSearchRequest{SearchId: id})); err != nil {
		t.Fatal(err)
	}
}

func TestCancelSearchEndsTheContextOfTheSearchWithItsID(t *testing.T) {
	source := newWaitingSource()
	handler := newWaitingHandler(t, source, testsupport.NewClock())

	answered := searchMeaning(handler, t.Context(), "query-7")
	<-source.started

	cancelSearch(t, handler, "query-7")

	if err := <-source.ended; !errors.Is(err, context.Canceled) {
		t.Fatalf("the source call ended with %v, want context.Canceled", err)
	}
	if err := <-answered; connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("the search answered %v, want a cancelled call", err)
	}
	handler.searches.mu.Lock()
	defer handler.searches.mu.Unlock()
	if len(handler.searches.running) != 0 {
		t.Errorf("the handler still holds %d searches after they ended", len(handler.searches.running))
	}
}

func TestCancelSearchLeavesTheSearchOfAnotherIDRunning(t *testing.T) {
	source := newWaitingSource()
	handler := newWaitingHandler(t, source, testsupport.NewClock())

	ctx, stop := context.WithCancel(t.Context())
	answered := searchMeaning(handler, ctx, "query-8")
	<-source.started

	cancelSearch(t, handler, "query-7")
	select {
	case err := <-source.ended:
		t.Fatalf("the cancel of another id ended the search: %v", err)
	default:
	}

	stop()
	<-source.ended
	<-answered
}

func TestCancelSearchEndsTheNameSearchWithItsID(t *testing.T) {
	index := waitingNames{started: make(chan struct{}), ended: make(chan error, 1)}
	handler := NewHandler(
		shownVault{v: domain.Vault{ID: "v1", Name: "Garden"}},
		search.NewFindNames(index, tabKinds{}),
		search.NewFindPassages(search.Search{}, titles{}, types{}, tabKinds{}),
		testsupport.NewClock(),
	)

	answered := make(chan error, 1)
	go func() {
		_, err := handler.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "seed", SearchId: "names-3"}))
		answered <- err
	}()
	<-index.started

	cancelSearch(t, handler, "names-3")

	if err := <-index.ended; !errors.Is(err, context.Canceled) {
		t.Fatalf("the name index call ended with %v, want context.Canceled", err)
	}
	if err := <-answered; connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("the name search answered %v, want a cancelled call", err)
	}
}

func TestCancelSearchThatArrivesBeforeItsSearchCancelsTheSearch(t *testing.T) {
	source := newWaitingSource()
	handler := newWaitingHandler(t, source, testsupport.NewClock())

	cancelSearch(t, handler, "query-9")
	err := <-searchMeaning(handler, t.Context(), "query-9")

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("the search answered %v, want a cancelled call", err)
	}
	select {
	case <-source.started:
		t.Fatal("the source was asked for a search cancelled before it started")
	default:
	}
}

func TestCancelSearchThatArrivedBeforeItsSearchCancelsItOnce(t *testing.T) {
	source := newWaitingSource()
	handler := newWaitingHandler(t, source, testsupport.NewClock())

	cancelSearch(t, handler, "query-9")
	<-searchMeaning(handler, t.Context(), "query-9")

	ctx, stop := context.WithCancel(t.Context())
	answered := searchMeaning(handler, ctx, "query-9")
	<-source.started
	stop()
	<-source.ended
	<-answered
}

func TestCancelSearchThatArrivedLongBeforeItsSearchIsForgotten(t *testing.T) {
	source := newWaitingSource()
	clock := testsupport.NewClock()
	handler := newWaitingHandler(t, source, clock)

	cancelSearch(t, handler, "query-10")
	clock.Advance(earlyCancelLife)

	ctx, stop := context.WithCancel(t.Context())
	answered := searchMeaning(handler, ctx, "query-10")
	<-source.started
	stop()
	<-source.ended
	<-answered
}

func TestCancelSearchKeepsTheNewestEarlyCancelsWithinTheLimit(t *testing.T) {
	source := newWaitingSource()
	clock := testsupport.NewClock()
	handler := newWaitingHandler(t, source, clock)

	cancelSearch(t, handler, "query-first")
	for i := range earlyCancelLimit {
		clock.Advance(1)
		cancelSearch(t, handler, fmt.Sprintf("query-%d", i))
	}

	handler.searches.mu.Lock()
	kept := len(handler.searches.early)
	handler.searches.mu.Unlock()
	if kept != earlyCancelLimit {
		t.Fatalf("the handler keeps %d early cancels, want %d", kept, earlyCancelLimit)
	}

	ctx, stop := context.WithCancel(t.Context())
	answered := searchMeaning(handler, ctx, "query-first")
	<-source.started
	stop()
	<-source.ended
	<-answered

	if err := <-searchMeaning(handler, t.Context(), fmt.Sprintf("query-%d", earlyCancelLimit-1)); connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("the search of the newest early cancel answered %v, want a cancelled call", err)
	}
}
