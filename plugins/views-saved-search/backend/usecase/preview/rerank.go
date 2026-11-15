package preview

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
)

// reranking is the candidates the reranker was given, in their order, with the reason when it did not
// order them.
type reranking struct {
	candidates []ranking.Candidate
	order      string
	warnings   []string
}

func (u *UseCase) rerank(ctx context.Context, in api.PreviewInput, compiled query.Query, kept []ranking.Candidate, limit int) (reranking, error) {
	kept = kept[:min(len(kept), u.config.RerankedPerResult*limit, u.config.MaxReranked)]
	fused := reranking{candidates: kept, order: api.OrderFused, warnings: nil}
	if len(kept) == 0 {
		return fused, nil
	}
	documents := make([]string, len(kept))
	for i, candidate := range kept {
		documents[i] = candidate.Text
	}
	scores, err := u.reranker.Rerank(ctx, getRerankQuery(in.Rerank, compiled.Leaves), "", documents)
	if err == nil && len(scores) != len(kept) {
		err = fmt.Errorf("it answered %d scores for %d passages", len(scores), len(kept))
	}
	switch {
	case err == nil:
	case errors.Is(err, ports.ErrNoReranker):
		return fused, nil
	case ctx.Err() != nil:
		return fused, ctx.Err()
	default:
		fused.warnings = []string{fmt.Sprintf("the reranking service did not answer (%v); the results are in fused order", err)}
		return fused, nil
	}
	reranked := make([]ranking.Candidate, len(kept))
	for i, candidate := range kept {
		score := scores[i]
		candidate.Score = &score
		reranked[i] = candidate
	}
	ranking.Sort(reranked)
	return reranking{candidates: reranked, order: api.OrderReranked, warnings: nil}, nil
}

func getRerankQuery(text string, leaves []query.Leaf) string {
	if strings.TrimSpace(text) != "" {
		return text
	}
	var texts []string
	for _, leaf := range leaves {
		if leaf.IsPositive {
			texts = append(texts, leaf.Text)
		}
	}
	return strings.Join(texts, "\n")
}
