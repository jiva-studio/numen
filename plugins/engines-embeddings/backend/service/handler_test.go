package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/sourcehealth"
)

// nearestPlace is the index of the host as a search source asks it: it records the queries and answers
// the chunks it holds.
type nearestPlace struct {
	asked   []host.NearestQuery
	matches []host.NearestMatch
}

func (p *nearestPlace) FindNearestChunks(_ context.Context, query host.NearestQuery) ([]host.NearestMatch, error) {
	p.asked = append(p.asked, query)
	return p.matches, nil
}

func noPlaces(string) (host.PlaceSearch, error) { return nil, errors.New("no host") }

func noHealth() (*health.Client, error) { return nil, hostcall.ErrNoHost }

func TestAModelThatHasNoProfileIsAFailedPrecondition(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	h := newHandler(newServer(nil), noPlaces, sourcehealth.New(noHealth, console.Report))

	_, describeErr := h.Describe(t.Context())
	_, indexErr := h.IndexTexts(t.Context(), searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "text"}})

	if !hasCode(describeErr, rpcerror.FailedPrecondition) || !hasCode(indexErr, rpcerror.FailedPrecondition) {
		t.Fatalf("Describe error %v and IndexTexts error %v, want FailedPrecondition", describeErr, indexErr)
	}
}

func hasCode(err error, code rpcerror.Code) bool {
	return err != nil && strings.HasPrefix(err.Error(), code.String()+": ")
}

func TestTheHandlerDescribesAndIndexesWithTheModelTheEnvironmentNames(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	h := newHandler(newServer(nil), noPlaces, sourcehealth.New(noHealth, console.Report))

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
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	place := &nearestPlace{matches: []host.NearestMatch{{ChunkID: "41", Similarity: 0.8}, {ChunkID: "7", Similarity: 0.5}}}
	var vaults []string
	h := newHandler(newServer(nil), func(vaultID string) (host.PlaceSearch, error) {
		vaults = append(vaults, vaultID)
		return place, nil
	}, sourcehealth.New(noHealth, console.Report))

	found, err := h.SearchChunks(t.Context(), searchsource.Place{VaultID: "01PHYSICS", Recipe: "mock|r"},
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
