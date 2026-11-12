package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
)

// nearestPlace is the index of the host as a search source asks it: it records the queries and answers
// the chunks it holds.
type nearestPlace struct {
	asked   []host.NearestQuery
	matches []host.NearestMatch
}

func (p *nearestPlace) FindNearestPassages(_ context.Context, query host.NearestQuery) ([]host.NearestMatch, error) {
	p.asked = append(p.asked, query)
	return p.matches, nil
}

func noPlaces(string) (host.PlaceSearch, error) { return nil, errors.New("no host") }

func TestAModelThatHasNoProfileIsAFailedPrecondition(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	h := newHandler(newServer(nil), noPlaces)

	_, describeErr := h.Describe(t.Context())
	_, indexErr := h.IndexTexts(t.Context(), searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "text"}})

	if rpcerror.GetCode(describeErr) != rpcerror.FailedPrecondition || rpcerror.GetCode(indexErr) != rpcerror.FailedPrecondition {
		t.Fatalf("Describe error %v and IndexTexts error %v, want FailedPrecondition", describeErr, indexErr)
	}
}

func TestTheHandlerDescribesAndIndexesWithTheModelTheEnvironmentNames(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	h := newHandler(newServer(nil), noPlaces)

	described, err := h.Describe(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := h.IndexTexts(t.Context(), searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "boat"}, {ChunkID: "2", Text: "ferry"}})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(described.Recipe, "mock|") || len(vectors) != 2 || len(vectors[0]) != int(described.Width) {
		t.Fatalf("description %+v with %d vectors", described, len(vectors))
	}
}

func TestASearchAsksThePlaceWithTheVectorOfTheQueryAndAnswersItsChunks(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	place := &nearestPlace{matches: []host.NearestMatch{{ChunkID: "41", Similarity: 0.8}, {ChunkID: "7", Similarity: 0.5}}}
	var vaults []string
	h := newHandler(newServer(nil), func(vaultID string) (host.PlaceSearch, error) {
		vaults = append(vaults, vaultID)
		return place, nil
	})

	found, err := h.SearchPassages(t.Context(), searchsource.Place{VaultID: "01PHYSICS", Recipe: "mock|r"},
		searchsource.Query{Text: "entropy", Limit: 5, Kinds: []string{"file"}, PathPrefix: "books/", Floor: 0.4})
	if err != nil {
		t.Fatal(err)
	}

	want := []searchsource.Match{{ChunkID: "41", Score: 0.8}, {ChunkID: "7", Score: 0.5}}
	if !slices.Equal(found, want) || !slices.Equal(vaults, []string{"01PHYSICS"}) {
		t.Fatalf("found %v in vaults %v, want %v in 01PHYSICS", found, vaults, want)
	}
	asked := place.asked[0]
	if asked.Recipe != "mock|r" || asked.Limit != 5 || asked.Floor != 0.4 || asked.PathPrefix != "books/" || len(asked.Kinds) != 1 || len(asked.Vector) == 0 {
		t.Errorf("the place was asked %+v", asked)
	}
}

func TestTheCoverageSaysTheIndexIsNotBuiltUntilAChunkHasAVector(t *testing.T) {
	h := newHandler(newServer(nil), noPlaces)

	empty, _ := h.DescribeCoverage(t.Context(), searchsource.Place{}, searchsource.Counts{Chunks: 12})
	started, _ := h.DescribeCoverage(t.Context(), searchsource.Place{}, searchsource.Counts{Chunks: 12, Indexed: 1})

	if empty != "Semantic index not built yet" || started != "" {
		t.Errorf("coverage %q before and %q after the first vector", empty, started)
	}
}
