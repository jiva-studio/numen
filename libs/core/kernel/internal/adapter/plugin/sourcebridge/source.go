package sourcebridge

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Host provides plugin resolution and activation for the search source.
type Host interface {
	bridge.Activator
	ResolveCapability(capability string) (manifest.Discovered, error)
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}

// description is what one read of the source answers: its description, and the floor hint, which is no part
// of it.
type description struct {
	source    searchsource.Description
	floorHint float64
}

// Source is the search source plugin seen as a searchsourceport.Source.
type Source struct {
	host     bridge.Activator
	pluginID string
	service  bridge.Service[numenv1connect.SearchSourceServiceClient]
	lease    *bridge.Lease
	mu       sync.Mutex
	current  description

	announceMu  sync.Mutex
	changeMu    sync.Mutex
	announced   searchsource.Description
	listeners   []func()
	onReadError func(error)
	stopWatch   func()
}

// Open activates the plugin that declares the search source capability and reads its description. A
// plugin whose settings do not let it describe a source fails with the sentence the plugin gives.
func Open(ctx context.Context, host Host) (*Source, error) {
	plugin, err := host.ResolveCapability(bridge.CapabilitySearchSource)
	if err != nil {
		return nil, err
	}

	client, lease, err := bridge.Open(ctx, host, plugin.Manifest.ID, numenv1connect.NewSearchSourceServiceClient)
	if err != nil {
		return nil, err
	}

	read, err := readDescription(ctx, client)
	if err != nil {
		return nil, errors.Join(readFailure(plugin.Manifest.ID, err), lease.Close())
	}

	b := &Source{
		host:      host,
		pluginID:  plugin.Manifest.ID,
		service:   bridge.Service[numenv1connect.SearchSourceServiceClient]{Host: host, PluginID: plugin.Manifest.ID, NewClient: numenv1connect.NewSearchSourceServiceClient},
		lease:     lease,
		current:   read,
		announced: read.source,
	}
	b.stopWatch = host.WatchPluginConfig(plugin.Manifest.ID, b.announceDescriptionChange)
	return b, nil
}

func readDescription(ctx context.Context, client numenv1connect.SearchSourceServiceClient) (description, error) {
	describeCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	resp, err := client.DescribeSearchSource(describeCtx, connect.NewRequest(&v1.DescribeSearchSourceRequest{}))
	if err != nil {
		return description{}, err
	}
	concurrency := 1
	if resp.Msg.GetConcurrency() > 0 {
		concurrency = int(resp.Msg.GetConcurrency())
	}
	return description{
		source: searchsource.Description{
			Name:          resp.Msg.GetName(),
			SourceRecipe:  resp.Msg.GetRecipe(),
			Width:         int(resp.Msg.GetWidth()),
			ChunkTokens:   int(max(resp.Msg.GetChunkTokens(), 0)),
			MinTextLength: int(max(resp.Msg.GetMinTextLength(), 0)),
			Concurrency:   concurrency,
		},
		floorHint: max(resp.Msg.GetFloorHint(), 0),
	}, nil
}

// readFailure is the failure of a read of the description. A plugin whose settings do not allow a source
// answers failed_precondition with a sentence a person reads, and that sentence is the failure.
func readFailure(pluginID string, err error) error {
	var failure *connect.Error
	if errors.As(err, &failure) && failure.Code() == connect.CodeFailedPrecondition {
		return errors.New(failure.Message())
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("the search source %s did not answer", pluginID)
	}
	return transport.ReadRPCError("describe search source "+pluginID, err)
}

