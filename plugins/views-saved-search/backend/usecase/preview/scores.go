package preview

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/scoring"
)

func (u *UseCase) keepMatches(ctx context.Context, compiled query.Query, found []ranking.Candidate) ([]ranking.Candidate, float64, error) {
	scores, floor, err := u.scoreCandidates(ctx, compiled, found)
	if err != nil {
		return nil, 0, err
	}
	var kept []ranking.Candidate
	for _, candidate := range found {
		chunkScores, isScored := scores[candidate.ChunkID]
		if !isScored {
			continue
		}
		outcome := scoring.Evaluate(compiled.Root, compiled.Leaves, chunkScores, floor, candidate.Path)
		if outcome.IsMatch {
			candidate.Score = outcome.Score
			kept = append(kept, candidate)
		}
	}
	return kept, floor, nil
}

func (u *UseCase) scoreCandidates(ctx context.Context, compiled query.Query, found []ranking.Candidate) (map[string]scoring.Scores, float64, error) {
	conditions := newConditions(compiled.Leaves)
	scores := map[string]scoring.Scores{}
	var floor float64
	for start := 0; start < len(found); start += host.MaxScoredChunks {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		ids := getChunkIDs(found[start:min(start+host.MaxScoredChunks, len(found))])
		answer, err := u.search.ScoreChunks(ctx, host.ScoreQuery{ChunkIDs: ids, Conditions: conditions})
		if errors.Is(err, host.ErrInvalidWords) {
			return nil, 0, u.findRefusedWords(ctx, compiled, ids, err)
		}
		if err != nil {
			return nil, 0, convertHostError(compiled, -1, err, "score the passages")
		}
		floor = answer.Floor
		for _, chunk := range answer.Chunks {
			scores[chunk.ChunkID] = chunk.Conditions
		}
	}
	return scores, floor, nil
}

func getChunkIDs(batch []ranking.Candidate) []string {
	ids := make([]string, len(batch))
	for i, candidate := range batch {
		ids[i] = candidate.ChunkID
	}
	return ids
}

func newConditions(leaves []query.Leaf) []host.ScoreCondition {
	conditions := make([]host.ScoreCondition, len(leaves))
	for i, leaf := range leaves {
		if leaf.IsMeaning {
			conditions[i] = host.ScoreCondition{Meaning: leaf.Text}
		} else {
			conditions[i] = host.ScoreCondition{Words: leaf.Text}
		}
	}
	return conditions
}

func (u *UseCase) findRefusedWords(ctx context.Context, compiled query.Query, ids []string, refused error) error {
	for i, leaf := range compiled.Leaves {
		if leaf.IsMeaning {
			continue
		}
		_, err := u.search.ScoreChunks(ctx, host.ScoreQuery{ChunkIDs: ids, Conditions: []host.ScoreCondition{{Words: leaf.Text}}})
		if errors.Is(err, host.ErrInvalidWords) {
			return convertHostError(compiled, i, err, "score the passages")
		}
		if err != nil {
			return convertHostError(compiled, -1, err, "score the passages")
		}
	}
	return convertHostError(compiled, -1, refused, "score the passages")
}
