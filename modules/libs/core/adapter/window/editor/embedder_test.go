package editor_test

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
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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

// vault gives a folder an identity and registers it, and answers with the
// configuration that opens it.
func vault(t *testing.T, embedding container.EmbeddingConfig) container.Config {
	t.Helper()
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		Embedding:    embedding,
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaults.NewAdd(cfg.VaultIdentity(), registry, time.Now).Execute(root, "asked"); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// The embedder an installation configured reaches the search as well as the
// filling of the index, and this asks for it.
func TestTheEmbedderConfiguredIsTheOneOnHand(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(&mockEmbeddingServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()
	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ plugin.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := vault(t, container.EmbeddingConfig{})
	cfg.PluginHost = host

	installation, err := editor.Open(
		t.Context(), editor.NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installation.Close() })

	if installation.Embedder == nil {
		t.Fatal("the vault opened with no embedder on hand")
	}
	if got := installation.Embedder.GetModel().Name; got != "asked-for" {
		t.Errorf("got %q", got)
	}
}

// An installation with no model opens, and says so by having none.
func TestAVaultWithNoModelOpensAnyway(t *testing.T) {
	installation, err := editor.Open(
		t.Context(), editor.NewAssembly(t, vault(t, container.EmbeddingConfig{})), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installation.Close() })

	if installation.Embedder != nil {
		t.Errorf("got an embedder: %v", installation.Embedder.GetModel())
	}
}

// Reading a vault is work, and work is one entry in the list of what is being
// done. A pass that reports itself anywhere else draws a second card for the
// same work, with a count of its own that nobody can place.
func TestAVaultSaysWhatItIsDoingWhileItReadsItself(t *testing.T) {
	cfg := vault(t, container.EmbeddingConfig{})

	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installation.Close() })

	// Said before the reading goroutine starts, so a window that opens on a
	// fresh vault is shown the walk from its first moment.
	list := installation.State.Window.Tasking.List()
	if len(list) != 1 {
		t.Errorf("a vault just opened is doing %+v, want the walk of its notes", list)
	}
	// Nobody asked for it, so it is drawn once it has lasted.
	for _, at := range list {
		if at.IsUserStarted {
			t.Errorf("%q says a person asked for it", at.Title)
		}
	}

	// Each pass takes itself out when it ends, so a vault that has been read
	// says it is doing nothing.
	testsupport.WaitFor(t, func() bool { return len(installation.State.Window.Tasking.List()) == 0 })
}
