package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type scorer struct {
	places  ChunkPlaces
	matcher WordMatcher
	words   WordQueries
	req     *searchquery.Evaluation
}

func (s scorer) score(
	ctx context.Context, candidates []domain.SearchResult, texts map[string]fileText, vectors map[int][]float32,
) ([][]searchquery.ConditionScore, error) {
	scores := make([][]searchquery.ConditionScore, len(candidates))
	for i := range scores {
		scores[i] = make([]searchquery.ConditionScore, len(s.req.Query.Conditions))
	}
	if len(candidates) == 0 || len(s.req.Query.Conditions) == 0 {
		return scores, nil
	}
	smalls, err := s.listSmalls(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if err := s.scoreMeaning(ctx, vectors, smalls, scores); err != nil {
		return nil, err
	}
	return scores, s.scoreWords(ctx, candidates, smalls, texts, scores)
}

func (s scorer) listSmalls(ctx context.Context, candidates []domain.SearchResult) ([][]domain.ChunkID, error) {
	ids := make([]domain.ChunkID, len(candidates))
	index := make(map[domain.ChunkID]int, len(candidates))
	for i, c := range candidates {
		ids[i] = c.ChunkID
		index[c.ChunkID] = i
	}
	listed, err := s.places.ListSmallChunks(ctx, s.req.Vault.ID, ids)
	if err != nil {
		return nil, err
	}
	smalls := make([][]domain.ChunkID, len(candidates))
	for i, c := range candidates {
		smalls[i] = []domain.ChunkID{c.ChunkID}
	}
	for _, one := range listed {
		if i, found := index[one.Chunk]; found && one.Small != candidates[i].ChunkID {
			smalls[i] = append(smalls[i], one.Small)
		}
	}
	return smalls, nil
}

func (s scorer) scoreMeaning(ctx context.Context, queries map[int][]float32, smalls [][]domain.ChunkID, scores [][]searchquery.ConditionScore) error {
	if len(queries) == 0 {
		return nil
	}
	var all []domain.ChunkID
	for _, list := range smalls {
		all = append(all, list...)
	}
	stored, err := s.places.ListStoredVectors(ctx, source.StoredVectorQuery{Recipe: s.req.Recipe, Chunks: all})
	if err != nil {
		return err
	}
	vectors := make(map[domain.ChunkID][]byte, len(stored))
	for _, one := range stored {
		vectors[one.Chunk] = one.Vector
	}
	for ci, query := range queries {
		for i, list := range smalls {
			scores[i][ci].Similarity = getBestSimilarity(query, list, vectors)
		}
	}
	return nil
}

func getBestSimilarity(query []float32, chunks []domain.ChunkID, vectors map[domain.ChunkID][]byte) *float64 {
	var best *float64
	for _, id := range chunks {
		stored, found := vectors[id]
		if !found || len(stored) != len(query) {
			continue
		}
		if similarity := vector.ComputeSimilarityBytes(query, stored); best == nil || similarity > *best {
			best = &similarity
		}
	}
	return best
}

func (s scorer) scoreWords(
	ctx context.Context, candidates []domain.SearchResult, smalls [][]domain.ChunkID, texts map[string]fileText,
	scores [][]searchquery.ConditionScore,
) error {
	chunkTexts := make([]string, len(candidates))
	for i, c := range candidates {
		chunkTexts[i] = getSpan(texts[c.Path].prose, c.Start, c.Length)
	}
	var all []domain.ChunkID
	for _, list := range smalls {
		all = append(all, list...)
	}
	for ci, c := range s.req.Query.Conditions {
		if c.IsMeaning {
			continue
		}
		matched, err := s.matcher.MatchWords(ctx, c.Text, chunkTexts)
		if err != nil {
			return getWordsError(s.req, ci, err)
		}
		wordScores, err := s.words.GetWordScores(ctx, s.req.Vault.ID, c.Text, all)
		if err != nil {
			return getWordsError(s.req, ci, err)
		}
		for i := range candidates {
			scores[i][ci].IsMatch = i < len(matched) && matched[i]
			scores[i][ci].WordScore = getBestWordScore(smalls[i], wordScores)
		}
	}
	return nil
}

func getBestWordScore(chunks []domain.ChunkID, scores map[domain.ChunkID]float64) *float64 {
	var best *float64
	for _, id := range chunks {
		if score, found := scores[id]; found && (best == nil || score > *best) {
			best = &score
		}
	}
	return best
}
