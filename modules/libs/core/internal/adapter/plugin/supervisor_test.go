package plugin

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestSupervisorLifecycleAndIdleTracking(t *testing.T) {
	sup := NewSupervisor()
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}

	err := sup.Attach(nil, time.Now())
	if err == nil {
		t.Fatal("expected error attaching nil process, got nil")
	}

	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}

	now := time.Now()
	if err := sup.Attach(cmd.Process, now); err != nil {
		t.Fatalf("failed to attach process: %v", err)
	}
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive, got %v", sup.GetState())
	}

	sup.Acquire(now)
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive after acquire, got %v", sup.GetState())
	}

	sup.Release(now)
	if sup.GetState() != StateIdle {
		t.Fatalf("expected StateIdle after release, got %v", sup.GetState())
	}

	threshold := 50 * time.Millisecond
	if sup.IsIdle(threshold, now) {
		t.Fatal("expected IsIdle to be false immediately")
	}
	if !sup.IsIdle(threshold, now.Add(60*time.Millisecond)) {
		t.Fatal("expected IsIdle to be true after threshold expired")
	}

	sup.Acquire(now.Add(70 * time.Millisecond))
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive after re-acquire, got %v", sup.GetState())
	}
	if sup.IsIdle(threshold, now.Add(100*time.Millisecond)) {
		t.Fatal("expected IsIdle to be false when ref count > 0")
	}

	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func TestSupervisorTeardownNotRunning(t *testing.T) {
	sup := NewSupervisor()
	err := sup.Teardown(t.Context(), 50*time.Millisecond, nil)
	if err == nil {
		t.Fatal("expected error tearing down non-running supervisor, got nil")
	}
}

