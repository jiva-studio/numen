package container_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Reading a vault is work, and work is one entry in the list of operations. Each
// pass removes its entry when it ends, so a fully read vault reports no operations.
func TestAVaultReportsNoOperationsOnceItIsRead(t *testing.T) {
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	cfg := container.NewTestInstallation(container.Config{
		Paths: container.Paths{
			IndexPath:    filepath.Join(t.TempDir(), "index.db"),
			RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		},
	}, nil)
	if _, err := cfg.EnsureVault(cfg.Resources, root, "alpha", false); err != nil {
		t.Fatal(err)
	}
	installation, err := cfg.OpenWindow(cfg.Execution, cfg.Preferences, cfg.Endpoints, cfg.Resources, t.Context(), container.WindowRequest{Named: "", Out: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	server := httptest.NewServer(installation.NewHandler(http.NotFoundHandler(), numenv1connect.WindowServiceName))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	client := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
	tasks, err := client.WatchTasks(ctx, connect.NewRequest(&v1.WatchTasksRequest{WindowId: "editor"}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, tasks)
	for tasks.Receive() {
		if len(tasks.Msg().GetTasks()) == 0 {
			return
		}
	}
	t.Fatalf("task stream ended before the work finished: %v", tasks.Err())
}
