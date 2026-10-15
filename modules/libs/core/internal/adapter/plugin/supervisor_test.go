package plugin

import (
	"context"
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

	err := sup.Teardown(t.Context(), 200*time.Millisecond, func(_ context.Context) error {
		return cmd.Process.Kill()
	})
	if err != nil {
		t.Fatalf("expected successful graceful teardown, got: %v", err)
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