// Describe returns the description of the search source, read again from the plugin when it answers.
func (b *Source) Describe() searchsource.Description {
	ctx, cancel := context.WithTimeout(context.Background(), transport.ModelRefreshTimeout)
	defer cancel()
	if client, call, err := b.service.Open(ctx); err == nil {
		read, readErr := readDescription(ctx, client)
		if errors.Join(readErr, call.Close()) == nil {
			b.mu.Lock()
			b.current = read
			b.mu.Unlock()
			return read.source
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current.source
}

// GetFloorHint returns the starting floor the plugin reported on the last read, and zero where it reported none.
func (b *Source) GetFloorHint(_ context.Context) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current.floorHint
}

// IndexTexts hands the texts to the plugin and decodes the vectors it answers.
func (b *Source) IndexTexts(ctx context.Context, place searchsource.Place, texts []searchsource.Text) (vectors [][]float32, err error) {
	if len(texts) == 0 {
		return nil, nil
	}
	indexed := make([]*v1.IndexedText, len(texts))
	for i, one := range texts {
		indexed[i] = &v1.IndexedText{ChunkId: string(one.ChunkID), Path: one.Path, VersionId: one.VersionID, Text: one.Text}
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.IndexTexts(ctx, connect.NewRequest(&v1.IndexTextsRequest{Place: writePlace(place), Texts: indexed}))
	if err != nil {
		return nil, transport.ReadRPCError("index texts rpc", err)
	}
	return readVectors(resp.Msg.GetVectors()), nil
}

// SearchPassages asks the plugin for the chunks of the place that answer the query.
func (b *Source) SearchPassages(ctx context.Context, place searchsource.Place, q searchsource.Query) (matches []searchsource.Match, err error) {
	kinds := make([]string, len(q.Kinds))
	for i, kind := range q.Kinds {
		kinds[i] = string(kind)
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.SearchPassages(ctx, connect.NewRequest(&v1.SearchSourceServiceSearchPassagesRequest{
		Place: writePlace(place), Query: q.Text, Limit: int32(min(max(q.Limit, 0), 1<<20)), Kinds: kinds, PathPrefix: q.PathPrefix, Floor: q.Floor,
	}))
	if err != nil {
		return nil, transport.ReadRPCError("search passages rpc", err)
	}
	matches = make([]searchsource.Match, len(resp.Msg.GetMatches()))
	for i, one := range resp.Msg.GetMatches() {
		matches[i] = searchsource.Match{ChunkID: domain.ChunkID(one.GetChunkId()), Score: one.GetScore()}
	}
	return matches, nil
}

// DescribeCoverage asks the plugin what a person reads about how far it has indexed the vault of the place.
func (b *Source) DescribeCoverage(ctx context.Context, place searchsource.Place, counts searchsource.Counts) (text string, err error) {
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.DescribeCoverage(ctx, connect.NewRequest(&v1.DescribeCoverageRequest{
		Place: writePlace(place), ChunkCount: counts.Chunks, IndexedCount: counts.Indexed,
	}))
	if err != nil {
		return "", transport.ReadRPCError("describe coverage rpc", err)
	}
	return resp.Msg.GetText(), nil
}

// EmbedQueries asks the plugin for the vectors of the queries.
func (b *Source) EmbedQueries(ctx context.Context, queries []string) (vectors [][]float32, err error) {
	if len(queries) == 0 {
		return nil, nil
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.EmbedQueries(ctx, connect.NewRequest(&v1.EmbedQueriesRequest{Queries: queries}))
	if err != nil {
		return nil, transport.ReadRPCError("embed queries rpc", err)
	}
	return readVectors(resp.Msg.GetVectors()), nil
}

func writePlace(place searchsource.Place) *v1.SearchPlace {
	return &v1.SearchPlace{VaultId: string(place.VaultID), Recipe: place.Recipe}
}

func readVectors(protoVectors []*v1.FloatVector) [][]float32 {
	vectors := make([][]float32, len(protoVectors))
	for i, pv := range protoVectors {
		vectors[i] = pv.GetValues()
	}
	return vectors
}

// Close releases the active reference count for the search source plugin in the host supervisor.
func (b *Source) Close() error {
	b.stopWatch()
	return b.lease.Close()
}

// OnDescriptionChange registers fn to run when the plugin reports a description unlike the last one
// announced: another recipe, width, shortest text, chunk size or concurrency. The plugin is read after it has been
// given its configuration, so a setting of another plugin never runs fn.
func (b *Source) OnDescriptionChange(fn func()) {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	b.listeners = append(b.listeners, fn)
}

// OnDescriptionReadError registers fn to receive the outcome of a read of the description that follows a
// configuration of the plugin: the error of a read that fails, and nil for a read that succeeds. After a
// failed read the description and the listeners stay as they were, and the next read that succeeds
// announces the change.
func (b *Source) OnDescriptionReadError(fn func(error)) {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	b.onReadError = fn
}

func (b *Source) reportReadError(err error) {
	if b.onReadError != nil {
		b.onReadError(err)
	}
}

func (b *Source) announceDescriptionChange() {
	b.announceMu.Lock()
	defer b.announceMu.Unlock()
	for _, fn := range b.readDescriptionChange() {
		fn()
	}
}

func (b *Source) readDescriptionChange() []func() {
	b.changeMu.Lock()
	defer b.changeMu.Unlock()
	// This read runs in the background. A source served through a connection may need the plugin of that
	// connection started first, so the read has the same limit as any other read of the description.
	ctx, cancel := context.WithTimeout(context.Background(), transport.ModelInfoTimeout)
	defer cancel()
	client, call, err := b.service.Open(ctx)
	if err != nil {
		b.reportReadError(fmt.Errorf("open search source %s: %w", b.pluginID, err))
		return nil
	}
	read, readErr := readDescription(ctx, client)
	if readErr != nil {
		b.reportReadError(errors.Join(readFailure(b.pluginID, readErr), call.Close()))
		return nil
	}
	if err := call.Close(); err != nil {
		b.reportReadError(err)
		return nil
	}
	b.mu.Lock()
	b.current = read
	b.mu.Unlock()
	b.reportReadError(nil)
	if read.source == b.announced {
		return nil
	}
	b.announced = read.source
	return slices.Clone(b.listeners)
}
