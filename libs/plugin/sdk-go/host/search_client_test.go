package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// searchHost is a host whose search index answers with fixed values, and that records the vault and
// the arguments each call named.
type searchHost struct {
	numenv1connect.UnimplementedHostSearchServiceHandler
	mu      sync.Mutex
	code    v1.HostSearchErrorCode
	status  *connect.Error
	source  v1.SearchFloorSource
	vaults  []string
	sampled []*v1.SimilarityQuery
	recipes []string
	floors  []float64
	queries [][]float32
}

func (h *searchHost) begin(header http.Header) *connect.Error {
	h.vaults = append(h.vaults, header.Get("Numen-Vault"))
	return h.status
}

func (h *searchHost) GetEmbeddingState(_ context.Context, req *connect.Request[v1.GetEmbeddingStateRequest]) (*connect.Response[v1.GetEmbeddingStateResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetEmbeddingStateResponse{
		Error: h.code, Recipe: "recipe", Model: "model", FloorHint: 0.4, ChunkCount: 8, EmbeddedCount: 8, IsReady: true,
	}), nil
}

func (h *searchHost) EmbedTexts(_ context.Context, req *connect.Request[v1.EmbedTextsRequest]) (*connect.Response[v1.EmbedTextsResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	vectors := make([]*v1.FloatVector, len(req.Msg.GetTexts()))
	for i := range vectors {
		vectors[i] = &v1.FloatVector{Values: []float32{float32(i), 1}}
	}
	return connect.NewResponse(&v1.EmbedTextsResponse{Error: h.code, Recipe: "recipe", Vectors: vectors}), nil
}

func (h *searchHost) SampleSimilarities(_ context.Context, req *connect.Request[v1.SampleSimilaritiesRequest]) (*connect.Response[v1.SampleSimilaritiesResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	h.recipes = append(h.recipes, req.Msg.GetRecipe())
	h.sampled = append(h.sampled, req.Msg.GetQueries()...)
	return connect.NewResponse(&v1.SampleSimilaritiesResponse{
		Error:        h.code,
		PassageCount: 3,
		SourceCount:  2,
		Rows:         []*v1.SimilarityRow{{Values: []float64{0.25, 0.5}}, {}},
	}), nil
}

func (h *searchHost) GetBestSimilarity(_ context.Context, req *connect.Request[v1.GetBestSimilarityRequest]) (*connect.Response[v1.GetBestSimilarityResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	h.recipes = append(h.recipes, req.Msg.GetRecipe())
	h.queries = append(h.queries, req.Msg.GetQuery())
	return connect.NewResponse(&v1.GetBestSimilarityResponse{Error: h.code, Similarity: 0.62}), nil
}

func (h *searchHost) GetSearchFloor(_ context.Context, req *connect.Request[v1.GetSearchFloorRequest]) (*connect.Response[v1.GetSearchFloorResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetSearchFloorResponse{
		Error: h.code, Floor: 0.5, Source: h.source,
		StoredFloor: 0.5, HintFloor: 0.4, DefaultFloor: 0.3,
	}), nil
}

func (h *searchHost) SetSearchFloor(_ context.Context, req *connect.Request[v1.SetSearchFloorRequest]) (*connect.Response[v1.SetSearchFloorResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	h.recipes = append(h.recipes, req.Msg.GetRecipe())
	h.floors = append(h.floors, req.Msg.GetFloor())
	return connect.NewResponse(&v1.SetSearchFloorResponse{Error: h.code}), nil
}

func dialSearch(t *testing.T, h *searchHost) *host.SearchClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewSearchClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestASearchClientReadsTheStateAndSamplesSimilarities(t *testing.T) {
	ctx := t.Context()
	h := &searchHost{}
	c := dialSearch(t, h)

	state, err := c.GetEmbeddingState(ctx)
	want := host.EmbeddingState{Recipe: "recipe", Model: "model", FloorHint: 0.4, ChunkCount: 8, EmbeddedCount: 8, IsReady: true}
	if err != nil || state != want {
		t.Errorf("state %+v (%v), want %+v", state, err, want)
	}
	sample, err := c.SampleSimilarities(ctx, "recipe", []host.SimilarityQuery{{Path: "a.md", Vector: []float32{1, 0}}, {}})
	wantSample := host.SimilaritySample{PassageCount: 3, SourceCount: 2, Rows: [][]float64{{0.25, 0.5}, nil}}
	if err != nil || sample.PassageCount != wantSample.PassageCount || sample.SourceCount != wantSample.SourceCount ||
		len(sample.Rows) != 2 || !slices.Equal(sample.Rows[0], wantSample.Rows[0]) || len(sample.Rows[1]) != 0 {
		t.Errorf("sample %+v (%v), want %+v", sample, err, wantSample)
	}
	best, err := c.GetBestSimilarity(ctx, "recipe", []float32{0.5, 0.25})
	if err != nil || best != 0.62 {
		t.Errorf("best %v (%v)", best, err)
	}
	if !slices.Equal(h.recipes, []string{"recipe", "recipe"}) || len(h.sampled) != 2 || h.sampled[0].GetPath() != "a.md" ||
		!slices.Equal(h.sampled[0].GetVector().GetValues(), []float32{1, 0}) ||
		len(h.queries) != 1 || !slices.Equal(h.queries[0], []float32{0.5, 0.25}) {
		t.Errorf("the host received recipes %v, sampled queries %v and queries %v", h.recipes, h.sampled, h.queries)
	}
}

func TestASearchClientEmbedsTextsAndReadsAndSetsTheFloor(t *testing.T) {
	ctx := t.Context()
	h := &searchHost{source: v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_STORED}
	c := dialSearch(t, h)

	embedded, err := c.EmbedTexts(ctx, []string{"a", "b"})
	if err != nil || embedded.Recipe != "recipe" || len(embedded.Vectors) != 2 || !slices.Equal(embedded.Vectors[1], []float32{1, 1}) {
		t.Errorf("embedded %+v (%v)", embedded, err)
	}
	floor, err := c.GetSearchFloor(ctx, "recipe")
	want := host.SearchFloor{Floor: 0.5, Source: host.FloorFromStore, Stored: 0.5, Hint: 0.4, Default: 0.3}
	if err != nil || floor != want {
		t.Errorf("floor %+v (%v), want %+v", floor, err, want)
	}
	if err := c.SetSearchFloor(ctx, "recipe", 0.45); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.floors, []float64{0.45}) {
		t.Errorf("the host received the floors %v", h.floors)
	}
}

func TestASearchOfAnotherVaultNamesItOnEveryCall(t *testing.T) {
	ctx := t.Context()
	h := &searchHost{}
	c := dialSearch(t, h)
	linked := c.With("Chemistry")

	if _, err := linked.GetEmbeddingState(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := linked.SampleSimilarities(ctx, "recipe", nil); err != nil {
		t.Fatal(err)
	}
	if err := linked.SetSearchFloor(ctx, "recipe", 0.5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetEmbeddingState(ctx); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.vaults, []string{"Chemistry", "Chemistry", "Chemistry", ""}) {
		t.Errorf("the calls named the vaults %q", h.vaults)
	}
}

func TestEachCodeOfTheSearchIsItsOwnError(t *testing.T) {
	cases := map[v1.HostSearchErrorCode]error{
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT:         host.ErrNoVault,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL:         host.ErrNoModel,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH:  host.ErrRecipeMismatch,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR:    host.ErrInvalidFloor,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED:   host.ErrLimitExceeded,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED: host.ErrEmbeddingFailed,
		v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_QUERY:    host.ErrInvalidQuery,
	}
	for code, want := range cases {
		ctx := t.Context()
		c := dialSearch(t, &searchHost{code: code})
		if _, err := c.GetEmbeddingState(ctx); !errors.Is(err, want) {
			t.Errorf("state, code %s: got %v, want %v", code, err, want)
		}
		if _, err := c.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, want) {
			t.Errorf("embed, code %s: got %v, want %v", code, err, want)
		}
		if _, err := c.SampleSimilarities(ctx, "r", nil); !errors.Is(err, want) {
			t.Errorf("sample, code %s: got %v, want %v", code, err, want)
		}
		if _, err := c.GetBestSimilarity(ctx, "r", nil); !errors.Is(err, want) {
			t.Errorf("best, code %s: got %v, want %v", code, err, want)
		}
		if _, err := c.GetSearchFloor(ctx, "r"); !errors.Is(err, want) {
			t.Errorf("floor, code %s: got %v, want %v", code, err, want)
		}
		if err := c.SetSearchFloor(ctx, "r", 0.5); !errors.Is(err, want) {
			t.Errorf("set, code %s: got %v, want %v", code, err, want)
		}
	}
}

func TestEveryErrorCodeAndFloorSourceOfTheSchemaIsRead(t *testing.T) {
	schematest.CheckHandled(t, func(code v1.HostSearchErrorCode) bool {
		h := &searchHost{code: code}
		_, err := dialSearch(t, h).GetEmbeddingState(t.Context())
		return code == v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED || err != nil
	})
	schematest.CheckHandled(t, func(source v1.SearchFloorSource) bool {
		got, err := dialSearch(t, &searchHost{source: source}).GetSearchFloor(t.Context(), "r")
		return err == nil && got.Source != 0
	})
}

func TestAPermissionTheManifestLacksIsDeniedAndAnUnknownVaultIsUnknown(t *testing.T) {
	ctx := t.Context()
	denied := dialSearch(t, &searchHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("plugin x: permission denied"))})
	if _, err := denied.SampleSimilarities(ctx, "r", nil); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("a denied read gave %v", err)
	}
	if err := denied.SetSearchFloor(ctx, "r", 0.5); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("a denied set gave %v", err)
	}
	unknown := dialSearch(t, &searchHost{status: connect.NewError(connect.CodeNotFound, errors.New("vault"))})
	if _, err := unknown.With("none").GetEmbeddingState(ctx); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("an unknown vault gave %v", err)
	}
	other := dialSearch(t, &searchHost{status: connect.NewError(connect.CodeInternal, errors.New("boom"))})
	if _, err := other.GetEmbeddingState(ctx); err == nil || errors.Is(err, host.ErrUnknownVault) || errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("an internal failure gave %v", err)
	}
}

