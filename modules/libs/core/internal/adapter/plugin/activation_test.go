package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func newDiscoveredHost(t *testing.T) *Host {
	t.Helper()
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

// activateOutsideBackoff calls EnsureActive until the answer is no longer a backoff refusal.
func activateOutsideBackoff(t *testing.T, host *Host) error {
	t.Helper()
	var err error
	waitUntil(t, "the backoff to end", func() bool {
		_, err = host.EnsureActive(t.Context(), "md.numen.ocr")
		return !errors.Is(err, errActivationBackoff)
	})
	return err
}

func TestAProcessThatNeverPrintsReadyIsKilledAtTheActivationDeadline(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := DefaultActivationPolicy()
	policy.Timeout = 50 * time.Millisecond
	host.SetActivationPolicy(policy)

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	t.Cleanup(func() { killProcess(t, cmd.Process) })
	silent, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close pipe: %v", err)
		}
	})
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, silent, nil
	})

	_, err := host.EnsureActive(context.Background(), "md.numen.ocr")
	if !errors.Is(err, errHandshakeTimeout) {
		t.Fatalf("expected a handshake timeout, got %v", err)
	}

	waitUntil(t, "the silent process to be killed", func() bool {
		return isProcessGone(cmd.Process)
	})
	if state := host.supervisors["md.numen.ocr"].GetState(); state != StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestACrashingPluginIsNotStartedAgainWithinItsBackoff(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := DefaultActivationPolicy()
	policy.BackoffBase = time.Hour
	policy.BackoffMax = time.Hour
	host.SetActivationPolicy(policy)

	launches := 0
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("exec format error")
	})

	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err == nil {
		t.Fatal("expected the first activation to fail")
	}
	_, err := host.EnsureActive(t.Context(), "md.numen.ocr")
	if !errors.Is(err, errActivationBackoff) {
		t.Fatalf("expected a backoff error, got %v", err)
	}
	if launches != 1 {
		t.Fatalf("expected one launch, got %d", launches)
	}
}

func TestAPluginIsReportedFailedAfterTheFailureLimit(t *testing.T) {
	host := newDiscoveredHost(t)
	policy := DefaultActivationPolicy()
	policy.BackoffBase = time.Millisecond
	policy.BackoffMax = time.Millisecond
	policy.MaxFailures = 3
	host.SetActivationPolicy(policy)

	var notified []string
	host.SetOnPluginFailed(func(id string, _ error) { notified = append(notified, id) })
	launches := 0
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("exec format error")
	})

	for range policy.MaxFailures {
		if err := activateOutsideBackoff(t, host); err == nil {
			t.Fatal("expected the activation to fail")
		}
	}
	err := activateOutsideBackoff(t, host)
	if !errors.Is(err, errPluginFailed) {
		t.Fatalf("expected the plugin to be failed, got %v", err)
	}
	if launches != policy.MaxFailures {
		t.Fatalf("expected %d launches, got %d", policy.MaxFailures, launches)
	}
	if len(notified) != 1 || notified[0] != "md.numen.ocr" {
		t.Fatalf("expected one failure notification, got %v", notified)
	}
	status, ok := host.GetPluginStatus("md.numen.ocr")
	if !ok || !status.Failed || status.FailureReason == "" {
		t.Fatalf("expected a failed status with a reason, got %+v", status)
	}

	if err := host.SetPluginEnabled("md.numen.ocr", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	status, ok = host.GetPluginStatus("md.numen.ocr")
	if !ok || status.Failed {
		t.Fatal("enabling the plugin should clear the failure")
	}
}

func TestTheBackoffDoublesUpToItsLimit(t *testing.T) {
	policy := ActivationPolicy{Timeout: 0, BackoffBase: time.Second, BackoffMax: 5 * time.Second, MaxFailures: 0}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second}
	for i, w := range want {
		if got := policy.getBackoff(i + 1); got != w {
			t.Fatalf("failure %d: expected %v, got %v", i+1, w, got)
		}
	}
}

func TestACanceledCallerDoesNotCountAsAFailedActivation(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("canceled")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := host.EnsureActive(ctx, "md.numen.ocr"); err == nil {
		t.Fatal("expected the activation to fail")
	}
	if err := host.supervisors["md.numen.ocr"].CheckActivationAllowed(time.Now()); err != nil {
		t.Fatalf("a canceled caller put the plugin into backoff: %v", err)
	}
}
