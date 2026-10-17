package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoEmbedderPlugin = errors.New("no active embedding plugin found")
)

// EmbedderPluginHost provides plugin resolution and activation for embeddings.
type EmbedderPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// EmbedderBridge connects port.Embedder calls to an out-of-process embedding plugin via Connect-RPC.
type EmbedderBridge struct {
	host        port.PluginActivator
	pluginID    string
	client      numenv1connect.EmbeddingServiceClient
	lease       *pluginLease
	concurrency int
}

// OpenEmbedderBridge activates or connects to the embedding plugin and returns a port.Embedder.
func OpenEmbedderBridge(ctx context.Context, host EmbedderPluginHost) (*EmbedderBridge, error) {
	plugin, ok := host.FindPluginByCapability("embedding")
	if !ok {
		return nil, errNoEmbedderPlugin
	}

	client, lease, err := openPlugin(ctx, host, plugin.Manifest.ID, numenv1connect.NewEmbeddingServiceClient)
	if err != nil {
		return nil, err
	}

	concurrency := 1
	modelReq := connect.NewRequest(&v1.GetEmbeddingModelRequest{})
	if resp, err := client.GetEmbeddingModel(ctx, modelReq); err == nil && resp.Msg.GetConcurrency() > 0 {
		concurrency = int(resp.Msg.GetConcurrency())
	}

	return &EmbedderBridge{
		host:        host,
		pluginID:    plugin.Manifest.ID,
		client:      client,
		lease:       lease,
		concurrency: concurrency,
	}, nil
}

// GetModel queries the embedding plugin for its model specification.
func (b *EmbedderBridge) GetModel() port.EmbeddingModel {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
	if b.concurrency <= 0 {
		return 1
	}
	return b.concurrency
}

// Close releases the active reference count for the embedding plugin in the host supervisor.
func (b *EmbedderBridge) Close() error {
	return b.lease.Close()
}