func TestTheSearchSourceReadyEventNamesTheVaultAndOnlyThatEvent(t *testing.T) {
	id, ok := host.ReadSearchSourceReadyEvent(protocol.GetSearchSourceReadyEvent("01VAULT"))
	if !ok || id != "01VAULT" {
		t.Errorf("read %q, %v", id, ok)
	}
	for _, event := range []string{"", "onFile:md", "onSearchSourceReady:", "onSearchSourceReady", "*"} {
		if id, ok := host.ReadSearchSourceReadyEvent(event); ok {
			t.Errorf("%q was read as the event of the vault %q", event, id)
		}
	}
}

func (h *searchHost) GetIndexStats(_ context.Context, req *connect.Request[v1.GetIndexStatsRequest]) (*connect.Response[v1.GetIndexStatsResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.begin(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetIndexStatsResponse{
		Error: h.code, Recipe: "recipe", Model: "model", Dimensions: 1024, ChunkCount: 12, VectorCount: 9, SourceCount: 3,
		Extensions:    []*v1.ExtensionStats{{Extension: "md", SourceCount: 2, ChunkCount: 7}, {Extension: "pdf", SourceCount: 1, ChunkCount: 5}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	}), nil
}

func TestASearchClientReadsTheIndexStatistics(t *testing.T) {
	ctx := t.Context()
	h := &searchHost{}
	c := dialSearch(t, h)

	got, err := c.With("Chemistry").GetIndexStats(ctx)
	want := host.IndexStats{
		Recipe: "recipe", Model: "model", Dimensions: 1024, ChunkCount: 12, VectorCount: 9, SourceCount: 3,
		Extensions:    []host.ExtensionStats{{Extension: "md", SourceCount: 2, ChunkCount: 7}, {Extension: "pdf", SourceCount: 1, ChunkCount: 5}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("statistics %+v (%v), want %+v", got, err, want)
	}
	if !slices.Equal(h.vaults, []string{"Chemistry"}) {
		t.Errorf("the call named the vaults %q", h.vaults)
	}

	if _, err := dialSearch(t, &searchHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL}).GetIndexStats(ctx); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("a host with no model gave %v", err)
	}
	denied := dialSearch(t, &searchHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("denied"))})
	if _, err := denied.GetIndexStats(ctx); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("a denied read gave %v", err)
	}
}
