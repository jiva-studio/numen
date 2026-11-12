package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	hostdomain "github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/querycache"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/serving"
)

// notBuilt is what a person reads while no chunk of the vault has a vector.
const notBuilt = "Semantic index not built yet"

// openPlace returns the search of the core index in the vault of the given identity.
type openPlace func(vaultID string) (host.PlaceSearch, error)

// handler answers the calls of the search source service with the model server: it turns texts into the
// vectors the core keeps, and a query into the chunks of the place nearest it.
type handler struct {
	model   *serving.Server
	places  openPlace
	queries *querycache.Cache
}

func newHandler(model *serving.Server, places openPlace) handler {
	return handler{model: model, places: places, queries: querycache.New(querycache.DefaultCapacity)}
}

// Describe describes the vectors the plugin makes with the model it serves.
func (h handler) Describe(ctx context.Context) (searchsource.Description, error) {
	info, err := h.model.GetModelInfo(ctx)
	if err != nil {
		return searchsource.Description{}, getRPCError(err)
	}
	return searchsource.Description{
		Name:          info.Name,
		Recipe:        info.GetRecipe(),
		Width:         int32(info.Dimensions),
		ChunkTokens:   int32(info.ChunkTokens),
		MinTextLength: int32(info.MinChunkLength),
		Concurrency:   int32(info.Concurrency),
		FloorHint:     info.FloorHint,
	}, nil
}

// IndexTexts returns one vector per text, made for the text as a passage that is searched.
func (h handler) IndexTexts(ctx context.Context, _ searchsource.Place, texts []searchsource.Text) ([][]float32, error) {
	plain := make([]string, len(texts))
	for i, one := range texts {
		plain[i] = one.Text
	}
	return h.embed(ctx, model.RoleDocument, plain)
}

// EmbedQueries returns one vector per query, made for the query as the question.
func (h handler) EmbedQueries(ctx context.Context, queries []string) ([][]float32, error) {
	return h.embed(ctx, model.RoleQuery, queries)
}

// SearchPassages embeds the query and asks the core for the chunks of the place nearest it.
func (h handler) SearchPassages(ctx context.Context, place searchsource.Place, q searchsource.Query) ([]searchsource.Match, error) {
	vector, err := h.getQueryVector(ctx, place.Recipe, q.Text)
	if err != nil {
		return nil, err
	}
	places, err := h.places(place.VaultID)
	if err != nil {
		return nil, rpcerror.New(rpcerror.Unavailable, err)
	}
	kinds := make([]hostdomain.SourceKind, len(q.Kinds))
	for i, kind := range q.Kinds {
		kinds[i] = hostdomain.SourceKind(kind)
	}
	found, err := places.FindNearestPassages(ctx, host.NearestQuery{
		Recipe: place.Recipe, Vector: vector, Limit: q.Limit, Kinds: kinds, PathPrefix: q.PathPrefix, Floor: q.Floor,
	})
	if err != nil {
		return nil, fmt.Errorf("find the nearest passages: %w", err)
	}
	matches := make([]searchsource.Match, len(found))
	for i, one := range found {
		matches[i] = searchsource.Match{ChunkID: one.ChunkID, Score: one.Similarity}
	}
	return matches, nil
}

// DescribeCoverage says that the index is not built while no chunk of the vault has a vector.
func (h handler) DescribeCoverage(_ context.Context, _ searchsource.Place, counts searchsource.Counts) (string, error) {
	if counts.Indexed == 0 {
		return notBuilt, nil
	}
	return "", nil
}

func (h handler) getQueryVector(ctx context.Context, recipe, query string) ([]float32, error) {
	if cached, ok := h.queries.Get(recipe, query); ok {
		return cached, nil
	}
	vectors, err := h.embed(ctx, model.RoleQuery, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("the model answered with %d vectors for one query", len(vectors))
	}
	h.queries.Put(recipe, query, vectors[0])
	return vectors[0], nil
}

func (h handler) embed(ctx context.Context, role model.Role, texts []string) ([][]float32, error) {
	vectors, err := h.model.EmbedTexts(ctx, role, texts)
	if err != nil {
		return nil, getRPCError(err)
	}
	return vectors, nil
}

func getRPCError(err error) error {
	if failure, isFailure := errors.AsType[serving.PreconditionError](err); isFailure && failure.Err != nil {
		return rpcerror.New(rpcerror.FailedPrecondition, err)
	}
	return rpcerror.New(rpcerror.Internal, err)
}
