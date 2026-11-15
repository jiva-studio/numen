package refresh

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/resultcache"
)

// cachedReranker is the reranker of one run. It reuses the score of a text the record scored, when the
// record was computed for the same query, rerank query and floor by the row that is first now, and sends
// the other texts to the reranker in one call. A call answered by another row than the one the reused
// scores came from is made again with every text. It keeps the scores of the last answer and the failure
// of the reranker.
type cachedReranker struct {
	reranker  ports.Reranker
	priority  connection.Priority
	old       *resultcache.Record
	queryHash string
	scores    map[string]float64
	failure   error
}

var _ ports.Reranker = (*cachedReranker)(nil)

func newCachedReranker(reranker ports.Reranker, priority connection.Priority, old *resultcache.Record, queryHash string) *cachedReranker {
	return &cachedReranker{
		reranker: reranker, priority: priority, old: old, queryHash: queryHash, scores: map[string]float64{}, failure: nil,
	}
}

func (c *cachedReranker) ListRerankers(ctx context.Context) ([]ports.RerankerID, error) {
	return c.reranker.ListRerankers(ctx)
}

func (c *cachedReranker) Rerank(ctx context.Context, request ports.RerankRequest) (ports.Reranked, error) {
	request.Priority = c.priority
	reused, row := c.getReusableScores(ctx, request)
	answer, err := c.rerankMissing(ctx, request, reused, row)
	if err == nil && reused != nil && !isSameID(answer.Reranker, row) {
		answer, err = c.rerankMissing(ctx, request, nil, row)
	}
	if err != nil {
		if !errors.Is(err, ports.ErrNoReranker) && ctx.Err() == nil {
			c.failure = err
		}
		return ports.Reranked{}, err
	}
	c.scores = map[string]float64{}
	for i, document := range request.Documents {
		c.scores[resultcache.HashText(document)] = answer.Scores[i]
	}
	return answer, nil
}

func (c *cachedReranker) rerankMissing(
	ctx context.Context, request ports.RerankRequest, reused map[string]float64, row ports.RerankerID,
) (ports.Reranked, error) {
	var missing []string
	for _, document := range request.Documents {
		if _, isReused := reused[resultcache.HashText(document)]; !isReused {
			missing = append(missing, document)
		}
	}
	fresh := map[string]float64{}
	if len(missing) > 0 {
		sent := request
		sent.Documents = missing
		answer, err := c.reranker.Rerank(ctx, sent)
		if err == nil && len(answer.Scores) != len(missing) {
			err = fmt.Errorf("it answered %d scores for %d passages", len(answer.Scores), len(missing))
		}
		if err != nil {
			return ports.Reranked{}, err
		}
		for i, document := range missing {
			fresh[resultcache.HashText(document)] = answer.Scores[i]
		}
		row = answer.Reranker
	}
	scores := make([]float64, len(request.Documents))
	for i, document := range request.Documents {
		hash := resultcache.HashText(document)
		if score, isFresh := fresh[hash]; isFresh {
			scores[i] = score
		} else {
			scores[i] = reused[hash]
		}
	}
	return ports.Reranked{Scores: scores, Reranker: row}, nil
}

func (c *cachedReranker) getReusableScores(ctx context.Context, request ports.RerankRequest) (map[string]float64, ports.RerankerID) {
	old := c.old
	if old == nil || old.Reranker == nil || old.QueryHash != c.queryHash ||
		old.RerankHash != resultcache.HashText(request.Query) || old.Floor != request.Floor {
		return nil, ports.RerankerID{}
	}
	listed, err := c.reranker.ListRerankers(ctx)
	if err != nil || len(listed) == 0 || !isSameRow(listed[0], *old.Reranker) {
		return nil, ports.RerankerID{}
	}
	return old.RerankScores, listed[0]
}

func isSameID(a, b ports.RerankerID) bool {
	return isSameRow(a, resultcache.Reranker{ConnectionID: b.ConnectionID, Options: b.Options})
}
