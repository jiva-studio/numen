// Package embedding serves an embedding model over EmbeddingService.
package embedding

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// ModelInfo describes the embedding model served by the plugin.
type ModelInfo struct {
	Name        string
	Dimensions  int32
	MaxTokens   int32
	Pooling     string
	From        string
	Concurrency int32
}

func (m ModelInfo) GetName() string       { return m.Name }
func (m ModelInfo) GetDimensions() int32  { return m.Dimensions }
func (m ModelInfo) GetMaxTokens() int32   { return m.MaxTokens }
func (m ModelInfo) GetPooling() string    { return m.Pooling }
func (m ModelInfo) GetFrom() string       { return m.From }
func (m ModelInfo) GetConcurrency() int32 { return m.Concurrency }

// Handler serves embedding model metadata and batch embeddings.
type Handler interface {
	GetModelInfo(ctx context.Context) (ModelInfo, error)
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)
}

type adapter struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
	h Handler
}

func (a *adapter) GetEmbeddingModel(
	ctx context.Context,
	_ *connect.Request[numenv1.GetEmbeddingModelRequest],
) (*connect.Response[numenv1.GetEmbeddingModelResponse], error) {
	info, err := a.h.GetModelInfo(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        info.Name,
		Dimensions:  info.Dimensions,
		MaxTokens:   info.MaxTokens,
		Pooling:     info.Pooling,
		From:        info.From,
		Concurrency: info.Concurrency,
	}), nil
}

func (a *adapter) EmbedBatch(
	ctx context.Context,
	req *connect.Request[numenv1.EmbedBatchRequest],
) (*connect.Response[numenv1.EmbedBatchResponse], error) {
	dense, err := a.h.EmbedTexts(ctx, req.Msg.GetTexts())
	if err != nil {
		return nil, err
	}
	vectors := make([]*numenv1.FloatVector, len(dense))
	for i, v := range dense {
		vectors[i] = &numenv1.FloatVector{Values: v}
	}
	return connect.NewResponse(&numenv1.EmbedBatchResponse{Vectors: vectors}), nil
}

// NewHandler creates the Connect-RPC handler for an embedding Handler.
func NewHandler(h Handler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewEmbeddingServiceHandler(&adapter{h: h}, opts...)
}

// Mount registers the embedding Handler on the plugin Server.
func Mount(s *server.Server, h Handler) {
	s.Register(NewHandler(h, s.HandlerOptions()...))
}

type EmbedBatchRequest = numenv1.EmbedBatchRequest

type EmbedBatchResponse = numenv1.EmbedBatchResponse

type EmbeddingServiceClient = numenv1connect.EmbeddingServiceClient

// NewEmbeddingServiceClient constructs a client for EmbeddingService.
func NewEmbeddingServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) EmbeddingServiceClient {
	return numenv1connect.NewEmbeddingServiceClient(httpClient, baseURL, opts...)
}
