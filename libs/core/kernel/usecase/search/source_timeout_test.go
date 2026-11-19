package search

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type rankFunc func(context.Context) (Ranking, error)

func (f rankFunc) Rank(ctx context.Context, _ domain.Vault, _ Query) (Ranking, error) {
	return f(ctx)
}

func TestRankingReturnsHealthyParticipantsWhenASourceHangs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		sourceContext := make(chan context.Context, 1)
		diagnostics := make(chan error, 2)
		answers := make(chan struct{}, 2)
		healthy := Ranking{Results: []domain.SearchResult{{Path: "answer.md"}}}
		u := Search{
			words: rankFunc(func(ctx context.Context) (Ranking, error) {
				if _, bounded := ctx.Deadline(); bounded {
					t.Error("built-in search received a deadline")
				}
				return healthy, nil
			}),
			sources: []Participant{rankFunc(func(ctx context.Context) (Ranking, error) {
				sourceContext <- ctx
				<-release
				return Ranking{}, nil
			})},
			errorHandler:   func(err error) { diagnostics <- err },
			reportAnswered: func() { answers <- struct{}{} },
		}

		done := make(chan []Ranking, 1)
		go func() {
			got, err := u.rankAll(t.Context(), domain.Vault{}, "word", Parameters{Lexical: 1, Sources: 1})
			if err != nil {
				t.Errorf("search failed: %v", err)
			}
			done <- got
		}()

		ctx := <-sourceContext
		select {
		case got := <-done:
			if len(got) != 1 || len(got[0].Results) != 1 || got[0].Results[0].Path != "answer.md" {
				t.Fatalf("rankings = %+v", got)
			}
		case <-time.After(time.Minute):
			t.Fatal("healthy participant answered, but search is still waiting for the blocked source")
		}
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("source context error = %v", ctx.Err())
		}
		if len(diagnostics) != 1 || !errors.Is(<-diagnostics, context.DeadlineExceeded) {
			t.Fatal("missing source timeout diagnostic")
		}
		if len(answers) != 0 {
			t.Fatal("timed-out source counted as answered")
		}
	})
}

func TestARequestDeadlineStopsSearchWhileAnExternalSourceIsBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		release := make(chan struct{})
		defer close(release)
		u := Search{
			sources: []Participant{rankFunc(func(context.Context) (Ranking, error) {
				<-release
				return Ranking{}, nil
			})},
			errorHandler:   func(err error) { t.Errorf("source failure: %v", err) },
			reportAnswered: func() { t.Error("source counted as answered") },
		}
		done := make(chan error, 1)
		go func() {
			_, err := u.rankAll(ctx, domain.Vault{}, "word", Parameters{Sources: 1})
			done <- err
		}()
		select {
		case err := <-done:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("request error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("search is still waiting after the request deadline")
		}
	})
}
