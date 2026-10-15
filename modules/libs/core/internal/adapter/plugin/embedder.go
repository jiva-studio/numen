package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoEmbedderPlugin = errors.New("no active embedding plugin found")
)

// EmbedderBridge connects port.Embedder calls to an out-of-process embedding plugin via Connect-RPC.
type EmbedderBridge struct {
	host     *Host
	pluginID string
	client   numenv1connect.EmbeddingServiceClient
}

// OpenEmbedderBridge activates or connects to the embedding plugin and returns a port.Embedder.
func OpenEmbedderBridge(ctx context.Context, host *Host) (*EmbedderBridge, error) {
	plugin, ok := host.FindPluginByCapability("embedding")
	if !ok {
		return nil, errNoEmbedderPlugin
	}

	endpoint, err := host.EnsureActive(ctx, plugin.Manifest.ID)
	if err != nil {
		return nil, fmt.Errorf("ensure active %s: %w", plugin.Manifest.ID, err)
	}

	httpClient := &http.Client{}
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewEmbeddingServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	return &EmbedderBridge{
		host:     host,
		pluginID: plugin.Manifest.ID,
		client:   client,
	}, nil
}

// GetModel queries the embedding plugin for its model specification.
func (b *EmbedderBridge) GetModel() port.EmbeddingModel {
	ctx := context.Background()
	req := connect.NewRequest(&v1.GetEmbeddingModelRequest{})
	resp, err := b.client.GetEmbeddingModel(ctx, req)
	if err != nil {
		return port.EmbeddingModel{
			Name:       "unknown",
			Dimensions: 384,
			MaxTokens:  256,
			Pooling:    "mean",
			From:       "plugin-error",
		}
	}

	return port.EmbeddingModel{
		Name:       resp.Msg.GetName(),
		Dimensions: int(resp.Msg.GetDimensions()),
		MaxTokens:  int(resp.Msg.GetMaxTokens()),
		Pooling:    resp.Msg.GetPooling(),
		From:       resp.Msg.GetFrom(),
	}
}

// Embed sends texts to the embedding plugin process and decodes returned vectors.
func (b *EmbedderBridge) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	req := connect.NewRequest(&v1.EmbedBatchRequest{
		Texts: texts,
	})
	resp, err := b.client.EmbedBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embed batch rpc: %w", err)
	}

	protoVectors := resp.Msg.GetVectors()
	vectors := make([][]float32, len(protoVectors))
	for i, pv := range protoVectors {
		vectors[i] = pv.GetValues()
	}
	return vectors, nil
}

// Concurrency returns how many batches the embedder runs in parallel.
func (b *EmbedderBridge) Concurrency() int {
	ctx := context.Background()
	req := connect.NewRequest(&v1.GetEmbeddingModelRequest{})
	resp, err := b.client.GetEmbeddingModel(ctx, req)
	if err != nil || resp.Msg.GetConcurrency() <= 0 {
		return 1
	}
	return int(resp.Msg.GetConcurrency())
}

// Close releases the active reference count for the embedding plugin in the host supervisor.
func (b *EmbedderBridge) Close() error {
	b.host.Release(b.pluginID)
	return nil
}
