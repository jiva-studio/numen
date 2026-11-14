package preview

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
)

// reranking is the candidates the reranker was given, in their order, with the reason when it did not
// order them, and the row that ordered them.
type reranking struct {
	candidates []ranking.Candidate
	order      string
	warnings   []string
	reranker   *ports.RerankerID
}

type rerankInput struct {
	text  string
	kept  []ranking.Candidate
	limit int
	floor float64
}

func (u *UseCase) rerank(ctx context.Context, in rerankInput) (reranking, error) {
	kept := in.kept[:min(len(in.kept), u.config.RerankedPerResult*in.limit, u.config.MaxReranked)]
	fused := reranking{candidates: kept, order: api.OrderFused, warnings: nil, reranker: nil}
	if len(kept) == 0 {
		return fused, nil
	}
	documents := make([]string, len(kept))
	for i, candidate := range kept {
		documents[i] = candidate.Text
	}
	answer, err := u.reranker.Rerank(ctx, ports.RerankRequest{
		Query: in.text, Instruction: "", Documents: documents, Priority: connection.PriorityInteractive, Floor: in.floor,
	})
	if err == nil && len(answer.Scores) != len(kept) {
		err = fmt.Errorf("it answered %d scores for %d passages", len(answer.Scores), len(kept))
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
		score := answer.Scores[i]
		candidate.Score = &score
		reranked[i] = candidate
	}
	ranking.Sort(reranked)
	row := answer.Reranker
	return reranking{candidates: reranked, order: api.OrderReranked, warnings: nil, reranker: &row}, nil
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
