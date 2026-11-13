package sourcebridge_test

import (
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// stuckModelServer answers the model once, and then holds every request until the test ends, as a
// plugin whose model is downloading under a lock.
type stuckModelServer struct {
	numenv1connect.UnimplementedSearchSourceServiceHandler
	isStuck atomic.Bool
	release chan struct{}
}

func (s *stuckModelServer) DescribeSearchSource(
	ctx context.Context, _ *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	if s.isStuck.Load() {
		select {
		case <-s.release:
		case <-ctx.Done():
		}
	}
	return connect.NewResponse(&v1.DescribeSearchSourceResponse{Name: "stuck-model", Recipe: "stuck|8", Width: 8}), nil
}

func TestTheModelReadDoesNotWaitForAStuckPlugin(t *testing.T) {
	stuck := &stuckModelServer{release: make(chan struct{})}
	defer close(stuck.release)
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(stuck))
	server := httptest.NewServer(mux)
	defer server.Close()

	tmp := t.TempDir()
	dir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "md.numen.embeddings", "name": "Vector Embeddings", "version": "1.0.0", "capabilities": ["search.source"],
		"targets": ["desktop.editor"], "services": ["numen.v1.SearchSourceService"], "backend": {"entrypoint": "bin/embeddings"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
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

	bridge, err := sourcebridge.Open(t.Context(), pluginhost.Join(host), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, bridge)

	stuck.isStuck.Store(true)
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	reply := make(chan searchsource.Description, 1)
	go func() { reply <- bridge.Describe() }()
	var model searchsource.Description
	select {
	case model = <-reply:
	case <-bound.Done():
		t.Fatal("the model read waited more than 5s for a plugin that does not answer")
	}
	if model.Name != "stuck-model" {
		t.Fatalf("the read gave %q, not the model read before", model.Name)
	}
}
