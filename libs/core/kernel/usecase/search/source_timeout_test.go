package search

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type rankFunc func(context.Context) (ParticipantResults, error)

func (f rankFunc) Rank(ctx context.Context, _ domain.Vault, _ Query) (ParticipantResults, error) {
	return f(ctx)
}

func TestRankingReturnsHealthyParticipantsWhenASourceHangs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		sourceContext := make(chan context.Context, 1)
		diagnostics := make(chan error, 2)
		answers := make(chan struct{}, 2)
		healthy := ParticipantResults{Results: []domain.SearchResult{{Path: "answer.md"}}}
		u := Search{
			words: rankFunc(func(ctx context.Context) (ParticipantResults, error) {
				if _, bounded := ctx.Deadline(); bounded {
					t.Error("built-in search received a deadline")
				}
				return healthy, nil
			}),
			sources: newExternalSources([]Participant{rankFunc(func(ctx context.Context) (ParticipantResults, error) {
				sourceContext <- ctx
				<-release
				return ParticipantResults{}, nil
			})}),
			errorHandler:   func(err error) { diagnostics <- err },
			reportAnswered: func() { answers <- struct{}{} },
		}

		done := make(chan []ParticipantResults, 1)
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
			sources: newExternalSources([]Participant{rankFunc(func(context.Context) (ParticipantResults, error) {
				<-release
				return ParticipantResults{}, nil
			})}),
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

func TestRepeatedSearchesBoundOutstandingCallsAndResumeWhenTheSourceRecovers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		started := make(chan struct{}, 12)
		plugin := rankFunc(func(ctx context.Context) (ParticipantResults, error) {
			started <- struct{}{}
			<-release
			return ParticipantResults{Results: []domain.SearchResult{{Path: "plugin.md"}}}, nil
		})
		diagnostics := make(chan error, 12)
		u := New(nil, nil, nil, []Participant{plugin}, func(err error) { diagnostics <- err }, nil)
		u.words = rankFunc(func(context.Context) (ParticipantResults, error) {
			return ParticipantResults{Results: []domain.SearchResult{{Path: "healthy.md"}}}, nil
		})
		params := Parameters{Lexical: 1, Sources: 1}
		for range 10 {
			got, err := u.rankAll(t.Context(), domain.Vault{}, "word", params)
			if err != nil || len(got) != 1 || got[0].Results[0].Path != "healthy.md" {
				t.Errorf("healthy ranking while source is blocked: %v, %v", got, err)
			}
		}
		close(release)
		synctest.Wait()
		if count := len(started); count > 4 {
			t.Errorf("%d source calls still running after ten timed-out searches, want at most four", count)
		}
		if count := len(diagnostics); count != 10 {
			t.Errorf("source failures reported = %d, want ten", count)
		}
		got, err := u.rankAll(t.Context(), domain.Vault{}, "word", params)
		if err != nil || len(got) != 2 || got[1].Results[0].Path != "plugin.md" {
			t.Fatalf("rankings after recovery: %v, %v", got, err)
		}
	})
}

func TestAShorterSourceDeadlinePreservesTheHealthyRanking(t *testing.T) {
	source := rankFunc(func(context.Context) (ParticipantResults, error) {
		return ParticipantResults{}, context.DeadlineExceeded
	})
	var reported error
	u := New(nil, nil, nil, []Participant{source}, func(err error) { reported = err }, nil)
	u.words = rankFunc(func(context.Context) (ParticipantResults, error) {
		return ParticipantResults{Results: []domain.SearchResult{{Path: "words.md"}}}, nil
	})
	got, err := u.rankAll(t.Context(), domain.Vault{}, "word", Parameters{Lexical: 1, Sources: 1})
	if err != nil || len(got) != 1 || got[0].Results[0].Path != "words.md" {
		t.Fatalf("rankings after source deadline: %v, %v", got, err)
	}
	if !errors.Is(reported, context.DeadlineExceeded) {
		t.Fatalf("source deadline diagnostic = %v", reported)
	}
}

func TestCallerCancellationStopsAnActiveSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	source := rankFunc(func(ctx context.Context) (ParticipantResults, error) {
		close(started)
		<-ctx.Done()
		return ParticipantResults{}, ctx.Err()
	})
	u := New(nil, nil, nil, []Participant{source}, func(err error) {
		t.Errorf("caller cancellation reported as source failure: %v", err)
	}, nil)
	done := make(chan error, 1)
	go func() {
		_, err := u.rankAll(ctx, domain.Vault{}, "word", Parameters{Sources: 1})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("search returned %v", err)
	}
}
