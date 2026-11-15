package hostendpoint

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestHostSearchTextPassesTheLimitTheResultsPerSourceTheModeAndTheRawWords(t *testing.T) {
	search := &recordingSearch{}
	client := newSearchClient(t, newSearchCallers(search), "token-a")

	_, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{
		Query: "NEAR(heron reed, 3)", Limit: 300, ResultsPerSource: 10, Mode: v1.SearchMode_SEARCH_MODE_WORDS, IsRawWords: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(search.textQueries) != 1 {
		t.Fatalf("the search ran %d times", len(search.textQueries))
	}
	got := search.textQueries[0]
	if got.Query != "NEAR(heron reed, 3)" || got.Limit != 300 || got.ResultsPerSource != 10 || got.Mode != plugin.SearchModeWords || !got.IsRawWords {
		t.Errorf("the search received %+v, want the raw words query, 300 results, 10 per file, words mode", got)
	}
}

func TestHostSearchTextReadsEveryModeOfTheWire(t *testing.T) {
	for wire, want := range map[v1.SearchMode]plugin.SearchMode{
		v1.SearchMode_SEARCH_MODE_UNSPECIFIED: plugin.SearchModeHybrid,
		v1.SearchMode_SEARCH_MODE_HYBRID:      plugin.SearchModeHybrid,
		v1.SearchMode_SEARCH_MODE_WORDS:       plugin.SearchModeWords,
		v1.SearchMode_SEARCH_MODE_MEANING:     plugin.SearchModeMeaning,
	} {
		search := &recordingSearch{}
		client := newSearchClient(t, newSearchCallers(search), "token-a")

		if _, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron", Mode: wire})); err != nil {
			t.Fatalf("%v: %v", wire, err)
		}
		if len(search.textQueries) != 1 || search.textQueries[0].Mode != want {
			t.Errorf("%v reached the search as %+v, want mode %v", wire, search.textQueries, want)
		}
	}
}

func TestHostSearchTextRejectsModeNames(t *testing.T) {
	for _, mode := range []v1.SearchMode{v1.SearchMode_SEARCH_MODE_NAMES, v1.SearchMode(99)} {
		search := &recordingSearch{}
		client := newSearchClient(t, newSearchCallers(search), "token-a")

		_, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron", Mode: mode}))
		if readCode(t, err) != connect.CodeInvalidArgument {
			t.Errorf("mode %v got %v, want invalid argument", mode, err)
		}
		if len(search.textQueries) != 0 {
			t.Errorf("mode %v ran the search: %+v", mode, search.textQueries)
		}
	}
}

func TestHostSearchTextRejectsRawWordsWithAnotherModeThanWords(t *testing.T) {
	for _, mode := range []v1.SearchMode{v1.SearchMode_SEARCH_MODE_UNSPECIFIED, v1.SearchMode_SEARCH_MODE_HYBRID, v1.SearchMode_SEARCH_MODE_MEANING} {
		search := &recordingSearch{}
		client := newSearchClient(t, newSearchCallers(search), "token-a")

		_, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron", Mode: mode, IsRawWords: true}))
		if readCode(t, err) != connect.CodeInvalidArgument {
			t.Errorf("raw words in mode %v got %v, want invalid argument", mode, err)
		}
		if len(search.textQueries) != 0 {
			t.Errorf("raw words in mode %v ran the search: %+v", mode, search.textQueries)
		}
	}
}

func TestHostSearchTextAnswersInvalidWordsForARawQueryTheIndexRejects(t *testing.T) {
	client := newSearchClient(t, newSearchCallers(&recordingSearch{err: plugin.ErrInvalidWords}), "token-a")

	got, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{
		Query: `"heron`, Mode: v1.SearchMode_SEARCH_MODE_WORDS, IsRawWords: true,
	}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_WORDS {
		t.Errorf("returned %v (%v), want INVALID_WORDS", got.Msg, err)
	}
}

type scoredSearch struct {
	recordingSearch
}

func (s *scoredSearch) SearchText(context.Context, domain.Vault, plugin.SearchResultQuery) ([]domain.SearchResult, error) {
	similarity, wordScore := 0.6, 3.5
	return []domain.SearchResult{
		{ChunkID: "41", Path: "notes/Heron.md", Text: "herons fish", Start: 20, Length: 40, Similarity: &similarity, WordScore: &wordScore},
		{ChunkID: "42", Path: "notes/Reed.md", Text: "reeds sway", Start: 0, Length: 10},
	}, nil
}

func TestHostSearchTextWritesTheChunkIDTheSimilarityAndTheWordScoreOfAResult(t *testing.T) {
	client := newSearchClient(t, newSearchCallers(&scoredSearch{}), "token-a")

	got, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron"}))
	if err != nil || len(got.Msg.GetResults()) != 2 {
		t.Fatalf("returned %v (%v)", got.Msg, err)
	}
	scored, plain := got.Msg.GetResults()[0], got.Msg.GetResults()[1]
	if scored.GetChunkId() != "41" || scored.Similarity == nil || scored.GetSimilarity() != 0.6 || scored.WordScore == nil || scored.GetWordScore() != 3.5 {
		t.Errorf("the scored result is %v, want chunk 41, similarity 0.6 and word score 3.5", scored)
	}
	if plain.GetChunkId() != "42" || plain.Similarity != nil || plain.WordScore != nil {
		t.Errorf("the result with no scores is %v, want chunk 42 and no scores", plain)
	}
}
