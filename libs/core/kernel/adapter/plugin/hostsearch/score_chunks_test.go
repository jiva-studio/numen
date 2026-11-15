package hostsearch

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// onePlace is an index that holds chunk c1 of notes/Heron.md, with one small chunk s1 on the first axis.
type onePlace struct{}

func (onePlace) GetEnclosingResults(_ context.Context, _ domain.VaultID, chunks []domain.ChunkID) ([]domain.SearchResult, error) {
	var out []domain.SearchResult
	for _, id := range chunks {
		if id == "c1" {
			out = append(out, domain.SearchResult{ChunkID: id, Path: "notes/Heron.md", Kind: domain.KindNote, Start: 0, Length: 15})
		}
	}
	return out, nil
}

func (onePlace) ListSmallChunks(_ context.Context, _ domain.VaultID, chunks []domain.ChunkID) ([]sourcedomain.SmallChunk, error) {
	var out []sourcedomain.SmallChunk
	for _, id := range chunks {
		out = append(out, sourcedomain.SmallChunk{Chunk: id, Small: "s1"})
	}
	return out, nil
}

func (onePlace) ListStoredVectors(context.Context, sourcedomain.StoredVectorQuery) ([]sourcedomain.StoredVector, error) {
	return []sourcedomain.StoredVector{{Chunk: "s1", Vector: []byte{127, 0, 0, 0}}}, nil
}

// fixedMatcher matches every text, or fails with its error.
type fixedMatcher struct{ err error }

func (m fixedMatcher) MatchWords(_ context.Context, _ string, texts []string) ([]bool, error) {
	if m.err != nil {
		return nil, m.err
	}
	matched := make([]bool, len(texts))
	for i := range matched {
		matched[i] = true
	}
	return matched, nil
}

func newScoreSearch(t *testing.T, words fixedMatcher, withModel bool) (Scoring, domain.Vault) {
	t.Helper()
	v := vaultfixture.NewVault(t, map[string]string{"notes/Heron.md": "the gray heron\n"})
	f := newFixture()
	deps := search.ScoreChunksDeps{Places: onePlace{}, Words: words, Readers: vaultreader.Readers{}, Floors: f.search.Policy}
	if withModel {
		deps.Source = f.embedder
	} else {
		f.search.Source = nil
	}
	return Scoring{Search: f.search, Scores: search.NewScoreChunks(deps)}, v
}

func TestScoreChunksReturnsTheScoresAndTheFloorOfTheVault(t *testing.T) {
	s, v := newScoreSearch(t, fixedMatcher{}, true)

	got, floor, err := s.ScoreChunks(t.Context(), v, plugin.ScoreQuery{
		ChunkIDs:   []domain.ChunkID{"c1", "gone"},
		Conditions: []plugin.ScoreCondition{{Meaning: "herons"}, {Words: "heron"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if floor != 0.4 {
		t.Errorf("the floor is %v, want the hint of the model, 0.4", floor)
	}
	if len(got) != 1 || got[0].ChunkID != "c1" || len(got[0].Conditions) != 2 {
		t.Fatalf("scored %+v, want chunk c1 with two scores", got)
	}
	meaning, words := got[0].Conditions[0], got[0].Conditions[1]
	if meaning.Similarity == nil || *meaning.Similarity < 0.99 || !meaning.IsMatch {
		t.Errorf("the meaning score is %+v, want about 1 and a match", meaning)
	}
	if words.Similarity != nil || !words.IsMatch {
		t.Errorf("the words score is %+v, want a match with no similarity", words)
	}
}

func TestScoreChunksReturnsNoFloorWithoutAMeaningCondition(t *testing.T) {
	s, v := newScoreSearch(t, fixedMatcher{}, false)

	got, floor, err := s.ScoreChunks(t.Context(), v, plugin.ScoreQuery{
		ChunkIDs: []domain.ChunkID{"c1"}, Conditions: []plugin.ScoreCondition{{Words: "heron"}},
	})
	if err != nil || floor != 0 || len(got) != 1 || !got[0].Conditions[0].IsMatch {
		t.Errorf("scored %+v with the floor %v, %v, want a match and no floor", got, floor, err)
	}
}

func TestScoreChunksReturnsErrNoModelForAMeaningConditionWithoutAModel(t *testing.T) {
	s, v := newScoreSearch(t, fixedMatcher{}, false)

	_, _, err := s.ScoreChunks(t.Context(), v, plugin.ScoreQuery{
		ChunkIDs: []domain.ChunkID{"c1"}, Conditions: []plugin.ScoreCondition{{Meaning: "herons"}},
	})
	if !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("a meaning condition with no model returned %v, want ErrNoModel", err)
	}
}

func TestScoreChunksReturnsPluginErrInvalidWordsForAQueryTheMatcherRejects(t *testing.T) {
	s, v := newScoreSearch(t, fixedMatcher{err: sourcedomain.ErrInvalidWords}, true)

	_, _, err := s.ScoreChunks(t.Context(), v, plugin.ScoreQuery{
		ChunkIDs: []domain.ChunkID{"c1"}, Conditions: []plugin.ScoreCondition{{Words: `"heron`}},
	})
	if !errors.Is(err, plugin.ErrInvalidWords) {
		t.Errorf("a query the matcher rejects returned %v, want plugin.ErrInvalidWords", err)
	}
}
