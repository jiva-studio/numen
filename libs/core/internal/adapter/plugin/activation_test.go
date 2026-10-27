package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func newDiscoveredHost(t *testing.T) *Host {
	t.Helper()
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

func activateOutsideBackoff(t *testing.T, host *Host) error {
	t.Helper()
	var err error
	plugintest.WaitUntil(t, "the backoff to end", func() bool {
		_, err = host.EnsureActive(t.Context(), "org.example.sample")
		return !errors.Is(err, lifecycle.ErrActivationBackoff)
	})
	return err
}

func TestAProcessThatNeverPrintsReadyIsKilledAtTheActivationDeadline(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = 50 * time.Millisecond
	host.SetActivationPolicy(policy)

	cmd := plugintest.NewIdleCommand()
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
	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
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

func TestACrashingPluginIsNotStartedAgainWithinItsBackoff(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.BackoffBase = time.Hour
	policy.BackoffMax = time.Hour
	host.SetActivationPolicy(policy)

	launches := 0
	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("exec format error")
	})

	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err == nil {
		t.Fatal("expected the first activation to fail")
	}
	_, err := host.EnsureActive(t.Context(), "org.example.sample")
	if !errors.Is(err, lifecycle.ErrActivationBackoff) {
		t.Fatalf("expected a backoff error, got %v", err)
	}
	if launches != 1 {
		t.Fatalf("expected one launch, got %d", launches)
	}
}

func TestAPluginIsReportedFailedAfterTheFailureLimit(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.BackoffBase = time.Millisecond
	policy.BackoffMax = time.Millisecond
	policy.MaxFailures = 3
	host.SetActivationPolicy(policy)

	var notified []string
	host.SetOnPluginFailed(func(failure pluginport.PluginFailure) { notified = append(notified, failure.PluginID) })
	launches := 0
	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("exec format error")
	})

	for range policy.MaxFailures {
		if err := activateOutsideBackoff(t, host); err == nil {
			t.Fatal("expected the activation to fail")
		}
	}
	err := activateOutsideBackoff(t, host)
	if !errors.Is(err, lifecycle.ErrPluginFailed) {
		t.Fatalf("expected the plugin to be failed, got %v", err)
	}
	if launches != policy.MaxFailures {
		t.Fatalf("expected %d launches, got %d", policy.MaxFailures, launches)
	}
	if len(notified) != 1 || notified[0] != "org.example.sample" {
		t.Fatalf("expected one failure notification, got %v", notified)
	}
	status, ok := host.GetPluginStatus("org.example.sample")
	if !ok || !status.Failed || status.FailureReason == "" {
		t.Fatalf("expected a failed status with a reason, got %+v", status)
	}

	if err := host.SetPluginEnabled(t.Context(), "org.example.sample", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	status, ok = host.GetPluginStatus("org.example.sample")
	if !ok || status.Failed {
		t.Fatal("enabling the plugin should clear the failure")
	}
}

func TestTheBackoffDoublesUpToItsLimit(t *testing.T) {
	policy := lifecycle.ActivationPolicy{Timeout: 0, BackoffBase: time.Second, BackoffMax: 5 * time.Second, MaxFailures: 0}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second}
	for i, w := range want {
		if got := policy.GetBackoff(i + 1); got != w {
			t.Fatalf("failure %d: expected %v, got %v", i+1, w, got)
		}
	}
}

func TestACanceledCallerDoesNotCountAsAFailedActivation(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("canceled")
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := host.EnsureActive(ctx, "org.example.sample"); err == nil {
		t.Fatal("expected the activation to fail")
	}
	if err := host.supervisors["org.example.sample"].CheckActivationAllowed(time.Now()); err != nil {
		t.Fatalf("a canceled caller put the plugin into backoff: %v", err)
	}
}

func TestConcurrentCallersDoNotRestartAFailedActivationWithinItsBackoff(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := lifecycle.DefaultActivationPolicy()
	policy.BackoffBase = time.Hour
	policy.BackoffMax = time.Hour
	host.SetActivationPolicy(policy)

	var launches atomic.Int32
	release := make(chan struct{})
	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launches.Add(1)
		<-release
		return nil, nil, errors.New("exec format error")
	})

	var callers sync.WaitGroup
	for range 8 {
		callers.Add(1)
		go func() {
			defer callers.Done()
			if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err == nil {
				t.Error("expected the activation to fail")
			}
		}()
	}
	plugintest.WaitUntil(t, "the first launch", func() bool { return launches.Load() >= 1 })
	close(release)
	callers.Wait()

	if got := launches.Load(); got != 1 {
		t.Fatalf("expected one launch within the backoff, got %d", got)
	}
}
