package embedderbridge

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

type mockEmbeddingServer struct {
	getModelCalls atomic.Int32
}

func (s *mockEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	s.getModelCalls.Add(1)
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Name:        "test-model",
		Dimensions:  384,
		MaxTokens:   256,
		Pooling:     "mean",
		From:        "test-provider",
		Concurrency: 4,
	}), nil
}

func (s *mockEmbeddingServer) EmbedBatch(
	_ context.Context,
	req *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
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
	host := plugin.NewHost("desktop.editor", t.TempDir())
	_ = host.Discover()

	bridge, err := Open(t.Context(), host)
	if err == nil {
		t.Fatal("expected error when no embedding plugin found, got nil")
	}
	if bridge != nil {
		t.Fatalf("expected nil bridge, got %v", bridge)
	}
}

func TestEmbedderBridge(t *testing.T) {
	mockServer := &mockEmbeddingServer{}
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(mockServer))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"backend": {
			"type": "process",
			"entrypoint": "bin/embeddings",
			"serviceName": "numen.v1.EmbeddingService"
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := Open(ctx, host)
	if err != nil {
		t.Fatalf("OpenEmbedderBridge failed: %v", err)
	}
	defer bridge.Close()

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

	callsBefore := mockServer.getModelCalls.Load()
	for i := 0; i < 5; i++ {
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

	empty, err := bridge.Embed(ctx, nil)
	if err != nil || empty != nil {
		t.Errorf("expected nil vectors for empty input, got %v, err=%v", empty, err)
	}
}
