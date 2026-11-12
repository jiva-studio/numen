package host

import (
	"context"
	"fmt"
	"math"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// SearchClient works the search index and the search floor through HostSearchService.
type SearchClient struct {
	rpc   numenv1connect.HostSearchServiceClient
	vault string
}

var (
	_ SearchIndex   = (*SearchClient)(nil)
	_ PassageSearch = (*SearchClient)(nil)
	_ SearchPolicy  = (*SearchClient)(nil)
	_ PlaceSearch   = (*SearchClient)(nil)
)

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

func NewSearchClient(endpoint hostcall.Endpoint) *SearchClient {
	return NewSearchClientOver(hostcall.HTTPClient(), endpoint)
}

func NewSearchClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *SearchClient {
	return &SearchClient{rpc: numenv1connect.NewHostSearchServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// With returns the client for the vault with the given identity, used as the shown vault is. A call on
// it returns ErrUnknownVault when the installation has no such vault.
func (c *SearchClient) With(id domain.VaultID) *SearchClient {
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

func (c *SearchClient) GetIndexStats(ctx context.Context) (IndexStats, error) {
	res, err := c.rpc.GetIndexStats(ctx, newSearchRequest(c, &v1.GetIndexStatsRequest{}))
	if err != nil {
		return IndexStats{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return IndexStats{}, err
	}
	extensions := make([]ExtensionStats, len(res.Msg.GetExtensions()))
	for i, one := range res.Msg.GetExtensions() {
		extensions[i] = ExtensionStats{Extension: one.GetExtension(), SourceCount: one.GetSourceCount(), ChunkCount: one.GetChunkCount()}
	}
	return IndexStats{
		Recipe:        res.Msg.GetRecipe(),
		Model:         res.Msg.GetModel(),
		Dimensions:    res.Msg.GetDimensions(),
		ChunkCount:    res.Msg.GetChunkCount(),
		VectorCount:   res.Msg.GetVectorCount(),
		SourceCount:   res.Msg.GetSourceCount(),
		Extensions:    extensions,
		DatabaseBytes: res.Msg.GetDatabaseBytes(),
		VectorBytes:   res.Msg.GetVectorBytes(),
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

func (c *SearchClient) SearchPassages(ctx context.Context, query PassageQuery) ([]Passage, error) {
	kinds := make([]string, len(query.Kinds))
	for i, kind := range query.Kinds {
		kinds[i] = string(kind)
	}
	res, err := c.rpc.SearchPassages(ctx, newSearchRequest(c, &v1.HostSearchServiceSearchPassagesRequest{
		Query: query.Query, Limit: int32(min(max(query.Limit, math.MinInt32), math.MaxInt32)), Kinds: kinds, PathPrefix: query.PathPrefix,
	}))
	if err != nil {
		return nil, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return nil, err
	}
	found := make([]Passage, len(res.Msg.GetPassages()))
	for i, one := range res.Msg.GetPassages() {
		start, end := int(one.GetRange().GetStart()), int(one.GetRange().GetEnd())
		found[i] = Passage{
			Path: one.GetPath(), Text: one.GetText(), Location: one.GetLocation(),
			Start: start, Length: end - start, Line: int(one.GetLine()), Selector: one.GetSelector(),
		}
	}
	return found, nil
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

func (c *SearchClient) FindNearestPassages(ctx context.Context, query NearestQuery) ([]NearestMatch, error) {
	kinds := make([]string, len(query.Kinds))
	for i, kind := range query.Kinds {
		kinds[i] = string(kind)
	}
	res, err := c.rpc.FindNearestPassages(ctx, newSearchRequest(c, &v1.FindNearestPassagesRequest{
		Place: &v1.SearchPlace{VaultId: c.vault, Recipe: query.Recipe}, Vector: query.Vector,
		Limit: int32(min(max(query.Limit, math.MinInt32), math.MaxInt32)), Kinds: kinds, PathPrefix: query.PathPrefix, Floor: query.Floor,
	}))
	if err != nil {
		return nil, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return nil, err
	}
	found := make([]NearestMatch, len(res.Msg.GetMatches()))
	for i, one := range res.Msg.GetMatches() {
		found[i] = NearestMatch{ChunkID: one.GetChunkId(), Similarity: one.GetScore()}
	}
	return found, nil
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
