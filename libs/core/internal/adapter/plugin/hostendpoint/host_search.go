package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// searchEndpoint answers HostSearchService for the plugins that call the host.
type searchEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostSearchServiceHandler = (*searchEndpoint)(nil)

// searchCall is an authorized search call: the search index and the vault it works. Search is nil
// where the host has no search index.
type searchCall struct {
	search pluginport.PluginSearch
	vault  domain.Vault
	// hasVault is whether the host has a vault to offer the call.
	hasVault bool
}

// begin authorizes a call that needs the permission and finds the search index. A call that names a
// vault by its Numen-Vault header works that vault, else the one the host shows.
func (e *searchEndpoint) begin(header http.Header, permission string, needsVault bool) (searchCall, error) {
	if _, err := e.endpoint.authorize(header.Get("Authorization"), permission); err != nil {
		return searchCall{}, err
	}
	call := searchCall{search: e.endpoint.callers.GetPluginSearch()}
	if call.search == nil || !needsVault {
		return call, nil
	}
	vault, err := e.endpoint.getVault(header)
	if err != nil || vault == nil {
		return call, err
	}
	call.vault, call.hasVault = vault.GetVault()
	return call, nil
}

// getRefusal is the error code a call that cannot be served is answered with, and zero for a call
// that can.
func (c searchCall) getRefusal(needsVault bool) v1.HostSearchErrorCode {
	switch {
	case c.search == nil:
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL
	case needsVault && !c.hasVault:
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT
	}
	return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED
}

// answerSearch sorts the failure of a search call: an answer the protocol names is a code, the end
// of the caller's context is cancelled, and anything else is internal.
func answerSearch(ctx context.Context, err error) (v1.HostSearchErrorCode, error) {
	if err == nil {
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, pluginport.ErrNoModel):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL, nil
	case errors.Is(err, pluginport.ErrRecipeMismatch):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_RECIPE_MISMATCH, nil
	case errors.Is(err, pluginport.ErrInvalidFloor):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_FLOOR, nil
	case errors.Is(err, pluginport.ErrLimitExceeded):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED, nil
	case errors.Is(err, pluginport.ErrEmbeddingFailed):
		return v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_EMBEDDING_FAILED, nil
	case errors.Is(err, pluginport.ErrInvalidQuery):
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
	queries := make([]pluginport.PluginSimilarityQuery, len(req.Msg.GetQueries()))
	for i, one := range req.Msg.GetQueries() {
		queries[i] = pluginport.PluginSimilarityQuery{Path: one.GetPath(), Vector: one.GetVector().GetValues()}
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

func writeFloorSource(source pluginport.PluginFloorSource) v1.SearchFloorSource {
	switch source {
	case pluginport.PluginFloorStored:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_STORED
	case pluginport.PluginFloorModelHint:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_MODEL_HINT
	case pluginport.PluginFloorDefault:
		return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_DEFAULT
	}
	return v1.SearchFloorSource_SEARCH_FLOOR_SOURCE_UNSPECIFIED
}
