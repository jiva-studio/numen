package embedderbridge

import (
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// stuckModelServer answers the model once, and then holds every request until the test ends, as a
// plugin whose model is downloading under a lock.
type stuckModelServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
	isStuck atomic.Bool
	release chan struct{}
}

func (s *stuckModelServer) GetEmbeddingModel(
	ctx context.Context, _ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	if s.isStuck.Load() {
		select {
		case <-s.release:
		case <-ctx.Done():
		}
	}
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{Name: "stuck-model", Dimensions: 8, MaxTokens: 64, Pooling: "mean"}), nil
}

func TestTheModelReadDoesNotWaitForAStuckPlugin(t *testing.T) {
	stuck := &stuckModelServer{release: make(chan struct{})}
	defer close(stuck.release)
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(stuck))
	server := httptest.NewServer(mux)
	defer server.Close()

	tmp := t.TempDir()
	dir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "md.numen.embeddings", "name": "Vector Embeddings", "version": "1.0.0", "capabilities": ["embedding"],
		"targets": ["desktop.editor"], "services": ["numen.v1.EmbeddingService"], "backend": {"entrypoint": "bin/embeddings"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		wiretest.Kill(t, cmd)
		wiretest.ReapProcess(t, cmd.Process)
	}()
	port := wiretest.GetPort(t, server)
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", port))), nil
	})

	bridge, err := Open(t.Context(), pluginhost.Join(host))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, bridge)

	stuck.isStuck.Store(true)
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	reply := make(chan embedding.Model, 1)
	go func() { reply <- bridge.GetModel() }()
	var model embedding.Model
	select {
	case model = <-reply:
	case <-bound.Done():
		t.Fatal("the model read waited more than 5s for a plugin that does not answer")
	}
	if model.Name != "stuck-model" {
		t.Fatalf("the read gave %q, not the model read before", model.Name)
	}
}
