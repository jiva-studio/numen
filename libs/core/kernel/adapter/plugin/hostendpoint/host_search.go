package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// searchEndpoint answers HostSearchService for the plugins that call the host.
type searchEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostSearchServiceHandler = (*searchEndpoint)(nil)

// searchCall is an authorized search call: the search index and the vault it works. Search is nil
// where the host has no search index.
type searchCall struct {
	search   *SearchAccess
	vault    domain.Vault
	hasVault bool
}

func (e *searchEndpoint) begin(header http.Header, permission string, needsVault bool) (searchCall, error) {
	if _, err := e.endpoint.Authorize(header.Get("Authorization"), permission); err != nil {
		return searchCall{}, err
	}
	call := searchCall{search: e.endpoint.callers.Lent.GetPluginSearch()}
	if call.search == nil || !needsVault {
		return call, nil
	}
	vault, err := e.endpoint.getVault(header)
	if err != nil || vault == nil {
		return call, err
	}
	call.vault, call.hasVault = vault.Switch.GetVault()
	return call, nil
}

func (c searchCall) getRefusal(needsVault bool) v1.HostSearchErrorCode {
	switch {
	case c.search == nil:
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL
	case needsVault && !c.hasVault:
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT
	}
	return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED
}

func answerSearch(ctx context.Context, err error) (v1.HostSearchErrorCode, error) {
	if err == nil {
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, plugin.ErrNoModel):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL, nil
	case errors.Is(err, plugin.ErrRecipeMismatch):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH, nil
	case errors.Is(err, plugin.ErrInvalidFloor):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR, nil
	case errors.Is(err, plugin.ErrLimitExceeded):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED, nil
	case errors.Is(err, plugin.ErrEmbeddingFailed):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED, nil
	case errors.Is(err, plugin.ErrInvalidQuery):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_QUERY, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

func (e *searchEndpoint) GetEmbeddingState(
	ctx context.Context, req *connect.Request[v1.GetEmbeddingStateRequest],
) (*connect.Response[v1.GetEmbeddingStateResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.GetEmbeddingStateResponse{Error: code}), nil
	}
	state, err := call.search.GetEmbeddingState(ctx, call.vault)
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetEmbeddingStateResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetEmbeddingStateResponse{
		Recipe:        state.Recipe,
		Model:         state.Model,
		FloorHint:     state.FloorHint,
		ChunkCount:    state.ChunkCount,
		EmbeddedCount: state.EmbeddedCount,
		IsReady:       state.IsReady(),
	}), nil
}

func (e *searchEndpoint) GetIndexStats(
	ctx context.Context, req *connect.Request[v1.GetIndexStatsRequest],
) (*connect.Response[v1.GetIndexStatsResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.GetIndexStatsResponse{Error: code}), nil
	}
	stats, err := call.search.GetIndexStats(ctx, call.vault)
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetIndexStatsResponse{Error: code}), err
	}
	extensions := make([]*v1.ExtensionStats, len(stats.Extensions))
	for i, one := range stats.Extensions {
		extensions[i] = &v1.ExtensionStats{Extension: one.Extension, SourceCount: one.Sources, ChunkCount: one.Chunks}
	}
	return connect.NewResponse(&v1.GetIndexStatsResponse{
		Recipe:        stats.Recipe,
		Model:         stats.Model,
		Dimensions:    stats.Dimensions,
		ChunkCount:    stats.Chunks,
		VectorCount:   stats.Vectors,
		SourceCount:   stats.Sources,
		Extensions:    extensions,
		DatabaseBytes: stats.DatabaseBytes,
		VectorBytes:   stats.VectorBytes,
	}), nil
}

func (e *searchEndpoint) EmbedTexts(
	ctx context.Context, req *connect.Request[v1.EmbedTextsRequest],
) (*connect.Response[v1.EmbedTextsResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, false)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(false); code != 0 {
		return connect.NewResponse(&v1.EmbedTextsResponse{Error: code}), nil
	}
	embedded, err := call.search.EmbedTexts(ctx, req.Msg.GetTexts())
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.EmbedTextsResponse{Error: code}), err
	}
	vectors := make([]*v1.FloatVector, len(embedded.Vectors))
	for i, one := range embedded.Vectors {
		vectors[i] = &v1.FloatVector{Values: one}
	}
	return connect.NewResponse(&v1.EmbedTextsResponse{Recipe: embedded.Recipe, Vectors: vectors}), nil
}

