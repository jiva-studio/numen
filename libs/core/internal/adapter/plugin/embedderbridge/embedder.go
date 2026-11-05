package embedderbridge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

// Host provides plugin resolution and activation for embeddings.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
	WatchPluginConfig(pluginID string, fn func()) (stop func())
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
	floorHint   float64

	// changeMu orders the reads that follow a configuration of the plugin.
	changeMu sync.Mutex
	// announced is the model the listeners were last notified of. A read by GetModel does not change it.
	announced embeddingport.EmbeddingModel
	listeners []func()
	stopWatch func()
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

	model, concurrency, floorHint, err := readClientModel(ctx, plugin.Manifest.ID, client)
	if err != nil {
		return nil, errors.Join(err, lease.Close())
	}

	b := &Embedder{
		host:        host,
		pluginID:    plugin.Manifest.ID,
		service:     bridge.Service[numenv1connect.EmbeddingServiceClient]{Host: host, PluginID: plugin.Manifest.ID, NewClient: numenv1connect.NewEmbeddingServiceClient},
		lease:       lease,
		model:       model,
		concurrency: concurrency,
		floorHint:   floorHint,
		announced:   model,
	}
	b.stopWatch = host.WatchPluginConfig(plugin.Manifest.ID, b.announceModelChange)
	return b, nil
}

func readClientModel(
	ctx context.Context,
	pluginID string,
	client numenv1connect.EmbeddingServiceClient,
) (embeddingport.EmbeddingModel, int, float64, error) {
	modelCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	resp, err := client.GetEmbeddingModel(modelCtx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		return embeddingport.EmbeddingModel{}, 0, 0, fmt.Errorf("get embedding model %s: %w", pluginID, err)
	}
	concurrency := 1
	if resp.Msg.GetConcurrency() > 0 {
		concurrency = int(resp.Msg.GetConcurrency())
	}
	model := embeddingport.EmbeddingModel{
		Name:           resp.Msg.GetName(),
		Dimensions:     int(resp.Msg.GetDimensions()),
		MaxTokens:      int(resp.Msg.GetMaxTokens()),
		Pooling:        resp.Msg.GetPooling(),
		From:           resp.Msg.GetFrom(),
		Handling:       resp.Msg.GetHandling(),
		MinChunkLength: int(max(resp.Msg.GetMinChunkLength(), 0)),
		ChunkTokens:    int(max(resp.Msg.GetChunkTokens(), 0)),
	}
	return model, concurrency, max(resp.Msg.GetFloorHint(), 0), nil
}

// GetModel returns the active model of the embedding plugin.
func (b *Embedder) GetModel() embeddingport.EmbeddingModel {
	ctx, cancel := context.WithTimeout(context.Background(), transport.ModelRefreshTimeout)
	defer cancel()
	if client, call, err := b.service.Open(ctx); err == nil {
		model, concurrency, floorHint, readErr := readClientModel(ctx, b.pluginID, client)
		if errors.Join(readErr, call.Close()) == nil {
			b.mu.Lock()
			b.model = model
			b.concurrency = concurrency
			b.floorHint = floorHint
			b.mu.Unlock()
			return model
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.model
}

// GetFloorHint returns the starting floor the embedding plugin reported with the
// model on the last read, and zero where it reported none.
func (b *Embedder) GetFloorHint(_ context.Context) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.floorHint
}

// Embed sends texts to the embedding plugin process as documents and decodes returned vectors.
func (b *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return b.embed(ctx, v1.EmbedRole_EMBED_ROLE_DOCUMENT, texts)
}

// EmbedQueries sends texts to the embedding plugin process as queries and decodes returned vectors.
func (b *Embedder) EmbedQueries(ctx context.Context, texts []string) ([][]float32, error) {
	return b.embed(ctx, v1.EmbedRole_EMBED_ROLE_QUERY, texts)
}

func (b *Embedder) embed(ctx context.Context, role v1.EmbedRole, texts []string) (vectors [][]float32, err error) {
	if len(texts) == 0 {
		return nil, nil
	}

	req := connect.NewRequest(&v1.EmbedBatchRequest{
		Texts: texts,
		Role:  role,
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.EmbedBatch(ctx, req)
	if err != nil {
		return nil, transport.ReadRPCError("embed batch rpc", err)
	}

	protoVectors := resp.Msg.GetVectors()
	vectors = make([][]float32, len(protoVectors))
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
	b.stopWatch()
	return b.lease.Close()
}

// OnModelChange registers fn to run when the plugin reports a model unlike the last one announced:
// another identity, handling, minimum chunk length or chunk size. The plugin is read
// after it has been given its configuration, so a setting of another plugin never runs fn.
func (b *Embedder) OnModelChange(fn func()) {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	b.listeners = append(b.listeners, fn)
}

// OnModelReadError registers fn to receive the error of a read of the model that follows a configuration of the plugin and fails.
func (b *Embedder) OnModelReadError(fn func(error)) {}

// announceModelChange reads the model and runs the listeners when it differs from the announced one.
func (b *Embedder) announceModelChange() {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), transport.ModelRefreshTimeout)
	defer cancel()
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return
	}
	model, concurrency, floorHint, readErr := readClientModel(ctx, b.pluginID, client)
	if errors.Join(readErr, call.Close()) != nil {
		return
	}
	b.mu.Lock()
	b.model, b.concurrency, b.floorHint = model, concurrency, floorHint
	b.mu.Unlock()
	if model == b.announced {
		return
	}
	b.announced = model
	for _, fn := range b.listeners {
		fn()
	}
}
