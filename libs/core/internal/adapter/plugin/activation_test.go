package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func newDiscoveredHost(t *testing.T) *Host {
	t.Helper()
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

func TestEnsureActive_KillsProcessAtActivationTimeout(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = 50 * time.Millisecond
	host.SetActivationPolicy(policy)

	cmd := plugintest.NewIdleCommand(t.Context())
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	t.Cleanup(func() { plugintest.KillProcess(t, cmd.Process) })
	silent, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close pipe: %v", err)
		}
	})
	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, silent, nil
	})

	_, err := host.EnsureActive(t.Context(), "org.example.sample")
	if !errors.Is(err, transport.ErrHandshakeTimeout) {
		t.Fatalf("expected a handshake timeout, got %v", err)
	}

	if err := cmd.Process.Signal(os.Kill); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("expected the silent process to be killed and reaped, got %v", err)
	}
	if state := host.supervisors["org.example.sample"].GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func failLaunching(launches *int) Launcher {
	return func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		*launches++
		return nil, nil, errors.New("exec format error")
	}
}

func TestAPluginWhoseStartFailedIsStartedAgainOnItsNextUseAtOnce(t *testing.T) {
	host := newDiscoveredHost(t)
	launches := 0
	host.SetLauncher(failLaunching(&launches))

	for range 3 {
		if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err == nil {
			t.Fatal("expected the start to fail")
		}
	}

	if launches != 3 {
		t.Fatalf("expected a launch for each use, got %d", launches)
	}
}

func TestAFailedStartIsReportedEachTimeAndShownWithItsReason(t *testing.T) {
	host := newDiscoveredHost(t)
	var notified []string
	host.SetOnPluginFailed(func(failure pluginport.PluginFailure) { notified = append(notified, failure.PluginID) })
	launches := 0
	host.SetLauncher(failLaunching(&launches))

	for range 2 {
		if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err == nil {
			t.Fatal("expected the start to fail")
		}
	}

	if len(notified) != 2 {
		t.Fatalf("expected a failure notification for each start, got %v", notified)
	}
	status, ok := host.GetPluginStatus("org.example.sample")
	if !ok || !status.Failed || status.FailureReason == "" {
		t.Fatalf("expected a failed status with a reason, got %+v", status)
	}
}

func TestACanceledCallerDoesNotRecordAFailedStart(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("canceled")
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := host.EnsureActive(ctx, "org.example.sample"); err == nil {
		t.Fatal("expected the activation to fail")
	}
	if host.supervisors["org.example.sample"].IsActivationFailed() {
		t.Fatal("a canceled caller recorded a failed start")
	}
}