func (e *searchEndpoint) SampleSimilarities(
	ctx context.Context, req *connect.Request[v1.SampleSimilaritiesRequest],
) (*connect.Response[v1.SampleSimilaritiesResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.SampleSimilaritiesResponse{Error: code}), nil
	}
	queries := make([]plugin.SimilarityQuery, len(req.Msg.GetQueries()))
	for i, one := range req.Msg.GetQueries() {
		queries[i] = plugin.SimilarityQuery{Path: one.GetPath(), Vector: one.GetVector().GetValues()}
	}
	sample, err := call.search.SampleSimilarities(ctx, call.vault, req.Msg.GetRecipe(), queries)
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.SampleSimilaritiesResponse{Error: code}), err
	}
	rows := make([]*v1.SimilarityRow, len(sample.Rows))
	for i, one := range sample.Rows {
		rows[i] = &v1.SimilarityRow{Values: one}
	}
	return connect.NewResponse(&v1.SampleSimilaritiesResponse{
		PassageCount: int32(sample.PassageCount),
		SourceCount:  int32(sample.SourceCount),
		Rows:         rows,
	}), nil
}

func (e *searchEndpoint) GetBestSimilarity(
	ctx context.Context, req *connect.Request[v1.GetBestSimilarityRequest],
) (*connect.Response[v1.GetBestSimilarityResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.GetBestSimilarityResponse{Error: code}), nil
	}
	best, err := call.search.GetBestSimilarity(ctx, call.vault, req.Msg.GetRecipe(), req.Msg.GetQuery())
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetBestSimilarityResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetBestSimilarityResponse{Similarity: best}), nil
}

func (e *searchEndpoint) SearchPassages(
	ctx context.Context, req *connect.Request[v1.HostSearchServiceSearchPassagesRequest],
) (*connect.Response[v1.HostSearchServiceSearchPassagesResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.HostSearchServiceSearchPassagesResponse{Error: code}), nil
	}
	kinds := make([]domain.SourceKind, len(req.Msg.GetKinds()))
	for i, kind := range req.Msg.GetKinds() {
		switch one := domain.SourceKind(kind); one {
		case domain.KindNote, domain.KindFile:
			kinds[i] = one
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid source kind %q: use %q or %q", kind, domain.KindNote, domain.KindFile))
		}
	}
	found, err := call.search.SearchPassages(ctx, call.vault, plugin.PassageQuery{
		Query: req.Msg.GetQuery(), Limit: int(req.Msg.GetLimit()), Kinds: kinds, PathPrefix: req.Msg.GetPathPrefix(),
	})
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostSearchServiceSearchPassagesResponse{Error: code}), err
	}
	passages, err := writePassages(found)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.HostSearchServiceSearchPassagesResponse{Passages: passages}), nil
}

func writePassages(found []domain.Passage) ([]*v1.Passage, error) {
	passages := make([]*v1.Passage, len(found))
	for i, one := range found {
		end := one.Start + one.Length
		if one.Start < 0 || one.Length < 0 || end > math.MaxInt32 || one.Line < 0 || one.Line > math.MaxInt32 {
			return nil, connect.NewError(connect.CodeOutOfRange, fmt.Errorf("the passage of %s at %d+%d, line %d, exceeds the 32-bit range", one.Source, one.Start, one.Length, one.Line))
		}
		passages[i] = &v1.Passage{
			Path: one.Source, Text: one.Text, Location: one.Location,
			Range:    &v1.Range{Start: int32(one.Start), End: int32(end)},
			Line:     int32(one.Line),
			Selector: string(one.Selector),
		}
	}
	return passages, nil
}

func (e *searchEndpoint) GetSearchFloor(
	ctx context.Context, req *connect.Request[v1.GetSearchFloorRequest],
) (*connect.Response[v1.GetSearchFloorResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionSearchPolicy, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.GetSearchFloorResponse{Error: code}), nil
	}
	found, err := call.search.GetSearchFloor(ctx, call.vault, req.Msg.GetRecipe())
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetSearchFloorResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.GetSearchFloorResponse{
		Floor:        found.Floor,
		Source:       writeFloorSource(found.Source),
		StoredFloor:  found.Stored,
		HintFloor:    found.Hint,
		DefaultFloor: found.Default,
	}), nil
}

func (e *searchEndpoint) SetSearchFloor(
	ctx context.Context, req *connect.Request[v1.SetSearchFloorRequest],
) (*connect.Response[v1.SetSearchFloorResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionSearchPolicy, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.SetSearchFloorResponse{Error: code}), nil
	}
	code, err := answerSearch(ctx, call.search.SetSearchFloor(ctx, call.vault, req.Msg.GetRecipe(), req.Msg.GetFloor()))
	return connect.NewResponse(&v1.SetSearchFloorResponse{Error: code}), err
}

func writeFloorSource(source plugin.FloorSource) v1.SearchFloorSource {
	switch source {
	case plugin.FloorStored:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_STORED
	case plugin.FloorModelHint:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_MODEL_HINT
	case plugin.FloorDefault:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_DEFAULT
	}
	return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_UNSPECIFIED
}
