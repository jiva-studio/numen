package hosttest

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// ScoreChunks scores the results SetIndexedResults set, found by their ChunkID, with the limits and the
// errors of the host. A words condition matches a result whose text contains every word of the query,
// with the FTS5 operators left out, which is a limit of the fake. A meaning condition compares the
// embedder vectors of the condition and of the text, and matches at or above the floor of the vault.
func (s *Search) ScoreChunks(_ context.Context, query host.ScoreQuery) (host.ChunkScores, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return host.ChunkScores{}, err
	}
	hasMeaning, err := checkScoreQuery(query)
	if err != nil {
		return host.ChunkScores{}, err
	}
	var floor float64
	if hasMeaning {
		if s.all.model == nil {
			return host.ChunkScores{}, host.ErrNoModel
		}
		if s.all.embed == nil {
			return host.ChunkScores{}, fmt.Errorf("%w: the fake has no embedder", host.ErrEmbeddingFailed)
		}
		floor = s.getFloorLocked(one, s.all.model.Recipe).Floor
	}
	out := host.ChunkScores{Floor: floor}
	seen := map[string]bool{}
	for _, id := range query.ChunkIDs {
		result, found := findIndexedResult(one.indexed, id)
		if !found || seen[id] {
			continue
		}
		seen[id] = true
		scores := make([]host.ConditionScore, len(query.Conditions))
		for i, condition := range query.Conditions {
			scores[i] = s.scoreLocked(result.Text, condition, floor)
		}
		out.Chunks = append(out.Chunks, host.ChunkScore{ChunkID: id, Conditions: scores})
	}
	return out, nil
}

func checkScoreQuery(query host.ScoreQuery) (bool, error) {
	if len(query.ChunkIDs) > host.MaxScoredChunks || len(query.Conditions) > host.MaxScoredConditions {
		return false, host.ErrLimitExceeded
	}
	hasMeaning := false
	for _, condition := range query.Conditions {
		if len(condition.Meaning) > host.MaxConditionBytes || len(condition.Words) > host.MaxConditionBytes {
			return false, host.ErrLimitExceeded
		}
		if (condition.Meaning != "") == (condition.Words != "") || strings.TrimSpace(condition.Meaning+condition.Words) == "" {
			return false, host.ErrInvalidCondition
		}
		hasMeaning = hasMeaning || condition.Meaning != ""
	}
	return hasMeaning, nil
}

func findIndexedResult(indexed []host.SearchResult, id string) (host.SearchResult, bool) {
	for _, result := range indexed {
		if result.ChunkID == id {
			return result, true
		}
	}
	return host.SearchResult{}, false
}

func (s *Search) scoreLocked(text string, condition host.ScoreCondition, floor float64) host.ConditionScore {
	if condition.Words != "" {
		return host.ConditionScore{IsMatch: hasEveryWord(text, getQueryWords(condition.Words, true))}
	}
	query, chunk := s.all.embed(condition.Meaning), s.all.embed(text)
	if query == nil || chunk == nil {
		return host.ConditionScore{}
	}
	similarity := s.compareLocked(query, chunk)
	return host.ConditionScore{Similarity: &similarity, IsMatch: similarity >= floor}
}
