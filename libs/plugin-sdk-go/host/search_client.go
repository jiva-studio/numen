package host

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// SearchClient works the search index and the search floor through HostSearchService.
type SearchClient struct {
	rpc numenv1connect.HostSearchServiceClient
	// vault is the identity of the vault the client works. Empty is the vault the window shows.
	vault string
}

var _ Search = (*SearchClient)(nil)

// NewSearchClientFromEnv connects to the host the process was started for. It returns hostcall.ErrNoHost
// where the host named none.
func NewSearchClientFromEnv() (*SearchClient, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewSearchClient(addr), nil
}

// NewSearchClientFromVars is NewSearchClientFromEnv over the variables given.
func NewSearchClientFromVars(vars map[string]string) (*SearchClient, error) {
	addr, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewSearchClient(addr), nil
}

// NewSearchClient connects to the host at the endpoint.
func NewSearchClient(endpoint hostcall.Endpoint) *SearchClient {
	return NewSearchClientOver(hostcall.HTTPClient(), endpoint)
}

// NewSearchClientOver connects to the host at the endpoint through the given HTTP client.
func NewSearchClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *SearchClient {
	return &SearchClient{rpc: numenv1connect.NewHostSearchServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

func (c *SearchClient) With(id VaultID) Search {
	linked := *c
	linked.vault = string(id)
	return &linked
}

func newSearchRequest[T any](c *SearchClient, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if c.vault != "" {
		req.Header().Set(vaultHeader, c.vault)
	}
	return req
}

// readSearchCallError converts the failure of a call into its error: a vault the host does not know is
// ErrUnknownVault, and a permission the manifest lacks is ErrPermissionDenied.
func readSearchCallError(err error) error {
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		return fmt.Errorf("%w: %w", ErrPermissionDenied, err)
	}
	return readError(err)
}

func readSearchError(code v1.HostSearchErrorCode) error {
	switch code {
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT:
		return ErrNoVault
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL:
		return ErrNoModel
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH:
		return ErrRecipeMismatch
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR:
		return ErrInvalidFloor
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED:
		return ErrLimitExceeded
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED:
		return ErrEmbeddingFailed
	case v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_QUERY:
		return ErrInvalidQuery
	}
	return fmt.Errorf("host: rejected with code %s", code)
}

func readFloorSource(source v1.SearchFloorSource) FloorSource {
	switch source {
	case v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_STORED:
		return FloorFromStore
	case v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_MODEL_HINT:
		return FloorFromModelHint
	case v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_DEFAULT, v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_UNSPECIFIED:
		return FloorFromDefault
	}
	return FloorFromDefault
}

func (c *SearchClient) GetEmbeddingState(ctx context.Context) (EmbeddingState, error) {
	res, err := c.rpc.GetEmbeddingState(ctx, newSearchRequest(c, &v1.GetEmbeddingStateRequest{}))
	if err != nil {
		return EmbeddingState{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return EmbeddingState{}, err
	}
	return EmbeddingState{
		Recipe:        res.Msg.GetRecipe(),
		Model:         res.Msg.GetModel(),
		FloorHint:     res.Msg.GetFloorHint(),
		ChunkCount:    res.Msg.GetChunkCount(),
		EmbeddedCount: res.Msg.GetEmbeddedCount(),
		IsReady:       res.Msg.GetIsReady(),
	}, nil
}

func (c *SearchClient) EmbedTexts(ctx context.Context, texts []string) (EmbeddedTexts, error) {
	res, err := c.rpc.EmbedTexts(ctx, newSearchRequest(c, &v1.EmbedTextsRequest{Texts: texts}))
	if err != nil {
		return EmbeddedTexts{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return EmbeddedTexts{}, err
	}
	vectors := make([][]float32, len(res.Msg.GetVectors()))
	for i, one := range res.Msg.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return EmbeddedTexts{Recipe: res.Msg.GetRecipe(), Vectors: vectors}, nil
}

func (c *SearchClient) SampleSimilarities(ctx context.Context, recipe string, queries []SimilarityQuery) (SimilaritySample, error) {
	wire := make([]*v1.SimilarityQuery, len(queries))
	for i, one := range queries {
		wire[i] = &v1.SimilarityQuery{Vector: &v1.FloatVector{Values: one.Vector}, Path: one.Path}
	}
	res, err := c.rpc.SampleSimilarities(ctx, newSearchRequest(c, &v1.SampleSimilaritiesRequest{Recipe: recipe, Queries: wire}))
	if err != nil {
		return SimilaritySample{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return SimilaritySample{}, err
	}
	rows := make([][]float64, len(res.Msg.GetRows()))
	for i, one := range res.Msg.GetRows() {
		rows[i] = one.GetValues()
	}
	return SimilaritySample{
		PassageCount: int(res.Msg.GetPassageCount()),
		SourceCount:  int(res.Msg.GetSourceCount()),
		Rows:         rows,
	}, nil
}

func (c *SearchClient) GetBestSimilarity(ctx context.Context, recipe string, query []float32) (float64, error) {
	res, err := c.rpc.GetBestSimilarity(ctx, newSearchRequest(c, &v1.GetBestSimilarityRequest{Recipe: recipe, Query: query}))
	if err != nil {
		return 0, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return 0, err
	}
	return res.Msg.GetSimilarity(), nil
}

func (c *SearchClient) GetSearchFloor(ctx context.Context, recipe string) (SearchFloor, error) {
	res, err := c.rpc.GetSearchFloor(ctx, newSearchRequest(c, &v1.GetSearchFloorRequest{Recipe: recipe}))
	if err != nil {
		return SearchFloor{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return SearchFloor{}, err
	}
	return SearchFloor{
		Floor:   res.Msg.GetFloor(),
		Source:  readFloorSource(res.Msg.GetSource()),
		Stored:  res.Msg.GetStoredFloor(),
		Hint:    res.Msg.GetHintFloor(),
		Default: res.Msg.GetDefaultFloor(),
	}, nil
}

func (c *SearchClient) SetSearchFloor(ctx context.Context, recipe string, floor float64) error {
	res, err := c.rpc.SetSearchFloor(ctx, newSearchRequest(c, &v1.SetSearchFloorRequest{Recipe: recipe, Floor: floor}))
	if err != nil {
		return readSearchCallError(err)
	}
	return readSearchError(res.Msg.GetError())
}
