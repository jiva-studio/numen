// Package embedding serves an embedding model over EmbeddingService.
package embedding

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Role is what a text is embedded for. A model that tells a question from the passage that answers
// it reads the two with different prefixes. The zero value is RoleDocument.
type Role int

// The roles of a text.
const (
	RoleDocument Role = iota
	RoleQuery
)

// ModelInfo describes the embedding model served by the plugin.
//
//numen:api
type ModelInfo struct {
	Name        string
	Dimensions  int32
	MaxTokens   int32
	Pooling     string
	From        string
	Concurrency int32
	FloorHint   float64

	// MinChunkLength is the fewest bytes a piece of text holds to be given a vector; 0 embeds every piece.
	MinChunkLength int32

	// ChunkTokens is the most tokens one chunk may hold; 0 names none.
	ChunkTokens int32

	// Handling is a short hash of what the plugin does to a text before the model reads it and to the output
	// after. It is part of the identity of the vectors: a change in it makes the vectors of the model stale.
	Handling string
}

func (m ModelInfo) GetName() string          { return m.Name }
func (m ModelInfo) GetDimensions() int32     { return m.Dimensions }
func (m ModelInfo) GetMaxTokens() int32      { return m.MaxTokens }
func (m ModelInfo) GetPooling() string       { return m.Pooling }
func (m ModelInfo) GetFrom() string          { return m.From }
func (m ModelInfo) GetConcurrency() int32    { return m.Concurrency }
func (m ModelInfo) GetFloorHint() float64    { return m.FloorHint }
func (m ModelInfo) GetMinChunkLength() int32 { return m.MinChunkLength }
func (m ModelInfo) GetChunkTokens() int32    { return m.ChunkTokens }
func (m ModelInfo) GetHandling() string      { return m.Handling }

// Handler serves embedding model metadata and batches of vectors.
type Handler interface {
	GetModelInfo(ctx context.Context) (ModelInfo, error)
	EmbedTexts(ctx context.Context, role Role, texts []string) ([][]float32, error)
}

type adapter struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
	h Handler
}

func (a *adapter) GetEmbeddingModel(
	ctx context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	info, err := a.h.GetModelInfo(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(writeModelInfo(info)), nil
}

func (a *adapter) EmbedBatch(
	ctx context.Context,
	req *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	dense, err := a.h.EmbedTexts(ctx, readRole(req.Msg.GetRole()), req.Msg.GetTexts())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(writeVectors(dense)), nil
}

func NewHandler(h Handler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewEmbeddingServiceHandler(&adapter{h: h}, opts...)
}

// Mount registers the embedding Handler on the plugin Server.
func Mount(s *server.Server, h Handler) {
	s.Register(NewHandler(h, s.HandlerOptions()...))
}

func readModelInfo(res *v1.GetEmbeddingModelResponse) ModelInfo {
	return ModelInfo{
		Name:           res.GetName(),
		Dimensions:     res.GetDimensions(),
		MaxTokens:      res.GetMaxTokens(),
		Pooling:        res.GetPooling(),
		From:           res.GetFrom(),
		Concurrency:    res.GetConcurrency(),
		FloorHint:      res.GetFloorHint(),
		MinChunkLength: res.GetMinChunkLength(),
		ChunkTokens:    res.GetChunkTokens(),
		Handling:       res.GetHandling(),
	}
}

func writeModelInfo(info ModelInfo) *v1.GetEmbeddingModelResponse {
	return &v1.GetEmbeddingModelResponse{
		Name:           info.Name,
		Dimensions:     info.Dimensions,
		MaxTokens:      info.MaxTokens,
		Pooling:        info.Pooling,
		From:           info.From,
		Concurrency:    info.Concurrency,
		FloorHint:      info.FloorHint,
		MinChunkLength: info.MinChunkLength,
		ChunkTokens:    info.ChunkTokens,
		Handling:       info.Handling,
	}
}

func readRole(role v1.EmbedRole) Role {
	if role == v1.EmbedRole_EMBED_ROLE_QUERY {
		return RoleQuery
	}
	// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
	return RoleDocument
}

func writeRole(role Role) v1.EmbedRole {
	if role == RoleQuery {
		return v1.EmbedRole_EMBED_ROLE_QUERY
	}
	return v1.EmbedRole_EMBED_ROLE_DOCUMENT
}

func readVectors(res *v1.EmbedBatchResponse) [][]float32 {
	vectors := make([][]float32, len(res.GetVectors()))
	for i, one := range res.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return vectors
}

func writeVectors(dense [][]float32) *v1.EmbedBatchResponse {
	vectors := make([]*v1.FloatVector, len(dense))
	for i, values := range dense {
		vectors[i] = &v1.FloatVector{Values: values}
	}
	return &v1.EmbedBatchResponse{Vectors: vectors}
}

// Client calls the embedding service of a plugin.
type Client struct {
	rpc numenv1connect.EmbeddingServiceClient
}

// NewClient creates a client of the embedding service served at baseURL.
func NewClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) *Client {
	return &Client{rpc: numenv1connect.NewEmbeddingServiceClient(httpClient, baseURL, opts...)}
}

// GetModelInfo returns the description of the model the plugin serves.
func (c *Client) GetModelInfo(ctx context.Context) (ModelInfo, error) {
	res, err := c.rpc.GetEmbeddingModel(ctx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		return ModelInfo{}, err
	}
	return readModelInfo(res.Msg), nil
}

// EmbedTexts returns one vector per text, in order, made for the role.
func (c *Client) EmbedTexts(ctx context.Context, role Role, texts []string) ([][]float32, error) {
	res, err := c.rpc.EmbedBatch(ctx, connect.NewRequest(&v1.EmbedBatchRequest{Texts: texts, Role: writeRole(role)}))
	if err != nil {
		return nil, err
	}
	return readVectors(res.Msg), nil
}
