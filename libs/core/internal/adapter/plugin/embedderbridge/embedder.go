package embedderbridge

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Host provides plugin resolution and activation for embeddings.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
}

// Embedder is the embedding provider seen as an embeddingport.Embedder.
type Embedder struct {
	host        pluginport.PluginActivator
	pluginID    string
	service     bridge.Service[numenv1connect.EmbeddingServiceClient]
	lease       *bridge.Lease
	mu          sync.Mutex
	model       embeddingport.EmbeddingModel
	concurrency int
}

// Open activates the plugin that declares the embedding capability and reads its model.
func Open(ctx context.Context, host Host) (*Embedder, error) {
	plugin, err := host.ResolveCapability(bridge.CapabilityEmbedding)
	if err != nil {
		return nil, err
	}

	client, lease, err := bridge.Open(ctx, host, plugin.Manifest.ID, numenv1connect.NewEmbeddingServiceClient)
	if err != nil {
		return nil, err
	}

	model, concurrency, err := readClientModel(ctx, plugin.Manifest.ID, client)
	if err != nil {
		_ = lease.Close()
		return nil, err
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

func readClientModel(
	ctx context.Context,
	pluginID string,
	client numenv1connect.EmbeddingServiceClient,
) (embeddingport.EmbeddingModel, int, error) {
	modelCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	resp, err := client.GetEmbeddingModel(modelCtx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		return embeddingport.EmbeddingModel{}, 0, fmt.Errorf("get embedding model %s: %w", pluginID, err)
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
	return model, concurrency, nil
}

// GetModel returns the active model of the embedding plugin.
func (b *Embedder) GetModel() embeddingport.EmbeddingModel {
	if client, call, err := b.service.Open(context.Background()); err == nil {
		model, concurrency, readErr := readClientModel(context.Background(), b.pluginID, client)
		_ = call.Close()
		if readErr == nil {
			b.mu.Lock()
			b.model = model
			b.concurrency = concurrency
			b.mu.Unlock()
			return model
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
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
	b.mu.Lock()
	concurrency := b.concurrency
	b.mu.Unlock()
	if concurrency <= 0 {
		return 1
	}
	return concurrency
}

// Close releases the active reference count for the embedding plugin in the host supervisor.
func (b *Embedder) Close() error {
	return b.lease.Close()
}
