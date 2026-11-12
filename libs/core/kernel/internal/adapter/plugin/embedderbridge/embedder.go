package embedderbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Host provides plugin resolution and activation for embeddings.
type Host interface {
	bridge.Activator
	ResolveCapability(capability string) (manifest.Discovered, error)
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}

// Embedder is the embedding provider seen as an embeddingport.Embedder.
type Embedder struct {
	host        bridge.Activator
	pluginID    string
	service     bridge.Service[numenv1connect.EmbeddingServiceClient]
	lease       *bridge.Lease
	mu          sync.Mutex
	model       embedding.Model
	concurrency int
	floorHint   float64

	announceMu  sync.Mutex
	changeMu    sync.Mutex
	announced   embedding.Model
	listeners   []func()
	onReadError func(error)
	stopWatch   func()
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
) (embedding.Model, int, float64, error) {
	modelCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	resp, err := client.GetEmbeddingModel(modelCtx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		return embedding.Model{}, 0, 0, fmt.Errorf("get embedding model %s: %w", pluginID, err)
	}
	concurrency := 1
	if resp.Msg.GetConcurrency() > 0 {
		concurrency = int(resp.Msg.GetConcurrency())
	}
	model := embedding.Model{
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
func (b *Embedder) GetModel() embedding.Model {
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

// OnModelReadError registers fn to receive the outcome of a read of the model that follows a configuration of the plugin:
// the error of a read that fails, and nil for a read that succeeds. After a failed read the model and the listeners stay as
// they were, and the next read that succeeds announces the change.
func (b *Embedder) OnModelReadError(fn func(error)) {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	b.onReadError = fn
}

// readFailure is the failure to report for a read of the model that failed. A plugin whose settings do not allow
// the read yet, such as a source that is a connection with no row chosen, is not failing: it reports nothing.
func readFailure(pluginID string, err error) error {
	// A plugin whose settings do not allow the read yet, such as a source that is a connection with no row
	// chosen, is not failing: it reports nothing.
	if connect.CodeOf(err) == connect.CodeFailedPrecondition {
		return nil
	}
	return fmt.Errorf("read model of embedding plugin %s: %w", pluginID, err)
}

func (b *Embedder) reportReadError(err error) {
	if b.onReadError != nil {
		b.onReadError(err)
	}
}

func (b *Embedder) announceModelChange() {
	b.announceMu.Lock()
	defer b.announceMu.Unlock()
	for _, fn := range b.readModelChange() {
		fn()
	}
}

func (b *Embedder) readModelChange() []func() {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), transport.ModelRefreshTimeout)
	defer cancel()
	client, call, err := b.service.Open(ctx)
	if err != nil {
		b.reportReadError(fmt.Errorf("open embedding plugin %s: %w", b.pluginID, err))
		return nil
	}
	model, concurrency, floorHint, readErr := readClientModel(ctx, b.pluginID, client)
	if err := errors.Join(readErr, call.Close()); err != nil {
		b.reportReadError(readFailure(b.pluginID, err))
		return nil
	}
	b.mu.Lock()
	b.model, b.concurrency, b.floorHint = model, concurrency, floorHint
	b.mu.Unlock()
	b.reportReadError(nil)
	if model == b.announced {
		return nil
	}
	b.announced = model
	return slices.Clone(b.listeners)
}
