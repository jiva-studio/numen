package searchcancel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
)

const (
	earlyCancelLife  = 5 * time.Second
	earlyCancelLimit = 256
)

func isCanceled(ctx context.Context) bool {
	return ctx.Err() != nil
}

func TestCancelEndsTheContextOfTheRunningSearchWithTheID(t *testing.T) {
	searches := searchcancel.NewSearches(testsupport.NewClock())
	ctx, done := searches.Start(t.Context(), "query-7")
	defer done()

	searches.Cancel("query-7")

	if !isCanceled(ctx) {
		t.Fatal("the search is running after the cancel of its id")
	}
}

func TestCancelKeepsTheSearchOfAnotherIDRunning(t *testing.T) {
	searches := searchcancel.NewSearches(testsupport.NewClock())
	ctx, done := searches.Start(t.Context(), "query-8")
	defer done()

	searches.Cancel("query-7")

	if isCanceled(ctx) {
		t.Fatal("the cancel of another id ended the search")
	}
}

func TestCancelBeforeTheSearchStartsTheSearchCanceled(t *testing.T) {
	searches := searchcancel.NewSearches(testsupport.NewClock())

	searches.Cancel("query-9")
	ctx, done := searches.Start(t.Context(), "query-9")
	defer done()

	if !isCanceled(ctx) {
		t.Fatal("a search canceled before it started is running")
	}
}

func TestCancelBeforeTheSearchCancelsOnlyOneSearch(t *testing.T) {
	searches := searchcancel.NewSearches(testsupport.NewClock())

	searches.Cancel("query-9")
	_, first := searches.Start(t.Context(), "query-9")
	first()
	ctx, done := searches.Start(t.Context(), "query-9")
	defer done()

	if isCanceled(ctx) {
		t.Fatal("the second search with the id started canceled")
	}
}

func TestCancelOlderThanEarlyCancelLifeIsDropped(t *testing.T) {
	clock := testsupport.NewClock()
	searches := searchcancel.NewSearches(clock)

	searches.Cancel("query-10")
	clock.Advance(earlyCancelLife)
	ctx, done := searches.Start(t.Context(), "query-10")
	defer done()

	if isCanceled(ctx) {
		t.Fatal("a search started canceled by a cancel older than the early cancel life")
	}
}

func TestCancelBeyondTheLimitDropsTheOldestEarlyCancel(t *testing.T) {
	clock := testsupport.NewClock()
	searches := searchcancel.NewSearches(clock)

	searches.Cancel("query-first")
	for i := range earlyCancelLimit {
		clock.Advance(1)
		searches.Cancel(fmt.Sprintf("query-%d", i))
	}

	oldest, doneOldest := searches.Start(t.Context(), "query-first")
	defer doneOldest()
	newest, doneNewest := searches.Start(t.Context(), fmt.Sprintf("query-%d", earlyCancelLimit-1))
	defer doneNewest()

	if isCanceled(oldest) {
		t.Error("the oldest early cancel is kept beyond the limit")
	}
	if !isCanceled(newest) {
		t.Error("the newest early cancel is dropped")
	}
}

func TestStartWithoutIDCannotBeCanceled(t *testing.T) {
	searches := searchcancel.NewSearches(testsupport.NewClock())
	ctx, done := searches.Start(t.Context(), "")
	defer done()

	searches.Cancel("")

	if isCanceled(ctx) {
		t.Fatal("a search without an id was canceled")
	}
}
