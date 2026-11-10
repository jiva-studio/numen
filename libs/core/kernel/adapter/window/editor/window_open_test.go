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
	"sync"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
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
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	})
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaults.NewAdd(cfg.VaultIdentity(), registry.Vaults, registry.Lookup, registry.Saver, testsupport.NewClock()).Execute(root, "alpha"); err != nil {
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
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(&mockEmbeddingServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()
	serverAddress, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the test server listens on %T, want *net.TCPAddr", server.Listener.Addr())
	}
	serverPort := serverAddress.Port

	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		testsupport.Kill(t, cmd)
		// The plugin host may have reaped the process already.
		if _, err := cmd.Process.Wait(); err != nil && !errors.Is(err, syscall.ECHILD) && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("wait: %v", err)
		}
	}()

	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := vaultConfig(t)
	cfg.PluginHost = host

	installation, err := cfg.OpenWindow(t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	if installation.Embedder == nil {
		t.Fatal("the vault opened with no embedder on hand")
	}
	if got := installation.Embedder.GetModel().Name; got != "asked-for" {
		t.Errorf("got %q", got)
	}
}

// An installation with no model opens, and reports that by having none.
func TestAVaultWithNoModelOpensAnyway(t *testing.T) {
	installation, err := vaultConfig(t).OpenWindow(t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	if installation.Embedder != nil {
		t.Errorf("got an embedder: %v", installation.Embedder.GetModel())
	}
}

func gateWalk(start editor.StartVaultFunc, release <-chan struct{}) editor.StartVaultFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool,
		notified func(paths, assets []string, reload bool),
		handleError windowdomain.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vaults.Refresh,
		unwatched error,
	) {
		walk, run, refresh, unwatched := start(ctx, v, rebuild, notified, handleError)
		read = func(ctx context.Context, during func()) (int, error) {
			select {
			case <-release:
				return walk(ctx, during)
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		return read, run, refresh, unwatched
	}
}

// Reading a vault is work, and work is one entry in the list of operations. A
// pass that reports itself elsewhere creates a second entry for the same work,
// with a separate count.
func TestAVaultReturnsWhatItIsDoingWhileItReadsItself(t *testing.T) {
	cfg := vaultConfig(t)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseWalk := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseWalk)

	ports, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	installation, err := cfg.OpenWindowWith(t.Context(), ports, db, "", os.Stderr, func(start editor.StartVaultFunc) editor.StartVaultFunc {
		return gateWalk(start, release)
	})
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	// Reported before the reading goroutine starts, so a window opening on a new
	// vault shows the walk from its start.
	list := installation.State.Window.Tasking.List()
	if len(list) != 1 {
		t.Errorf("a vault just opened is doing %+v, want the walk of its notes", list)
	}
	// Not requested by the user, so it is displayed only once it has lasted.
	for _, item := range list {
		if item.IsUserStarted {
			t.Errorf("%q says a person asked for it", item.Title)
		}
	}

	// Each pass removes its entry when it ends, so a fully read vault reports no
	// operations.
	releaseWalk()
	testsupport.WaitFor(t, func() bool { return len(installation.State.Window.Tasking.List()) == 0 })
}
