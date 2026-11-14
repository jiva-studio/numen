package hostendpoint

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// recordingSearch answers the search calls with fixed values and records the vault and the arguments
// of each. A call answers err when it is set.
type recordingSearch struct {
	err     error
	vaults  []domain.VaultID
	recipes []string
	queries []plugin.SimilarityQuery
	floors  []float64

	textQueries []plugin.SearchResultQuery
}

func (s *recordingSearch) GetEmbeddingState(_ context.Context, v domain.Vault) (plugin.EmbeddingState, error) {
	s.vaults = append(s.vaults, v.ID)
	return plugin.EmbeddingState{Recipe: "r", Model: "m", FloorHint: 0.4, ChunkCount: 5, EmbeddedCount: 5}, s.err
}

func (s *recordingSearch) GetIndexStats(_ context.Context, v domain.Vault) (plugin.IndexStats, error) {
	s.vaults = append(s.vaults, v.ID)
	return plugin.IndexStats{
		Recipe: "r", Model: "m", Dimensions: 1024, Chunks: 12, Vectors: 9, Sources: 3,
		Extensions:    []plugin.ExtensionStats{{Extension: "md", Sources: 2, Chunks: 7}, {Extension: "pdf", Sources: 1, Chunks: 5}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	}, s.err
}

func (s *recordingSearch) EmbedTexts(_ context.Context, texts []string) (plugin.EmbeddedTexts, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), 1}
	}
	return plugin.EmbeddedTexts{Recipe: "r", Vectors: vectors}, s.err
}

func (s *recordingSearch) SampleSimilarities(_ context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error) {
	s.vaults = append(s.vaults, v.ID)
	s.recipes = append(s.recipes, recipe)
	s.queries = append(s.queries, queries...)
	return plugin.SimilaritySample{ChunkCount: 3, SourceCount: 2, Rows: [][]float64{{0.25, 0.5}, nil}}, s.err
}

func (s *recordingSearch) GetBestSimilarity(context.Context, domain.Vault, string, []float32) (float64, error) {
	return 0.75, s.err
}

func (s *recordingSearch) GetSearchFloor(context.Context, domain.Vault, string) (plugin.SearchFloor, error) {
	return plugin.SearchFloor{
		Floor: 0.5, Source: plugin.FloorStored, Stored: 0.5, Hint: 0.4, Default: 0.3,
	}, s.err
}

func (s *recordingSearch) SetSearchFloor(_ context.Context, v domain.Vault, _ string, floor float64) error {
	s.vaults = append(s.vaults, v.ID)
	s.floors = append(s.floors, floor)
	return s.err
}

func newSearchClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostSearchServiceClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers.asCallers()}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(&searchEndpoint{endpoint: endpoint}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostSearchServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func newSearchCallers(search any) *callers {
	fakeCallers := newCallers()
	fakeCallers.vault = vaultOf(newMemoryVault(nil))
	fakeCallers.search = searchOf(search)
	return fakeCallers
}

func TestHostSearchAnswersEveryCallAndChecksItsPermission(t *testing.T) {
	ctx := t.Context()
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	state, err := client.GetEmbeddingState(ctx, connect.NewRequest(&v1.GetEmbeddingStateRequest{}))
	if err != nil || state.Msg.GetRecipe() != "r" || state.Msg.GetModel() != "m" || state.Msg.GetFloorHint() != 0.4 ||
		state.Msg.GetChunkCount() != 5 || state.Msg.GetEmbeddedCount() != 5 || !state.Msg.GetIsReady() {
		t.Fatalf("state %v (%v)", state.Msg, err)
	}
	embedded, err := client.EmbedTexts(ctx, connect.NewRequest(&v1.EmbedTextsRequest{Texts: []string{"a", "b"}}))
	if err != nil || embedded.Msg.GetRecipe() != "r" || len(embedded.Msg.GetVectors()) != 2 || embedded.Msg.GetVectors()[1].GetValues()[0] != 1 {
		t.Fatalf("embedded %v (%v)", embedded.Msg, err)
	}
	sample, err := client.SampleSimilarities(ctx, connect.NewRequest(&v1.SampleSimilaritiesRequest{
		Recipe: "r",
		Queries: []*v1.SimilarityQuery{
			{Vector: &v1.FloatVector{Values: []float32{1, 0}}, Path: "a.md"},
			{},
		},
	}))
	if err != nil || sample.Msg.GetChunkCount() != 3 || sample.Msg.GetSourceCount() != 2 || len(sample.Msg.GetRows()) != 2 ||
		!slices.Equal(sample.Msg.GetRows()[0].GetValues(), []float64{0.25, 0.5}) || len(sample.Msg.GetRows()[1].GetValues()) != 0 {
		t.Fatalf("sample %v (%v)", sample.Msg, err)
	}
	best, err := client.GetBestSimilarity(ctx, connect.NewRequest(&v1.GetBestSimilarityRequest{Recipe: "r", Query: []float32{1}}))
	if err != nil || best.Msg.GetSimilarity() != 0.75 {
		t.Fatalf("best %v (%v)", best.Msg, err)
	}
	floor, err := client.GetSearchFloor(ctx, connect.NewRequest(&v1.GetSearchFloorRequest{Recipe: "r"}))
	if err != nil || floor.Msg.GetFloor() != 0.5 || floor.Msg.GetSource() != v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_STORED ||
		floor.Msg.GetStoredFloor() != 0.5 || floor.Msg.GetHintFloor() != 0.4 || floor.Msg.GetDefaultFloor() != 0.3 {
		t.Fatalf("floor %v (%v)", floor.Msg, err)
	}
	set, err := client.SetSearchFloor(ctx, connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.6}))
	if err != nil || set.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("set %v (%v)", set.Msg, err)
	}
	if !slices.Equal(search.recipes, []string{"r"}) || len(search.queries) != 2 || search.queries[0].Path != "a.md" ||
		!slices.Equal(search.queries[0].Vector, []float32{1, 0}) || len(search.queries[1].Vector) != 0 ||
		!slices.Equal(search.floors, []float64{0.6}) {
		t.Errorf("the host received recipes %v, queries %v and floors %v", search.recipes, search.queries, search.floors)
	}
	for _, permission := range []string{"index:read", "search:policy"} {
		if !slices.Contains(fakeCallers.checked, "md.numen.a "+permission) {
			t.Errorf("checked %v, want %s asked", fakeCallers.checked, permission)
		}
	}
	wantChecked := []string{
		"md.numen.a index:read", "md.numen.a index:read", "md.numen.a index:read", "md.numen.a index:read",
		"md.numen.a search:policy", "md.numen.a search:policy",
	}
	if !slices.Equal(fakeCallers.checked, wantChecked) {
		t.Errorf("checked %v, want %v", fakeCallers.checked, wantChecked)
	}
}

func TestHostSearchRefusesAPluginWithoutPermissionAndAnUnknownToken(t *testing.T) {
	ctx := t.Context()
	fakeCallers := newSearchCallers(&recordingSearch{})

	denied := newSearchClient(t, fakeCallers, "token-b")
	if _, err := denied.SampleSimilarities(ctx, connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"})); readCode(t, err) != connect.CodePermissionDenied {
		t.Errorf("a plugin without the permission got %v", err)
	}
	if _, err := denied.SetSearchFloor(ctx, connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.5})); readCode(t, err) != connect.CodePermissionDenied {
		t.Errorf("a plugin without the permission set a floor: %v", err)
	}
	unknown := newSearchClient(t, fakeCallers, "token-unknown")
	if _, err := unknown.GetEmbeddingState(ctx, connect.NewRequest(&v1.GetEmbeddingStateRequest{})); readCode(t, err) != connect.CodeUnauthenticated {
		t.Errorf("an unknown token got %v", err)
	}
}

func TestHostSearchNamesTheVaultOfTheHeaderAndRefusesAnUnknownOne(t *testing.T) {
	ctx := t.Context()
	search := &recordingSearch{}
	physics := newMemoryVault(nil)
	physics.others = map[domain.VaultID]*memoryVault{
		"v2": {vault: &domain.Vault{ID: "v2", Name: "Chemistry", Path: "/vaults/Chemistry"}},
	}
	fakeCallers := newCallers()
	fakeCallers.vault = vaultOf(physics)
	fakeCallers.search = searchOf(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	named := connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"})
	named.Header().Set("Numen-Vault", "v2")
	if _, err := client.SampleSimilarities(ctx, named); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SampleSimilarities(ctx, connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"})); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(search.vaults, []domain.VaultID{"v2", "v1"}) {
		t.Errorf("the calls worked the vaults %v, want the named v2 and then the shown v1", search.vaults)
	}

	unknown := connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"})
	unknown.Header().Set("Numen-Vault", "v9")
	if _, err := client.SampleSimilarities(ctx, unknown); readCode(t, err) != connect.CodeNotFound {
		t.Errorf("an unknown vault got %v, want not found", err)
	}
	setUnknown := connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.5})
	setUnknown.Header().Set("Numen-Vault", "v9")
	if _, err := client.SetSearchFloor(ctx, setUnknown); readCode(t, err) != connect.CodeNotFound {
		t.Errorf("a floor set on an unknown vault got %v, want not found", err)
	}
	if len(search.floors) != 0 {
		t.Errorf("a floor was set on an unknown vault: %v", search.floors)
	}
}

