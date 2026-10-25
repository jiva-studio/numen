package embedderbridge

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Host provides plugin resolution and activation for embeddings.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
}

// Embedder connects port.Embedder calls to an out-of-process embedding plugin via Connect-RPC.
type Embedder struct {
	host        pluginport.PluginActivator
	pluginID    string
	service     bridge.Service[numenv1connect.EmbeddingServiceClient]
	lease       *bridge.Lease
	model       embeddingport.EmbeddingModel
	concurrency int
}

// Open activates or connects to the embedding plugin and returns a port.Embedder.
func Open(ctx context.Context, host Host) (*Embedder, error) {
	plugin, err := host.ResolveCapability(bridge.CapabilityEmbedding)
	if err != nil {
		return nil, err
	}

	client, lease, err := bridge.Open(ctx, host, plugin.Manifest.ID, numenv1connect.NewEmbeddingServiceClient)
	if err != nil {
		return nil, err
	}

	modelCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
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

	model := embeddingport.EmbeddingModel{
		Name:       resp.Msg.GetName(),
		Dimensions: int(resp.Msg.GetDimensions()),
		MaxTokens:  int(resp.Msg.GetMaxTokens()),
		Pooling:    resp.Msg.GetPooling(),
		From:       resp.Msg.GetFrom(),
	}

	return &Embedder{
		host:        host,
		pluginID:    plugin.Manifest.ID,
		service:     bridge.Service[numenv1connect.EmbeddingServiceClient]{Host: host, PluginID: plugin.Manifest.ID, NewClient: numenv1connect.NewEmbeddingServiceClient},
		lease:       lease,
		model:       model,
		concurrency: concurrency,
	}, nil
}

// GetModel returns the model the embedding plugin answered with when the bridge was opened.
func (b *Embedder) GetModel() embeddingport.EmbeddingModel {
	return b.model
}

// Embed sends texts to the embedding plugin process and decodes returned vectors.
func (b *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	req := connect.NewRequest(&v1.EmbedBatchRequest{
		Texts: texts,
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.EmbedBatch(ctx, req)
	if err != nil {
		return nil, transport.ReadRPCError("embed batch rpc", err)
	}

	protoVectors := resp.Msg.GetVectors()
	vectors := make([][]float32, len(protoVectors))
	for i, pv := range protoVectors {
		vectors[i] = pv.GetValues()
	}
	return vectors, nil
}

// Concurrency returns how many batches the embedder runs in parallel.
func (b *Embedder) Concurrency() int {
	if b.concurrency <= 0 {
		return 1
	}
	return b.concurrency
}

// Close releases the active reference count for the embedding plugin in the host supervisor.
func (b *Embedder) Close() error {
	return b.lease.Close()
}
