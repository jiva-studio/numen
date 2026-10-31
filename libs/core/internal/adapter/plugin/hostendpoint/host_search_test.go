package hostendpoint

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// recordingSearch answers the search calls with fixed values and records the vault and the arguments
// of each. A call answers err when it is set.
type recordingSearch struct {
	err    error
	vaults []domain.VaultID
	limits []int
	floors []float64
}

func (s *recordingSearch) GetEmbeddingState(_ context.Context, v domain.Vault) (pluginport.PluginEmbeddingState, error) {
	s.vaults = append(s.vaults, v.ID)
	return pluginport.PluginEmbeddingState{Recipe: "r", Model: "m", FloorHint: 0.4, ChunkCount: 5, EmbeddedCount: 5}, s.err
}

func (s *recordingSearch) EmbedTexts(_ context.Context, texts []string) (pluginport.PluginEmbeddedTexts, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), 1}
	}
	return pluginport.PluginEmbeddedTexts{Recipe: "r", Vectors: vectors}, s.err
}

func (s *recordingSearch) ListNoteTitles(_ context.Context, v domain.Vault, limit int) ([]pluginport.PluginNoteTitle, error) {
	s.vaults = append(s.vaults, v.ID)
	s.limits = append(s.limits, limit)
	return []pluginport.PluginNoteTitle{{Source: 2, Title: "Entropy"}, {Source: 7, Title: "Tide"}}, s.err
}

func (s *recordingSearch) ListSmallChunkVectors(_ context.Context, v domain.Vault, _ string, limit int) ([]pluginport.PluginChunkVector, error) {
	s.vaults = append(s.vaults, v.ID)
	s.limits = append(s.limits, limit)
	return []pluginport.PluginChunkVector{{Source: 3, Vector: []byte{1, 2, 3}}}, s.err
}

func (s *recordingSearch) GetBestSimilarity(context.Context, domain.Vault, string, []float32) (float64, error) {
	return 0.75, s.err
}

func (s *recordingSearch) GetSearchFloor(context.Context, domain.Vault, string) (pluginport.PluginSearchFloor, error) {
	return pluginport.PluginSearchFloor{
		Floor: 0.5, Source: pluginport.PluginFloorStored, Stored: 0.5, Hint: 0.4, Default: 0.3,
	}, s.err
}

func (s *recordingSearch) SetSearchFloor(_ context.Context, v domain.Vault, _ string, floor float64) error {
	s.vaults = append(s.vaults, v.ID)
	s.floors = append(s.floors, floor)
	return s.err
}

func newSearchClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostSearchServiceClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(&searchEndpoint{endpoint: endpoint}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostSearchServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func newSearchCallers(search pluginport.PluginSearch) *callers {
	fakeCallers := newCallers(nil)
	fakeCallers.vault = newMemoryVault(nil)
	fakeCallers.search = search
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
	titles, err := client.ListNoteTitles(ctx, connect.NewRequest(&v1.ListNoteTitlesRequest{Limit: 9}))
	if err != nil || len(titles.Msg.GetTitles()) != 2 || titles.Msg.GetTitles()[0].GetSourceId() != 2 || titles.Msg.GetTitles()[1].GetTitle() != "Tide" {
		t.Fatalf("titles %v (%v)", titles.Msg, err)
	}
	vectors, err := client.ListSmallChunkVectors(ctx, connect.NewRequest(&v1.ListSmallChunkVectorsRequest{Recipe: "r", Limit: 4}))
	if err != nil || len(vectors.Msg.GetVectors()) != 1 || !slices.Equal(vectors.Msg.GetVectors()[0].GetEmbedding(), []byte{1, 2, 3}) {
		t.Fatalf("vectors %v (%v)", vectors.Msg, err)
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
	if !slices.Equal(search.limits, []int{9, 4}) || !slices.Equal(search.floors, []float64{0.6}) {
		t.Errorf("the host received limits %v and floors %v", search.limits, search.floors)
	}
	for _, permission := range []string{"index:read", "search:policy"} {
		if !slices.Contains(fakeCallers.checked, "md.numen.a "+permission) {
			t.Errorf("checked %v, want %s asked", fakeCallers.checked, permission)
		}
	}
	wantChecked := []string{
		"md.numen.a index:read", "md.numen.a index:read", "md.numen.a index:read", "md.numen.a index:read",
		"md.numen.a index:read", "md.numen.a search:policy", "md.numen.a search:policy",
	}
	if !slices.Equal(fakeCallers.checked, wantChecked) {
		t.Errorf("checked %v, want %v", fakeCallers.checked, wantChecked)
	}
}

func TestHostSearchRefusesAPluginWithoutPermissionAndAnUnknownToken(t *testing.T) {
	ctx := t.Context()
	fakeCallers := newSearchCallers(&recordingSearch{})

	denied := newSearchClient(t, fakeCallers, "token-b")
	if _, err := denied.ListNoteTitles(ctx, connect.NewRequest(&v1.ListNoteTitlesRequest{})); readCode(t, err) != connect.CodePermissionDenied {
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
	fakeCallers := newCallers(nil)
	fakeCallers.vault = physics
	fakeCallers.search = search
	client := newSearchClient(t, fakeCallers, "token-a")

	named := connect.NewRequest(&v1.ListNoteTitlesRequest{})
	named.Header().Set("Numen-Vault", "v2")
	if _, err := client.ListNoteTitles(ctx, named); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListNoteTitles(ctx, connect.NewRequest(&v1.ListNoteTitlesRequest{})); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(search.vaults, []domain.VaultID{"v2", "v1"}) {
		t.Errorf("the calls worked the vaults %v, want the named v2 and then the shown v1", search.vaults)
	}

	unknown := connect.NewRequest(&v1.ListNoteTitlesRequest{})
	unknown.Header().Set("Numen-Vault", "v9")
	if _, err := client.ListNoteTitles(ctx, unknown); readCode(t, err) != connect.CodeNotFound {
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

	noVault := newCallers(nil)
	noVault.search = &recordingSearch{}
	state, err := newSearchClient(t, noVault, "token-a").GetEmbeddingState(ctx, connect.NewRequest(&v1.GetEmbeddingStateRequest{}))
	if err != nil || state.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT {
		t.Errorf("a host showing no vault answered %v (%v)", state.Msg, err)
	}

	noSearch := newCallers(nil)
	noSearch.vault = newMemoryVault(nil)
	embedded, err := newSearchClient(t, noSearch, "token-a").EmbedTexts(ctx, connect.NewRequest(&v1.EmbedTextsRequest{Texts: []string{"a"}}))
	if err != nil || embedded.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Errorf("a host with no search answered %v (%v)", embedded.Msg, err)
	}

	for _, tc := range []struct {
		name string
		err  error
		want v1.HostSearchErrorCode
	}{
		{"no model", pluginport.ErrNoModel, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL},
		{"a recipe mismatch", pluginport.ErrRecipeMismatch, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH},
		{"an invalid floor", pluginport.ErrInvalidFloor, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR},
		{"a limit exceeded", pluginport.ErrLimitExceeded, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED},
		{"an embedding failure", pluginport.ErrEmbeddingFailed, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newSearchClient(t, newSearchCallers(&recordingSearch{err: tc.err}), "token-a")
			set, err := client.SetSearchFloor(ctx, connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.5}))
			if err != nil || set.Msg.GetError() != tc.want {
				t.Errorf("answered %v (%v), want %v", set.Msg, err, tc.want)
			}
			titles, err := client.GetBestSimilarity(ctx, connect.NewRequest(&v1.GetBestSimilarityRequest{Recipe: "r"}))
			if err != nil || titles.Msg.GetError() != tc.want || titles.Msg.GetSimilarity() != 0 {
				t.Errorf("answered %v (%v), want only the code %v", titles.Msg, err, tc.want)
			}
		})
	}

	failing := newSearchClient(t, newSearchCallers(&recordingSearch{err: errors.New("disk")}), "token-a")
	if _, err := failing.GetSearchFloor(ctx, connect.NewRequest(&v1.GetSearchFloorRequest{Recipe: "r"})); readCode(t, err) != connect.CodeInternal {
		t.Errorf("a failure the protocol does not name got %v, want internal", err)
	}
}