func TestHostSearchAnswersACodeForWhatItCannotServe(t *testing.T) {
	ctx := t.Context()

	noVault := newCallers()
	noVault.search = searchOf(&recordingSearch{})
	state, err := newSearchClient(t, noVault, "token-a").GetEmbeddingState(ctx, connect.NewRequest(&v1.GetEmbeddingStateRequest{}))
	if err != nil || state.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT {
		t.Errorf("a host showing no vault answered %v (%v)", state.Msg, err)
	}

	noSearch := newCallers()
	noSearch.vault = vaultOf(newMemoryVault(nil))
	embedded, err := newSearchClient(t, noSearch, "token-a").EmbedTexts(ctx, connect.NewRequest(&v1.EmbedTextsRequest{Texts: []string{"a"}}))
	if err != nil || embedded.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Errorf("a host with no search answered %v (%v)", embedded.Msg, err)
	}

	for _, tc := range []struct {
		name string
		err  error
		want v1.HostSearchErrorCode
	}{
		{"no model", plugin.ErrNoModel, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL},
		{"a recipe mismatch", plugin.ErrRecipeMismatch, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH},
		{"an invalid floor", plugin.ErrInvalidFloor, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR},
		{"a limit exceeded", plugin.ErrLimitExceeded, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED},
		{"an embedding failure", plugin.ErrEmbeddingFailed, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED},
		{"an invalid query", plugin.ErrInvalidQuery, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_QUERY},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newSearchClient(t, newSearchCallers(&recordingSearch{err: tc.err}), "token-a")
			set, err := client.SetSearchFloor(ctx, connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.5}))
			if err != nil || set.Msg.GetError() != tc.want {
				t.Errorf("answered %v (%v), want %v", set.Msg, err, tc.want)
			}
			sampled, err := client.SampleSimilarities(ctx, connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"}))
			if err != nil || sampled.Msg.GetError() != tc.want || len(sampled.Msg.GetRows()) != 0 || sampled.Msg.GetChunkCount() != 0 {
				t.Errorf("answered %v (%v), want only the code %v", sampled.Msg, err, tc.want)
			}
		})
	}

	failing := newSearchClient(t, newSearchCallers(&recordingSearch{err: errors.New("disk")}), "token-a")
	if _, err := failing.GetSearchFloor(ctx, connect.NewRequest(&v1.GetSearchFloorRequest{Recipe: "r"})); readCode(t, err) != connect.CodeInternal {
		t.Errorf("a failure the protocol does not name got %v, want internal", err)
	}
}

func TestHostSearchAnswersTheIndexStatistics(t *testing.T) {
	ctx := t.Context()
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	stats, err := client.GetIndexStats(ctx, connect.NewRequest(&v1.GetIndexStatsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := stats.Msg
	if got.GetRecipe() != "r" || got.GetModel() != "m" || got.GetDimensions() != 1024 ||
		got.GetChunkCount() != 12 || got.GetVectorCount() != 9 || got.GetSourceCount() != 3 ||
		got.GetDatabaseBytes() != 4096 || got.GetVectorBytes() != 2048 {
		t.Errorf("answered %v", got)
	}
	if len(got.GetExtensions()) != 2 || got.GetExtensions()[0].GetExtension() != "md" ||
		got.GetExtensions()[0].GetSourceCount() != 2 || got.GetExtensions()[0].GetChunkCount() != 7 ||
		got.GetExtensions()[1].GetExtension() != "pdf" {
		t.Errorf("answered extensions %v", got.GetExtensions())
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a index:read"}) {
		t.Errorf("checked %v, want index:read asked once", fakeCallers.checked)
	}
	if !slices.Equal(search.vaults, []domain.VaultID{"v1"}) {
		t.Errorf("the call worked the vaults %v, want the shown v1", search.vaults)
	}

	failing := newSearchClient(t, newSearchCallers(&recordingSearch{err: plugin.ErrNoModel}), "token-a")
	noModel, err := failing.GetIndexStats(ctx, connect.NewRequest(&v1.GetIndexStatsRequest{}))
	if err != nil || noModel.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL || noModel.Msg.GetChunkCount() != 0 {
		t.Errorf("an installation with no model answered %v (%v)", noModel.Msg, err)
	}
}
