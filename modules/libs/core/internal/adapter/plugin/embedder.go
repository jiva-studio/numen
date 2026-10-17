package plugin

import (
	"context"
	"errors"
	"fmt"

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
	model       port.EmbeddingModel
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

	modelCtx, cancel := context.WithTimeout(ctx, modelInfoTimeout)
	defer cancel()
	resp, err := client.GetEmbeddingModel(modelCtx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("get embedding model %s: %w", plugin.Manifest.ID, err)
	}
	concurrency := 1
	if resp.Msg.GetConcurrency() > 0 {
		concurrency = int(resp.Msg.GetConcurrency())
	}

	model := port.EmbeddingModel{
		Name:       resp.Msg.GetName(),
		Dimensions: int(resp.Msg.GetDimensions()),
		MaxTokens:  int(resp.Msg.GetMaxTokens()),
		Pooling:    resp.Msg.GetPooling(),
		From:       resp.Msg.GetFrom(),
	}

	return &EmbedderBridge{
		host:        host,
		pluginID:    plugin.Manifest.ID,
		client:      client,
		lease:       lease,
		model:       model,
		concurrency: concurrency,
	}, nil
}

// GetModel returns the model the embedding plugin answered with when the bridge was opened.
func (b *EmbedderBridge) GetModel() port.EmbeddingModel {
	return b.model
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
