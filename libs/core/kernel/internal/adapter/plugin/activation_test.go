package plugin_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"os"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newDiscoveredHost(t *testing.T) *container.PluginRoles {
	t.Helper()
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

func TestEnsureActive_KillsProcessAtActivationTimeout(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = 50 * time.Millisecond
	host.Processes.SetActivationPolicy(policy)

	cmd := wiretest.NewIdleCommand(t.Context())
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	t.Cleanup(func() { wiretest.KillProcess(t, cmd.Process) })
	silent, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close pipe: %v", err)
		}
	})
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, silent, nil
	})

	_, err := host.Activation.EnsureActive(t.Context(), "org.example.sample")
	if !errors.Is(err, transport.ErrHandshakeTimeout) {
		t.Fatalf("expected a handshake timeout, got %v", err)
	}

	if err := cmd.Process.Signal(os.Kill); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("expected the silent process to be killed and reaped, got %v", err)
	}
	if state := supervisorOf(host, "org.example.sample").GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func failLaunchingCounting(launches *int) supervision.Launcher {
	return func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		*launches++
		return nil, nil, errors.New("exec format error")
	}
}

func TestAPluginWhoseStartFailedIsStartedAgainOnItsNextUseAtOnce(t *testing.T) {
	host := newDiscoveredHost(t)
	launches := 0
	host.Processes.SetLauncher(failLaunchingCounting(&launches))

	for range 3 {
		if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err == nil {
			t.Fatal("expected the start to fail")
		}
	}

	if launches != 3 {
		t.Fatalf("expected a launch for each use, got %d", launches)
	}
}

func TestAFailedStartIsListedOnceAndShownWithItsReason(t *testing.T) {
	host := newDiscoveredHost(t)
	launches := 0
	host.Processes.SetLauncher(failLaunchingCounting(&launches))

	for range 2 {
		if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err == nil {
			t.Fatal("expected the start to fail")
		}
	}

	if items := host.Health.List(); len(items) != 1 || items[0].PluginID != "org.example.sample" || items[0].Key != plugindomain.HealthKeyStart {
		t.Fatalf("expected one start item for the plugin, got %+v", items)
	}
	status, ok := host.Registry.GetPluginStatus("org.example.sample")
	if !ok || !status.Failed || status.FailureReason == "" {
		t.Fatalf("expected a failed status with a reason, got %+v", status)
	}
}

func TestACanceledCallerDoesNotRecordAFailedStart(t *testing.T) {
	host := newDiscoveredHost(t)
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("canceled")
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := host.Activation.EnsureActive(ctx, "org.example.sample"); err == nil {
		t.Fatal("expected the activation to fail")
	}
	if supervisorOf(host, "org.example.sample").Failure.IsFailed() {
		t.Fatal("a canceled caller recorded a failed start")
	}
}
