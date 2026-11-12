package editor_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type mockEmbeddingServer struct{}

func (s *mockEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Name:       "asked-for",
		Dimensions: 384,
	}), nil
}

func (s *mockEmbeddingServer) EmbedBatch(
	_ context.Context,
	_ *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	return connect.NewResponse(&v1.EmbedBatchResponse{}), nil
}

func vaultConfig(t *testing.T) container.Config {
	t.Helper()
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	if _, err := cfg.AddVault(root, "alpha"); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// The embedder configured for an installation serves search as well as index
// filling; this test checks both.
func TestEmbedder_ConfiguredEmbedderServesSearch(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestText := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/embeddings"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifestText), 0o600); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(&mockEmbeddingServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()
	serverAddress, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the test server listens on %T, want *net.TCPAddr", server.Listener.Addr())
	}
	serverPort := serverAddress.Port

	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		wiretest.Kill(t, cmd)
		// The plugin host may have reaped the process already.
		if _, err := cmd.Process.Wait(); err != nil && !errors.Is(err, syscall.ECHILD) && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("wait: %v", err)
		}
	}()

	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := vaultConfig(t)
	cfg.PluginHost = container.AdaptPluginHost(host)

	installation, _, err := openRegisteredWindow(cfg, t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	name, _ := installation.State.Indexing.Embedding.Describe()
	if name == "" {
		t.Fatal("the vault opened with no embedder on hand")
	}
	if got, _, _ := strings.Cut(name, "@"); got != "asked-for" {
		t.Errorf("got %q", got)
	}
}

// An installation with no model opens, and reports that by having none.
func TestAVaultWithNoModelOpensAnyway(t *testing.T) {
	installation, _, err := openRegisteredWindow(vaultConfig(t), t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	if name, _ := installation.State.Indexing.Embedding.Describe(); name != "" {
		t.Errorf("got an embedder: %v", name)
	}
}
