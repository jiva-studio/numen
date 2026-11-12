package searchsource

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

type fakeHandler struct {
	description Description
	places      *[]Place
	err         error
}

func (f fakeHandler) Describe(context.Context) (Description, error) {
	return f.description, f.err
}

func (f fakeHandler) IndexTexts(_ context.Context, place Place, texts []Text) ([][]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.places != nil {
		*f.places = append(*f.places, place)
	}
	vectors := make([][]float32, len(texts))
	for i, one := range texts {
		vectors[i] = []float32{float32(i), float32(len(one.Text))}
	}
	return vectors, nil
}

func (f fakeHandler) SearchPassages(_ context.Context, _ Place, q Query) ([]Match, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []Match{{ChunkID: q.Text, Score: q.Floor}}, nil
}

func (f fakeHandler) DescribeCoverage(_ context.Context, _ Place, counts Counts) (string, error) {
	if counts.Indexed == 0 {
		return "Not built yet", f.err
	}
	return "", f.err
}

func (f fakeHandler) EmbedQueries(_ context.Context, queries []string) ([][]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	vectors := make([][]float32, len(queries))
	for i, one := range queries {
		vectors[i] = []float32{float32(len(one))}
	}
	return vectors, nil
}

func serve(t *testing.T, h Handler) *testClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewHandler(h))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return newTestClient(web.Client(), web.URL)
}

func TestClient_RoundTripsTheDescriptionAndTheVectorsThroughTheHandler(t *testing.T) {
	want := Description{Name: "small", Recipe: "hub|small|2|1a2b3c4d", Width: 2, ChunkTokens: 200, MinTextLength: 40, Concurrency: 3, FloorHint: 0.25}
	var places []Place
	client := serve(t, fakeHandler{description: want, places: &places})

	got, err := client.Describe(t.Context())
	if err != nil || got != want {
		t.Fatalf("Describe = %+v, %v; want %+v", got, err, want)
	}
	place := Place{VaultID: "01VAULT", Recipe: "hub|small|2|1a2b3c4d|int8@0.4"}
	vectors, err := client.IndexTexts(t.Context(), place, []Text{{ChunkID: "7", Text: "a"}, {ChunkID: "8", Path: "book.pdf", VersionID: "v1", Text: "bcd"}})
	if err != nil || !reflect.DeepEqual(vectors, [][]float32{{0, 1}, {1, 3}}) {
		t.Fatalf("IndexTexts = %v, %v", vectors, err)
	}
	if !reflect.DeepEqual(places, []Place{place}) {
		t.Errorf("handler got places %v, want %v", places, []Place{place})
	}
	queries, err := client.EmbedQueries(t.Context(), []string{"entropy"})
	if err != nil || !reflect.DeepEqual(queries, [][]float32{{7}}) {
		t.Fatalf("EmbedQueries = %v, %v", queries, err)
	}
}

func TestTheAdapterCarriesTheQueryAndTheCountsToTheHandler(t *testing.T) {
	ad := &adapter{h: fakeHandler{}}
	found, err := ad.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchSourceServiceSearchPassagesRequest{Query: "12", Floor: 0.5}))
	if err != nil || len(found.Msg.GetMatches()) != 1 || found.Msg.GetMatches()[0].GetChunkId() != "12" || found.Msg.GetMatches()[0].GetScore() != 0.5 {
		t.Fatalf("SearchPassages = %v, %v", found, err)
	}
	coverage, err := ad.DescribeCoverage(t.Context(), connect.NewRequest(&v1.DescribeCoverageRequest{ChunkCount: 4}))
	if err != nil || coverage.Msg.GetText() != "Not built yet" {
		t.Fatalf("DescribeCoverage = %v, %v", coverage, err)
	}
}

func TestClient_PropagatesHandlerErrors(t *testing.T) {
	wantErr := errors.New("model failed")
	ad := &adapter{h: fakeHandler{err: wantErr}}
	if _, err := ad.DescribeSearchSource(t.Context(), connect.NewRequest(&v1.DescribeSearchSourceRequest{})); !errors.Is(err, wantErr) {
		t.Fatalf("adapter.DescribeSearchSource err = %v, want %v", err, wantErr)
	}
	if _, err := ad.IndexTexts(t.Context(), connect.NewRequest(&v1.IndexTextsRequest{})); !errors.Is(err, wantErr) {
		t.Fatalf("adapter.IndexTexts err = %v, want %v", err, wantErr)
	}

	client := serve(t, fakeHandler{err: wantErr})
	if _, err := client.Describe(t.Context()); err == nil {
		t.Fatal("expected error from Describe")
	}
	if _, err := client.IndexTexts(t.Context(), Place{}, []Text{{Text: "a"}}); err == nil {
		t.Fatal("expected error from IndexTexts")
	}
}

func writePlace(place Place) *v1.SearchPlace {
	return &v1.SearchPlace{VaultId: place.VaultID, Recipe: place.Recipe}
}

func readDescription(res *v1.DescribeSearchSourceResponse) Description {
	return Description{
		Name:          res.GetName(),
		Recipe:        res.GetRecipe(),
		Width:         res.GetWidth(),
		ChunkTokens:   res.GetChunkTokens(),
		MinTextLength: res.GetMinTextLength(),
		Concurrency:   res.GetConcurrency(),
		FloorHint:     res.GetFloorHint(),
	}
}

func readVectors(vectors []*v1.FloatVector) [][]float32 {
	out := make([][]float32, len(vectors))
	for i, one := range vectors {
		out[i] = one.GetValues()
	}
	return out
}

// testClient calls the search source service of a plugin.
type testClient struct {
	rpc numenv1connect.SearchSourceServiceClient
}

// newTestClient creates a client of the search source service served at baseURL.
func newTestClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) *testClient {
	return &testClient{rpc: numenv1connect.NewSearchSourceServiceClient(httpClient, baseURL, opts...)}
}

// Describe returns the description of the source the plugin serves.
func (c *testClient) Describe(ctx context.Context) (Description, error) {
	res, err := c.rpc.DescribeSearchSource(ctx, connect.NewRequest(&v1.DescribeSearchSourceRequest{}))
	if err != nil {
		return Description{}, err
	}
	return readDescription(res.Msg), nil
}

// IndexTexts returns one vector per text, in order.
func (c *testClient) IndexTexts(ctx context.Context, place Place, texts []Text) ([][]float32, error) {
	indexed := make([]*v1.IndexedText, len(texts))
	for i, one := range texts {
		indexed[i] = &v1.IndexedText{ChunkId: one.ChunkID, Path: one.Path, VersionId: one.VersionID, Text: one.Text}
	}
	res, err := c.rpc.IndexTexts(ctx, connect.NewRequest(&v1.IndexTextsRequest{Place: writePlace(place), Texts: indexed}))
	if err != nil {
		return nil, err
	}
	return readVectors(res.Msg.GetVectors()), nil
}

// EmbedQueries returns one vector per query, in order.
func (c *testClient) EmbedQueries(ctx context.Context, queries []string) ([][]float32, error) {
	res, err := c.rpc.EmbedQueries(ctx, connect.NewRequest(&v1.EmbedQueriesRequest{Queries: queries}))
	if err != nil {
		return nil, err
	}
	return readVectors(res.Msg.GetVectors()), nil
}
