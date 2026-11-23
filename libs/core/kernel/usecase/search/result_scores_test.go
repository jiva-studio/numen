package search_test

import (
	"context"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func newScore(value float64) *float64 { return new(value) }

func findResult(t *testing.T, found []domain.SearchResult, path string) domain.SearchResult {
	t.Helper()
	i := slices.IndexFunc(found, func(p domain.SearchResult) bool { return p.Path == path })
	if i < 0 {
		t.Fatalf("no result of %s among %v", path, sources(found))
	}
	return found[i]
}

func (c corpus) addVault(t *testing.T, notes map[string]string) domain.Vault {
	t.Helper()
	v := vaulttest.NewVault(t, notes)
	scan := indexing.NewScan(vaultreader.Readers{}, c.db.Vaults(), c.db.Notes(), c.db.NoteQueries(), c.db.Maintenance(), nil)
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (c corpus) vectorizePath(t *testing.T, v domain.Vault, path string, direction func(record chunk.Record) []float32) {
	t.Helper()
	unembedded, err := c.db.ChunkVectors().GetUnembeddedChunks(t.Context(), v.ID, getModel().Recipe(), chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var vectors []chunk.Vector
	for _, p := range unembedded {
		if p.Path != path {
			continue
		}
		raw, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		one := direction(p)
		vectors = append(vectors, chunk.Vector{Chunk: p.Chunk, Hash: raw, Recipe: getModel().Recipe(), Value: precise(one), Coarse: vector.Bits(one)})
	}
	if len(vectors) == 0 {
		t.Fatalf("no small chunk of %s to embed", path)
	}
	if err := c.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

func TestASearchResultHasTheSimilarityOfTheMeaningSearchAndNoneWithoutIt(t *testing.T) {
	c := newCorpus(t)
	c.cut(t, c.first, "notes/Entropy.md", "a measure of disorder, shared")
	c.cut(t, c.first, "notes/Heat.md", "energy in transit, shared")
	c.vectorizePath(t, c.first, "notes/Entropy.md", func(chunk.Record) []float32 { return newVector(+1) })

	found := c.searchIn(t, c.first, "shared", oneWay{direction: newVector(+1)})

	want := vector.SimilarityBytes(newVector(+1), precise(newVector(+1)))
	if entropy := findResult(t, found, "notes/Entropy.md"); entropy.Similarity == nil || *entropy.Similarity != want {
		t.Errorf("the result the meaning search found has the similarity %v, want %v", entropy.Similarity, want)
	}
	if heat := findResult(t, found, "notes/Heat.md"); heat.Similarity != nil {
		t.Errorf("a result only the words found has the similarity %v, want none", *heat.Similarity)
	}
}

func TestASearchResultHasAWordScoreOnlyWhereTheWordsMatched(t *testing.T) {
	c := newCorpus(t)
	c.vectorize(t, c.first, newVector(+1))

	byWords := c.searchIn(t, c.first, "disorder", nil)
	if len(byWords) != 1 {
		t.Fatalf("the words found %v, want the one note", sources(byWords))
	}
	if byWords[0].WordScore == nil || *byWords[0].WordScore <= 0 {
		t.Errorf("a result the words matched has the word score %v, want a positive one", byWords[0].WordScore)
	}
	if byWords[0].Similarity != nil {
		t.Errorf("a search with no model returned the similarity %v", *byWords[0].Similarity)
	}

	byMeaning := c.searchIn(t, c.first, "zzqqxx", oneWay{direction: newVector(+1)})
	if len(byMeaning) == 0 {
		t.Fatal("the meaning search found nothing")
	}
	for _, p := range byMeaning {
		if p.WordScore != nil {
			t.Errorf("%s was found only by meaning and has the word score %v", p.Path, *p.WordScore)
		}
		if p.Similarity == nil {
			t.Errorf("%s was found by meaning and has no similarity", p.Path)
		}
	}
}

type fixedWords struct{ found []domain.SearchResult }

func (f fixedWords) FindLexical(context.Context, domain.VaultID, sourcedomain.LexicalQuery) ([]domain.SearchResult, error) {
	return slices.Clone(f.found), nil
}

func (fixedWords) GetNamedSections(context.Context, domain.VaultID, sourcedomain.LexicalQuery) ([]domain.SearchResult, error) {
	return nil, nil
}

type fixedRanking struct{ found []domain.SearchResult }

func (f fixedRanking) Rank(context.Context, domain.Vault, search.Query) (search.Ranking, error) {
	return search.Ranking{Results: slices.Clone(f.found)}, nil
}

func TestASearchKeepsTheBestSimilarityAndWordScoreOfEveryHitInOnePlace(t *testing.T) {
	c := newCorpus(t)
	const path = "notes/Entropy.md"
	length := len(getFirstNotes()[path])
	place := func(id domain.ChunkID, start, size int) domain.SearchResult {
		return domain.SearchResult{ChunkID: id, Path: path, Kind: domain.KindNote, Start: start, Length: size}
	}
	withWords := func(p domain.SearchResult, score float64) domain.SearchResult {
		p.WordScore = newScore(score)
		return p
	}
	withSimilarity := func(p domain.SearchResult, score float64) domain.SearchResult {
		p.Similarity = newScore(score)
		return p
	}

	words := fixedWords{found: []domain.SearchResult{
		withWords(place("1", 0, 20), 2),
		withWords(place("2", 0, 20), 5),
		withWords(place("9", 20, length-20), 1),
	}}
	meaning := fixedRanking{found: []domain.SearchResult{
		withSimilarity(place("1", 0, 20), 0.4),
		withSimilarity(place("3", 0, 20), 0.7),
	}}
	finds := search.New(words, vaultreader.Readers{}, nil, []search.Participant{meaning}, nil, nil)

	found, err := finds.Execute(t.Context(), c.first, "disorder", search.Parameters{Each: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 || found[0].ChunkID != "1" || found[1].ChunkID != "9" {
		t.Fatalf("found %+v, want the place of chunk 1 and then the place of chunk 9", found)
	}
	if got := found[0]; got.Similarity == nil || *got.Similarity != 0.7 || got.WordScore == nil || *got.WordScore != 5 {
		t.Errorf("the first place has the similarity %v and the word score %v, want the best of its hits, 0.7 and 5", got.Similarity, got.WordScore)
	}
	if got := found[1]; got.Similarity != nil || got.WordScore == nil || *got.WordScore != 1 {
		t.Errorf("the second place has the similarity %v and the word score %v, want none and 1", got.Similarity, got.WordScore)
	}
}
