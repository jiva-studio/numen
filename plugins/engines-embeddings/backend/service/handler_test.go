package service

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	hostdomain "github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/sourcehealth"
)

// nearestPlace is the index of the host as a search source asks it: it records the queries and answers
// the chunks it holds.
type nearestPlace struct {
	asked   []host.NearestQuery
	matches []host.NearestMatch
	err     error
}

func (p *nearestPlace) FindNearestChunks(_ context.Context, query host.NearestQuery) ([]host.NearestMatch, error) {
	p.asked = append(p.asked, query)
	return p.matches, p.err
}

func noPlaces(string) (host.PlaceSearch, error) { return nil, errors.New("no host") }

func noHealth() (*health.Client, error) { return nil, hostcall.ErrNoHost }

func TestAModelThatHasNoProfileIsAFailedPrecondition(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	hostHealth := healthtest.NewHost(t)
	for name, value := range hostHealth.BuildVars(linkedToken) {
		t.Setenv(name, value)
	}
	h := newHandler(newServer(nil, nil, hosttest.NewQueue().EnqueueWork), noPlaces, sourcehealth.New(health.NewClientFromEnv, console.Report))

	_, describeErr := h.Describe(t.Context())
	if !hasCode(describeErr, "failed_precondition") {
		t.Fatalf("Describe error %v, want FailedPrecondition", describeErr)
	}

	_, indexErr := h.IndexTexts(t.Context(), searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "text"}})
	_, searchErr := h.SearchChunks(t.Context(), searchsource.Place{VaultID: "V", Recipe: "r"}, searchsource.Query{Text: "q"})

	if !hasCode(indexErr, "failed_precondition") || !hasCode(searchErr, "failed_precondition") {
		t.Fatalf("IndexTexts error %v and SearchChunks error %v, want FailedPrecondition", indexErr, searchErr)
	}
	if items := hostHealth.List(); len(items) != 1 || items[0].Key != "source" {
		t.Fatalf("health items after embed failure = %+v, want one source item", items)
	}
}

func hasCode(err error, code string) bool {
	return err != nil && strings.HasPrefix(err.Error(), code+": ")
}

func TestTheHandlerDescribesAndIndexesWithTheModelTheEnvironmentNames(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	h := newHandler(newServer(nil, nil, hosttest.NewQueue().EnqueueWork), noPlaces, sourcehealth.New(noHealth, console.Report))

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
	if slices.Equal(vectors[0], make([]float32, len(vectors[0]))) || !slices.Equal(vectors[0], vectors[1]) {
		t.Fatalf("expected non-zero matching concept vectors for boat and ferry")
	}
}

func TestASearchAsksThePlaceWithTheVectorOfTheQueryAndAnswersItsChunks(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	place := &nearestPlace{matches: []host.NearestMatch{{ChunkID: "41", Similarity: 0.8}, {ChunkID: "7", Similarity: 0.5}}}
	var vaults []string
	h := newHandler(newServer(nil, nil, hosttest.NewQueue().EnqueueWork), func(vaultID string) (host.PlaceSearch, error) {
		vaults = append(vaults, vaultID)
		return place, nil
	}, sourcehealth.New(noHealth, console.Report))

	query := searchsource.Query{Text: "entropy", Limit: 5, Kinds: []string{"file", "note"}, PathPrefix: "books/", Floor: 0.4}
	found, err := h.SearchChunks(t.Context(), searchsource.Place{VaultID: "01PHYSICS", Recipe: "mock|r"}, query)
	if err != nil {
		t.Fatal(err)
	}

	want := []searchsource.Match{{ChunkID: "41", Score: 0.8}, {ChunkID: "7", Score: 0.5}}
	if !slices.Equal(found, want) || !slices.Equal(vaults, []string{"01PHYSICS"}) {
		t.Fatalf("found %v in vaults %v, want %v in 01PHYSICS", found, vaults, want)
	}
	asked := place.asked[0]
	if asked.Recipe != "mock|r" || asked.Limit != 5 || asked.Floor != 0.4 || asked.PathPrefix != "books/" ||
		!slices.Equal(asked.Kinds, []hostdomain.SourceKind{"file", "note"}) || len(asked.Vector) == 0 {
		t.Errorf("the place was asked %+v", asked)
	}
	if h.queries.Count() != 1 {
		t.Fatalf("cached queries = %d, want 1", h.queries.Count())
	}

	wantErr := errors.New("index broken")
	place.err = wantErr
	if _, err := h.SearchChunks(t.Context(), searchsource.Place{VaultID: "01PHYSICS", Recipe: "mock|r"}, query); !errors.Is(err, wantErr) {
		t.Fatalf("SearchChunks place error = %v, want %v", err, wantErr)
	}

	hNoPlace := newHandler(h.model, noPlaces, sourcehealth.New(noHealth, console.Report))
	if _, err := hNoPlace.SearchChunks(t.Context(), searchsource.Place{VaultID: "01PHYSICS", Recipe: "mock|r"}, query); !hasCode(err, "unavailable") {
		t.Fatalf("SearchChunks noPlaces error = %v, want Unavailable", err)
	}
}

func TestGetHealthTextUppercasesAndTruncatesAt120Runes(t *testing.T) {
	if got := getHealthText(""); got != "" {
		t.Fatalf("getHealthText(\"\") = %q, want empty", got)
	}
	exact := "é" + strings.Repeat("b", 119)
	wantExact := "É" + strings.Repeat("b", 119)
	if got := getHealthText(exact); got != wantExact {
		t.Fatalf("getHealthText(120 runes) = %q (len %d), want %q", got, len([]rune(got)), wantExact)
	}
	over := "é" + strings.Repeat("b", 120)
	wantOver := "É" + strings.Repeat("b", 118) + "…"
	if got := getHealthText(over); got != wantOver {
		t.Fatalf("getHealthText(121 runes) = %q (len %d), want %q", got, len([]rune(got)), wantOver)
	}
}
