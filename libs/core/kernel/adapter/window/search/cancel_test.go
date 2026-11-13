package search

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// waitingSource is a search source that answers only when the context of its call ends, as a plugin
// that is still embedding the query.
type waitingSource struct {
	started chan struct{}
	ended   chan error
}

func (s waitingSource) Rank(ctx context.Context, _ domain.Vault, _ search.Query) (search.Ranking, error) {
	close(s.started)
	<-ctx.Done()
	s.ended <- ctx.Err()
	return search.Ranking{}, ctx.Err()
}

func newWaitingHandler(t *testing.T, source waitingSource) *Handler {
	t.Helper()
	failed := func(err error) { t.Errorf("a cancelled search was reported as a failure of its source: %v", err) }
	finds := search.New(nil, nil, nil, []search.Participant{source}, failed)
	return NewHandler(
		shownVault{v: domain.Vault{ID: "v1", Name: "Garden"}},
		search.NewFindNames(names{}, tabKinds{}),
		search.NewFindPassages(finds, titles{}, types{}, tabKinds{}),
	)
}

func TestCancelSearchEndsTheContextOfTheSearchWithItsID(t *testing.T) {
	source := waitingSource{started: make(chan struct{}), ended: make(chan error, 1)}
	handler := newWaitingHandler(t, source)

	answered := make(chan error, 1)
	go func() {
		_, err := handler.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
			Query: "seeds in winter", Mode: v1.SearchMode_SEARCH_MODE_MEANING, SearchId: "query-7",
		}))
		answered <- err
	}()
	<-source.started

	if _, err := handler.CancelSearch(t.Context(), connect.NewRequest(&v1.CancelSearchRequest{SearchId: "query-7"})); err != nil {
		t.Fatal(err)
	}

	if err := <-source.ended; !errors.Is(err, context.Canceled) {
		t.Fatalf("the source call ended with %v, want context.Canceled", err)
	}
	if err := <-answered; connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("the search answered %v, want a cancelled call", err)
	}
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.running) != 0 {
		t.Errorf("the handler still holds %d searches after they ended", len(handler.running))
	}
}

func TestCancelSearchLeavesTheSearchOfAnotherIDRunning(t *testing.T) {
	source := waitingSource{started: make(chan struct{}), ended: make(chan error, 1)}
	handler := newWaitingHandler(t, source)

	ctx, stop := context.WithCancel(t.Context())
	answered := make(chan error, 1)
	go func() {
		_, err := handler.SearchPassages(ctx, connect.NewRequest(&v1.SearchPassagesRequest{
			Query: "seeds in winter", Mode: v1.SearchMode_SEARCH_MODE_MEANING, SearchId: "query-8",
		}))
		answered <- err
	}()
	<-source.started

	if _, err := handler.CancelSearch(t.Context(), connect.NewRequest(&v1.CancelSearchRequest{SearchId: "query-7"})); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-source.ended:
		t.Fatalf("the cancel of another id ended the search: %v", err)
	default:
	}

	stop()
	<-source.ended
	<-answered
}