func TestSupervisorTeardownGraceful(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	sup := NewSupervisor()
	if err := sup.Attach(cmd.Process, time.Now()); err != nil {
		t.Fatalf("failed to attach: %v", err)
	}

	rpcCalled := false
	err := sup.Teardown(t.Context(), 200*time.Millisecond, func(_ context.Context) error {
		rpcCalled = true
		return cmd.Process.Kill()
	})
	if err != nil {
		t.Fatalf("expected successful graceful teardown, got: %v", err)
	}
	if !rpcCalled {
		t.Fatal("expected rpcShutdown callback to be invoked")
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestSupervisorTeardownInProcessCallsRPC(t *testing.T) {
	sup := NewSupervisor()
	sup.SetActivating()
	sup.ActivateInProcess(time.Now())

	rpcCalled := false
	err := sup.Teardown(t.Context(), 50*time.Millisecond, func(_ context.Context) error {
		rpcCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected successful in-process teardown, got: %v", err)
	}
	if !rpcCalled {
		t.Fatal("expected rpcShutdown to be invoked for in-process supervisor")
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestSupervisorTeardownForceKill(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	sup := NewSupervisor()
	if err := sup.Attach(cmd.Process, time.Now()); err != nil {
		t.Fatalf("failed to attach: %v", err)
	}

	err := sup.Teardown(t.Context(), 20*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("expected successful force kill teardown, got: %v", err)
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestSupervisorPermissionNormalization(t *testing.T) {
	sup := NewSupervisor()
	sup.SetPermissions([]string{PermissionVaultRead, "network"})

	if !sup.HasPermission(PermissionVaultRead) {
		t.Errorf("expected permission %s to be present", PermissionVaultRead)
	}
	if !sup.HasPermission(PermissionNetworkOutbound) {
		t.Errorf("expected permission %s to be normalized and present", PermissionNetworkOutbound)
	}
	if !sup.HasPermission("network") {
		t.Errorf("expected alias network to return true")
	}
	if err := sup.CheckPermission(PermissionNetworkOutbound); err != nil {
		t.Errorf("expected CheckPermission for %s to pass: %v", PermissionNetworkOutbound, err)
	}
	if sup.HasPermission(PermissionVaultWrite) {
		t.Errorf("expected undeclared permission %s to be false", PermissionVaultWrite)
	}
}

func TestSupervisorProcessCrashTriggersOnExit(t *testing.T) {
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	sup := NewSupervisor()
	onExitCh := make(chan struct{})
	sup.SetOnExit(func() {
		close(onExitCh)
	})

	if err := sup.Attach(cmd.Process, time.Now()); err != nil {
		t.Fatalf("attach process: %v", err)
	}

	_ = cmd.Process.Kill()

	select {
	case <-onExitCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for onExit callback on crashed process")
	}

	if sup.GetState() != StateDormant {
		t.Errorf("expected StateDormant after crash, got %v", sup.GetState())
	}
}

func TestSupervisorActivationLifecycle(t *testing.T) {
	sup := NewSupervisor()
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}

	closedCh := sup.ActivatingChan()
	select {
	case <-closedCh:
	default:
		t.Fatal("expected closed channel when not activating")
	}

	ch1 := sup.SetActivating()
	if sup.GetState() != StateActivating {
		t.Fatalf("expected StateActivating, got %v", sup.GetState())
	}
	ch2 := sup.ActivatingChan()

	select {
	case <-ch1:
		t.Fatal("channel should not be closed while activating")
	default:
	}

	expectedErr := errors.New("activation failed")
	sup.SetDormant(expectedErr)
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
	if !errors.Is(sup.ActivationError(), expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, sup.ActivationError())
	}

	select {
	case <-ch1:
	default:
		t.Fatal("expected ch1 to be closed after SetDormant")
	}
	select {
	case <-ch2:
	default:
		t.Fatal("expected ch2 to be closed after SetDormant")
	}

	ch3 := sup.SetActivating()
	if sup.ActivationError() != nil {
		t.Fatal("expected ActivationError to be cleared on new activation")
	}
	sup.FinishActivating()
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive, got %v", sup.GetState())
	}
	select {
	case <-ch3:
	default:
		t.Fatal("expected ch3 to be closed after FinishActivating")
	}
}

func TestSupervisorTeardownWhileActivating(t *testing.T) {
	sup := NewSupervisor()
	ch := sup.SetActivating()

	err := sup.Teardown(t.Context(), 50*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("expected teardown during activation to succeed, got %v", err)
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant after teardown, got %v", sup.GetState())
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected activating channel to be closed on teardown")
	}
}

func TestSupervisorMemoryTrackingAndEviction(t *testing.T) {
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	sup := NewSupervisor()
	sup.SetMemoryLimit(100 * 1024 * 1024)
	if sup.GetMemoryLimit() != 100*1024*1024 {
		t.Fatalf("got memory limit %d, want %d", sup.GetMemoryLimit(), 100*1024*1024)
	}

	if err := sup.Attach(cmd.Process, time.Now()); err != nil {
		t.Fatalf("failed to attach process: %v", err)
	}

	rss, err := sup.GetResidentMemory()
	if err != nil {
		t.Fatalf("GetResidentMemory failed: %v", err)
	}
	if rss > 100*1024*1024 {
		t.Fatalf("expected initial RSS to be below 100MB, got %d", rss)
	}
	if sup.IsMemoryExceeded() {
		t.Fatal("expected IsMemoryExceeded to be false")
	}

	// Override RSS reader to simulate memory limit exceeded
	sup.SetRSSReader(func(_ int) (uint64, error) {
		return 200 * 1024 * 1024, nil
	})

	if !sup.IsMemoryExceeded() {
		t.Fatal("expected IsMemoryExceeded to be true when simulated RSS exceeds limit")
	}

	rpcCalled := make(chan struct{})
	sup.SetRPCShutdown(func(_ context.Context) error {
		close(rpcCalled)
		_ = cmd.Process.Kill()
		return nil
	})
	sup.SetMemoryGracePeriod(100 * time.Millisecond)
	sup.SetMemoryMonitorInterval(10 * time.Millisecond)

	// Trigger memory monitor by re-setting limit or wait for monitor tick
	sup.SetMemoryLimit(50 * 1024 * 1024)

	select {
	case <-rpcCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected graceful RPC shutdown on memory limit exceeded")
	}

	deadline := time.Now().Add(2 * time.Second)
	for sup.GetState() != StateDormant && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant after eviction, got %v", sup.GetState())
	}
}
