package sourcebridge_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// The bridge makes query vectors apart from document vectors, which search relies on through the port.
var _ embeddingport.QueryEmbedder = (*embedderbridge.Embedder)(nil)

type mockEmbeddingServer struct {
	getModelCalls    atomic.Int32
	modelName        atomic.Value
	dimensions       atomic.Int32
	floorHint        atomic.Value
	minChunk         atomic.Int32
	chunkTokens      atomic.Int32
	handling         atomic.Value
	failRead         atomic.Bool
	failPrecondition atomic.Bool

	mu    sync.Mutex
	roles []v1.EmbedRole
}

func (s *mockEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	s.getModelCalls.Add(1)
	if s.failPrecondition.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no connection serves the operation"))
	}
	if s.failRead.Load() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the model is not available"))
	}
	name := "test-model"
	if v, ok := s.modelName.Load().(string); ok && v != "" {
		name = v
	}
	dims := int32(384)
	if d := s.dimensions.Load(); d > 0 {
		dims = d
	}
	hint := 0.0
	if v, ok := s.floorHint.Load().(float64); ok {
		hint = v
	}
	handling := ""
	if v, ok := s.handling.Load().(string); ok {
		handling = v
	}
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Handling:       handling,
		Name:           name,
		Dimensions:     dims,
		MaxTokens:      256,
		Pooling:        "mean",
		From:           "test-provider",
		Concurrency:    4,
		FloorHint:      hint,
		MinChunkLength: s.minChunk.Load(),
		ChunkTokens:    s.chunkTokens.Load(),
	}), nil
}

func (s *mockEmbeddingServer) EmbedBatch(
	_ context.Context,
	req *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
	s.mu.Lock()
	s.roles = append(s.roles, req.Msg.GetRole())
	s.mu.Unlock()
	vectors := make([]*v1.FloatVector, len(texts))
	for i := range texts {
		vectors[i] = &v1.FloatVector{
			Values: []float32{float32(i) * 0.1, 0.5},
		}
	}
	return connect.NewResponse(&v1.EmbedBatchResponse{
		Vectors: vectors,
	}), nil
}

func TestOpenEmbedderBridgeNoPlugin(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", t.TempDir())
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	bridge, err := embedderbridge.Open(t.Context(), pluginhost.Join(host))
	if err == nil {
		t.Fatal("expected error when no embedding plugin found, got nil")
	}
	if bridge != nil {
		t.Fatalf("expected nil bridge, got %v", bridge)
	}
}

func TestEmbedderBridge(t *testing.T) {
	mockServer := &mockEmbeddingServer{}
	mockServer.floorHint.Store(0.35)
	mockServer.minChunk.Store(40)
	mockServer.handling.Store("1a2b3c4d")
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(mockServer))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := wiretest.GetPort(t, server)

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifestText := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"services": ["numen.v1.EmbeddingService"], "backend": {"entrypoint": "bin/embeddings"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifestText), 0o600); err != nil {
		t.Fatal(err)
	}

	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := wiretest.StartSleepProcess(t)

	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := embedderbridge.Open(ctx, pluginhost.Join(host))
	if err != nil {
		t.Fatalf("OpenEmbedderBridge failed: %v", err)
	}
	defer testsupport.Close(t, bridge)

	model := bridge.GetModel()
	if model.Name != "test-model" {
		t.Errorf("model Name = %q, want test-model", model.Name)
	}
	if model.Dimensions != 384 {
		t.Errorf("model Dimensions = %d, want 384", model.Dimensions)
	}
	if model.MaxTokens != 256 {
		t.Errorf("model MaxTokens = %d, want 256", model.MaxTokens)
	}
	if model.Pooling != "mean" {
		t.Errorf("model Pooling = %q, want mean", model.Pooling)
	}
	if model.From != "test-provider" {
		t.Errorf("model From = %q, want test-provider", model.From)
	}

	if model.Handling != "1a2b3c4d" {
		t.Errorf("model Handling = %q, want 1a2b3c4d", model.Handling)
	}
	if model.MinChunkLength != 40 {
		t.Errorf("model MinChunkLength = %d, want 40", model.MinChunkLength)
	}
	if hint := bridge.GetFloorHint(ctx); hint != 0.35 {
		t.Errorf("floor hint = %v, want 0.35", hint)
	}
	recipe := model.Recipe()
	mockServer.floorHint.Store(0.42)
	mockServer.minChunk.Store(0)
	if refreshed := bridge.GetModel(); refreshed.Recipe() != recipe {
		t.Errorf("a new floor hint changed the recipe: %q, was %q", refreshed.Recipe(), recipe)
	}
	if refreshed := bridge.GetModel(); refreshed.MinChunkLength != 0 {
		t.Errorf("a new minimum chunk length reads as %d, want 0", refreshed.MinChunkLength)
	}
	if hint := bridge.GetFloorHint(ctx); hint != 0.42 {
		t.Errorf("floor hint after a read of the model = %v, want 0.42", hint)
	}
	mockServer.floorHint.Store(-1.0)
	bridge.GetModel()
	if hint := bridge.GetFloorHint(ctx); hint != 0 {
		t.Errorf("a negative floor hint reads as %v, want 0", hint)
	}
	mockServer.floorHint.Store(0.0)
	bridge.GetModel()
	if hint := bridge.GetFloorHint(ctx); hint != 0 {
		t.Errorf("no floor hint reads as %v, want 0", hint)
	}

	callsBefore := mockServer.getModelCalls.Load()
	for range 5 {
		if bridge.Concurrency() != 4 {
			t.Errorf("concurrency = %d, want 4", bridge.Concurrency())
		}
	}
	callsAfter := mockServer.getModelCalls.Load()
	if callsBefore != callsAfter {
		t.Errorf("Concurrency() made RPC calls: before=%d, after=%d", callsBefore, callsAfter)
	}

	vectors, err := bridge.Embed(ctx, []string{"first text", "second text"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != 2 || vectors[0][0] != 0.0 {
		t.Errorf("unexpected vector[0]: %v", vectors[0])
	}
	if len(vectors[1]) != 2 || vectors[1][0] != 0.1 {
		t.Errorf("unexpected vector[1]: %v", vectors[1])
	}

	if _, err := bridge.EmbedQueries(ctx, []string{"a question"}); err != nil {
		t.Fatalf("EmbedQueries failed: %v", err)
	}
	mockServer.mu.Lock()
	gotRoles := slices.Clone(mockServer.roles)
	mockServer.mu.Unlock()
	if want := []v1.EmbedRole{v1.EmbedRole_EMBED_ROLE_DOCUMENT, v1.EmbedRole_EMBED_ROLE_QUERY}; !slices.Equal(gotRoles, want) {
		t.Errorf("the plugin was asked for roles %v, want %v", gotRoles, want)
	}

	empty, err := bridge.Embed(ctx, nil)
	if err != nil || empty != nil {
		t.Errorf("expected nil vectors for empty input, got %v, err=%v", empty, err)
	}

	mockServer.handling.Store("ffffffff")
	if refreshed := bridge.GetModel(); refreshed.Recipe() == recipe {
		t.Errorf("a new handling left the recipe as it was: %q", refreshed.Recipe())
	}
	mockServer.handling.Store("1a2b3c4d")
	mockServer.modelName.Store("updated-model")
	mockServer.dimensions.Store(1024)
	currentModel := bridge.GetModel()
	if currentModel.Name != "updated-model" || currentModel.Dimensions != 1024 {
		t.Errorf("GetModel() after update = %+v, want updated-model@1024", currentModel)
	}
}
