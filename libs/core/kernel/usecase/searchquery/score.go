package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// score scores every candidate against every condition.
func (p *plan) score(ctx context.Context, candidates []domain.SearchResult, texts map[string]fileText) ([][]searchquery.ConditionScore, error) {
	scores := make([][]searchquery.ConditionScore, len(candidates))
	for i := range scores {
		scores[i] = make([]searchquery.ConditionScore, len(p.query.Conditions))
	}
	if len(candidates) == 0 || len(p.query.Conditions) == 0 {
		return scores, nil
	}
	smalls, err := p.listSmalls(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if err := p.scoreMeaning(ctx, smalls, scores); err != nil {
		return nil, err
	}
	return scores, p.scoreWords(ctx, candidates, smalls, texts, scores)
}

// listSmalls returns the chunks inside each candidate, the candidate's own chunk among them.
func (p *plan) listSmalls(ctx context.Context, candidates []domain.SearchResult) ([][]domain.ChunkID, error) {
	ids := make([]domain.ChunkID, len(candidates))
	index := make(map[domain.ChunkID]int, len(candidates))
	for i, c := range candidates {
		ids[i] = c.ChunkID
		index[c.ChunkID] = i
	}
	listed, err := p.u.places.ListSmallChunks(ctx, p.v.ID, ids)
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

func (p *plan) scoreMeaning(ctx context.Context, smalls [][]domain.ChunkID, scores [][]searchquery.ConditionScore) error {
	if len(p.vectors) == 0 {
		return nil
	}
	var all []domain.ChunkID
	for _, list := range smalls {
		all = append(all, list...)
	}
	stored, err := p.u.places.ListStoredVectors(ctx, source.StoredVectorQuery{Recipe: p.recipe, Chunks: all})
	if err != nil {
		return err
	}
	vectors := make(map[domain.ChunkID][]byte, len(stored))
	for _, one := range stored {
		vectors[one.Chunk] = one.Vector
	}
	for ci, query := range p.vectors {
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
		if similarity := vector.SimilarityBytes(query, stored); best == nil || similarity > *best {
			best = &similarity
		}
	}
	return best
}

func (p *plan) scoreWords(
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
	for ci, c := range p.query.Conditions {
		if c.IsMeaning {
			continue
		}
		matched, err := p.u.matcher.MatchWords(ctx, c.Text, chunkTexts)
		if err != nil {
			return p.getWordsError(ci, err)
		}
		wordScores, err := p.u.words.GetWordScores(ctx, p.v.ID, c.Text, all)
		if err != nil {
			return p.getWordsError(ci, err)
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
