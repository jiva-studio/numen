// Package searchsource serves a search source over SearchSourceService: a participant of the search of the
// core. The core hands the source the texts of a vault and the place its vectors are kept in, keeps the
// vectors the source answers there, and merges the chunks the source answers to a query with the answers
// of the other participants.
package searchsource

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Description is what the core needs to keep the data of the source.
//
//numen:api
type Description struct {
	// Name is the text a person knows the data of the source by.
	Name string
	// Recipe is opaque to the core: two vectors made under different recipes cannot be compared, and a
	// vector kept under another recipe is stale.
	Recipe string
	// Width is how many values one vector of the source holds.
	Width int32
	// ChunkTokens is the most tokens one chunk holds, which the core cuts at; zero leaves the core default.
	ChunkTokens int32
	// MinTextLength is the fewest bytes a chunk holds to be handed to the source; zero hands every chunk.
	MinTextLength int32
	// Concurrency is how many calls of IndexTexts the source answers at once.
	Concurrency int32
	// FloorHint is the similarity a chunk starts out having to reach, zero where the source knows none.
	FloorHint float64
}

// Place is where the vectors of the source are kept: the vault and the recipe. A source passes it back to
// the host unchanged.
//
//numen:api
type Place struct {
	VaultID string
	Recipe  string
}

// Text is one text the core hands the source: the chunk it was cut as, the path of its source relative to
// the vault, and the text version it was cut from, empty for a note.
//
//numen:api
type Text struct {
	ChunkID   string
	Path      string
	VersionID string
	Text      string
}

// Query is what a search asks the source. A Floor of zero applies the floor of the vault.
//
//numen:api
type Query struct {
	Text       string
	Limit      int
	Kinds      []string
	PathPrefix string
	Floor      float64
}

// Match is a chunk the core handed the source, and how well it answers a query.
//
//numen:api
type Match struct {
	ChunkID string
	Score   float64
}

// Handler is what a plugin implements to serve a search source.
type Handler interface {
	// Describe returns the description of the source. A failure the settings cause is a failed
	// precondition whose message a person reads.
	Describe(ctx context.Context) (Description, error)
	// IndexTexts returns one vector per text, in order, of the width of the description.
	IndexTexts(ctx context.Context, place Place, texts []Text) ([][]float32, error)
	// SearchChunks returns the chunks of the place that answer the query, best first.
	SearchChunks(ctx context.Context, place Place, q Query) ([]Match, error)
	// EmbedQueries returns one vector per query, in order, as the source compares a query with its place.
	EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
}

type adapter struct {
	numenv1connect.UnimplementedSearchSourceServiceHandler
	h Handler
}

func (a *adapter) DescribeSearchSource(
	ctx context.Context,
	_ *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	d, err := a.h.Describe(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(writeDescription(d)), nil
}

func (a *adapter) IndexTexts(
	ctx context.Context,
	req *connect.Request[v1.IndexTextsRequest],
) (*connect.Response[v1.IndexTextsResponse], error) {
	texts := make([]Text, len(req.Msg.GetTexts()))
	for i, one := range req.Msg.GetTexts() {
		texts[i] = Text{ChunkID: one.GetChunkId(), Path: one.GetPath(), VersionID: one.GetVersionId(), Text: one.GetText()}
	}
	vectors, err := a.h.IndexTexts(ctx, readPlace(req.Msg.GetPlace()), texts)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.IndexTextsResponse{Vectors: writeVectors(vectors)}), nil
}

func (a *adapter) SearchChunks(
	ctx context.Context,
	req *connect.Request[v1.SearchSourceServiceSearchChunksRequest],
) (*connect.Response[v1.SearchSourceServiceSearchChunksResponse], error) {
	msg := req.Msg
	matches, err := a.h.SearchChunks(ctx, readPlace(msg.GetPlace()), Query{
		Text: msg.GetQuery(), Limit: int(msg.GetLimit()), Kinds: msg.GetKinds(), PathPrefix: msg.GetPathPrefix(), Floor: msg.GetFloor(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ChunkMatch, len(matches))
	for i, one := range matches {
		out[i] = &v1.ChunkMatch{ChunkId: one.ChunkID, Score: one.Score}
	}
	return connect.NewResponse(&v1.SearchSourceServiceSearchChunksResponse{Matches: out}), nil
}

func (a *adapter) EmbedQueries(
	ctx context.Context,
	req *connect.Request[v1.EmbedQueriesRequest],
) (*connect.Response[v1.EmbedQueriesResponse], error) {
	vectors, err := a.h.EmbedQueries(ctx, req.Msg.GetQueries())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.EmbedQueriesResponse{Vectors: writeVectors(vectors)}), nil
}

// NewHandler returns the path and the handler of the search source service over h.
func NewHandler(h Handler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewSearchSourceServiceHandler(&adapter{h: h}, opts...)
}

// Mount registers the search source service of h on s.
func Mount(s *server.Server, h Handler) {
	s.Register(NewHandler(h, s.HandlerOptions()...))
}

func readPlace(place *v1.SearchPlace) Place {
	return Place{VaultID: place.GetVaultId(), Recipe: place.GetRecipe()}
}

func writeDescription(d Description) *v1.DescribeSearchSourceResponse {
	return &v1.DescribeSearchSourceResponse{
		Name:          d.Name,
		Recipe:        d.Recipe,
		Width:         d.Width,
		ChunkTokens:   d.ChunkTokens,
		MinTextLength: d.MinTextLength,
		Concurrency:   d.Concurrency,
		FloorHint:     d.FloorHint,
	}
}

func writeVectors(dense [][]float32) []*v1.FloatVector {
	vectors := make([]*v1.FloatVector, len(dense))
	for i, values := range dense {
		vectors[i] = &v1.FloatVector{Values: values}
	}
	return vectors
}
