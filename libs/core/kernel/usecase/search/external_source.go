package search

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// sourceTimeout bounds the wait for one external ranking of an interactive search.
const sourceTimeout = 2 * time.Second

type sourceAnswer struct {
	ranking ParticipantResults
	err     error
}

func (u Search) rankSource(ctx context.Context, v domain.Vault, source *externalSource, q Query) (*ParticipantResults, error) {
	sourceCtx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()
	answer := source.rank(sourceCtx, v, q)
	if err := sourceCtx.Err(); err != nil {
		answer.err = err
	}
	switch {
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case errors.Is(answer.err, context.DeadlineExceeded), errors.Is(answer.err, context.Canceled):
		u.errorHandler(fmt.Errorf("search source %d: %w", source.number, answer.err))
	case answer.err == nil:
		u.reportAnswered()
		return &answer.ranking, nil
	default:
		u.errorHandler(answer.err)
	}
	return nil, nil
}

// maxSourceCalls bounds outstanding calls to a registered source across searches.
const maxSourceCalls = 4

type externalSource struct {
	participant Participant
	number      int
	slots       chan struct{}
}

func newExternalSources(participants []Participant) []externalSource {
	sources := make([]externalSource, len(participants))
	for i, participant := range participants {
		sources[i] = externalSource{participant: participant, number: i + 1, slots: make(chan struct{}, maxSourceCalls)}
	}
	return sources
}

func (s *externalSource) rank(ctx context.Context, v domain.Vault, q Query) sourceAnswer {
	select {
	case s.slots <- struct{}{}:
	case <-ctx.Done():
		return sourceAnswer{}
	}
	done := make(chan sourceAnswer, 1)
	go func() {
		defer func() { <-s.slots }()
		if ctx.Err() != nil {
			return
		}
		ranking, err := s.participant.Rank(ctx, v, q)
		done <- sourceAnswer{ranking: ranking, err: err}
	}()
	select {
	case answer := <-done:
		return answer
	case <-ctx.Done():
		return sourceAnswer{}
	}
}
