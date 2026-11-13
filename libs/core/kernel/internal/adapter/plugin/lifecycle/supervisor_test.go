package lifecycle

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestSupervisorLifecycleAndIdleTracking(t *testing.T) {
	sup := New(nil)
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}

	err := sup.Attach(t.Context(), nil, testsupport.GetEpoch())
	if err == nil {
		t.Fatal("expected error attaching nil process, got nil")
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}

	now := testsupport.GetEpoch()
	if err := sup.Attach(t.Context(), cmd.Process, now); err != nil {
		t.Fatalf("failed to attach process: %v", err)
	}
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive, got %v", sup.GetState())
	}

	sup.Acquire(useLease, now)
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive after acquire, got %v", sup.GetState())
	}

	sup.Release(useLease, now)
	if sup.GetState() != StateIdle {
		t.Fatalf("expected StateIdle after release, got %v", sup.GetState())
	}

	threshold := 50 * time.Millisecond
	if sup.BeginDeactivation(threshold, now) {
		t.Fatal("BeginDeactivation took a plugin that was just released")
	}

	sup.Acquire(useLease, now.Add(70*time.Millisecond))
	if sup.GetState() != StateActive {
		t.Fatalf("expected StateActive after re-acquire, got %v", sup.GetState())
	}
	if sup.BeginDeactivation(threshold, now.Add(200*time.Millisecond)) {
		t.Fatal("BeginDeactivation took a plugin that has a consumer")
	}

	wiretest.Kill(t, cmd)
	wiretest.ReapProcess(t, cmd.Process)
}

func TestSupervisorTeardownNotRunning(t *testing.T) {
	sup := New(nil)
	err := sup.Teardown(t.Context(), 50*time.Millisecond, nil)
	if err == nil {
		t.Fatal("expected error tearing down non-running supervisor, got nil")
	}
}

func TestSupervisorTeardownGraceful(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	sup := New(nil)
	if err := sup.Attach(t.Context(), cmd.Process, testsupport.GetEpoch()); err != nil {
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
	sup := New(nil)
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(useLease, testsupport.GetEpoch())

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
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	sup := New(nil)
	if err := sup.Attach(t.Context(), cmd.Process, testsupport.GetEpoch()); err != nil {
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
	sup := New(nil)
	sup.Permissions.Set([]string{manifest.PermissionVaultRead, "network"})

	if !sup.Permissions.Has(manifest.PermissionVaultRead) {
		t.Errorf("expected permission %s to be present", manifest.PermissionVaultRead)
	}
	if !sup.Permissions.Has(manifest.PermissionLinkOpen) {
		t.Errorf("expected permission %s to be normalized and present", manifest.PermissionLinkOpen)
	}
	if !sup.Permissions.Has("network") {
		t.Errorf("expected alias network to return true")
	}
	if err := sup.Permissions.Check(manifest.PermissionLinkOpen); err != nil {
		t.Errorf("expected CheckPermission for %s to pass: %v", manifest.PermissionLinkOpen, err)
	}
	if sup.Permissions.Has(manifest.PermissionVaultWrite) {
		t.Errorf("expected undeclared permission %s to be false", manifest.PermissionVaultWrite)
	}
}

func TestSupervisorProcessCrashTriggersOnExit(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	sup := New(nil)
	var stateInOnExit atomic.Value
	sup.SetOnExit(func() {
		stateInOnExit.Store(sup.GetState())
	})

	if err := sup.Attach(t.Context(), cmd.Process, testsupport.GetEpoch()); err != nil {
		t.Fatalf("attach process: %v", err)
	}
	exited := sup.GetExitSignal()

	wiretest.Kill(t, cmd)

	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the exit of the crashed process")
	}

	if state := stateInOnExit.Load(); state != StateActive {
		t.Errorf("onExit ran in state %v, want it to run before the supervisor leaves StateActive", state)
	}
	if sup.GetState() != StateDormant {
		t.Errorf("expected StateDormant after crash, got %v", sup.GetState())
	}
}

func TestSupervisorActivationLifecycle(t *testing.T) {
	sup := New(nil)
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}

	closedCh := sup.ActivatingChan()
	select {
	case <-closedCh:
	default:
		t.Fatal("expected closed channel when not activating")
	}

	ch1 := sup.SetActivating(plugin.Holder{})
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

	ch3 := sup.SetActivating(plugin.Holder{})
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
	sup := New(nil)
	ch := sup.SetActivating(plugin.Holder{})

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

func TestAcquire_RejectsDeactivatingPlugin(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer func() {
		wiretest.KillProcess(t, cmd.Process)
	}()
	sup := New(nil)
	now := testsupport.GetEpoch()
	if err := sup.Attach(t.Context(), cmd.Process, now); err != nil {
		t.Fatalf("failed to attach: %v", err)
	}
	sup.Release(useLease, now)

	if !sup.BeginDeactivation(0, now) {
		t.Fatal("expected an idle plugin to begin deactivating")
	}
	if sup.GetState() != StateDeactivating {
		t.Fatalf("expected StateDeactivating, got %v", sup.GetState())
	}
	if sup.Acquire(useLease, now) {
		t.Fatal("Acquire held a deactivating plugin")
	}
}

func TestADeactivationDoesNotBeginOnAHeldPlugin(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer func() {
		wiretest.KillProcess(t, cmd.Process)
	}()
	sup := New(nil)
	now := testsupport.GetEpoch()
	if err := sup.Attach(t.Context(), cmd.Process, now); err != nil {
		t.Fatalf("failed to attach: %v", err)
	}
	sup.Acquire(useLease, now)

	if sup.BeginDeactivation(0, now) {
		t.Fatal("a held plugin began deactivating")
	}
}

func TestSupervisorPermissionsAndPathValidation(t *testing.T) {
	vaultDir := t.TempDir()
	docFile := filepath.Join(vaultDir, "doc.pdf")
	if err := os.WriteFile(docFile, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("permission denied without vault:read", func(t *testing.T) {
		sup := New(nil)
		sup.Permissions.Set([]string{"link:open"})

		if err := sup.Permissions.Check("vault:read"); !errors.Is(err, plugin.ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got %v", err)
		}

		if _, err := sup.Permissions.ValidatePath(vaultDir, "doc.pdf"); !errors.Is(err, plugin.ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got %v", err)
		}
	})

	t.Run("permission granted with vault:read", func(t *testing.T) {
		sup := New(nil)
		sup.Permissions.Set([]string{"vault:read", "vault:write"})

		if err := sup.Permissions.Check("vault:read"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := sup.Permissions.ValidatePath(vaultDir, "doc.pdf")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != docFile {
			t.Fatalf("got %s, want %s", got, docFile)
		}
	})

	t.Run("traversal blocked even with vault:read", func(t *testing.T) {
		sup := New(nil)
		sup.Permissions.Set([]string{"vault:read"})

		_, err := sup.Permissions.ValidatePath(vaultDir, "../secret.txt")
		if !errors.Is(err, plugin.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})
}

func TestGetUsageReturnsTheLastUseAndTheLeases(t *testing.T) {
	sup := New(nil)
	if usage, isRunning := sup.GetUsage(); isRunning {
		t.Fatalf("a dormant plugin has the usage %+v", usage)
	}
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(useLease, start)
	sup.Acquire(useLease, start.Add(time.Second))

	usage, _ := sup.GetUsage()
	if want := (Usage{
		LastActiveTime: start.Add(time.Second), LeaseCount: 2, IsInProcess: true, IsUsed: true,
		Holders: []plugin.Holder{useLease.Holder, useLease.Holder},
	}); !reflect.DeepEqual(usage, want) {
		t.Errorf("held: got %+v, want %+v", usage, want)
	}

	sup.Release(useLease, start.Add(2*time.Second))
	sup.Release(useLease, start.Add(3*time.Second))
	usage, _ = sup.GetUsage()
	if len(usage.Holders) != 0 {
		t.Errorf("holders after the release %+v, want none", usage.Holders)
	}
	usage.Holders = nil
	if want := (Usage{LastActiveTime: start.Add(3 * time.Second), IsInProcess: true, IsIdle: true, IsUsed: true}); !reflect.DeepEqual(usage, want) {
		t.Errorf("released: got %+v, want %+v", usage, want)
	}
}

func TestAPluginStartedForAQuestionStopsWhenTheQuestionEnds(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(questionLease, start)

	if sup.BeginDeactivation(testIdleLimit, start.Add(time.Second)) {
		t.Fatal("a plugin was stopped while a question held it")
	}
	sup.Release(questionLease, start.Add(time.Second))

	if usage, _ := sup.GetUsage(); usage.IsUsed || !usage.LastActiveTime.Equal(start) {
		t.Errorf("after a question: got %+v, want an unused plugin last active at its start", usage)
	}
	if !sup.BeginDeactivation(testIdleLimit, start.Add(time.Second)) {
		t.Error("a plugin only asked a question was not stopped at once")
	}
}

func TestAQuestionKeepsTheIdleCountdownOfAUsedPlugin(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(useLease, start)
	sup.Release(useLease, start.Add(time.Minute))

	sup.Acquire(questionLease, start.Add(2*time.Minute))
	sup.Release(questionLease, start.Add(3*time.Minute))

	if usage, _ := sup.GetUsage(); !usage.IsUsed || !usage.LastActiveTime.Equal(start.Add(time.Minute)) {
		t.Errorf("got %+v, want the last use one minute after the start", usage)
	}
	if sup.BeginDeactivation(testIdleLimit, start.Add(time.Minute+testIdleLimit-time.Second)) {
		t.Error("a used plugin was stopped before its idle limit")
	}
	if !sup.BeginDeactivation(testIdleLimit, start.Add(time.Minute+testIdleLimit)) {
		t.Error("the question moved the idle countdown of a used plugin")
	}
}

func TestAUseDuringAQuestionMakesThePluginUsed(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(questionLease, start)

	sup.Acquire(useLease, start.Add(time.Second))
	sup.Release(useLease, start.Add(2*time.Second))
	sup.Release(questionLease, start.Add(3*time.Second))

	if usage, _ := sup.GetUsage(); !usage.IsUsed || !usage.LastActiveTime.Equal(start.Add(2*time.Second)) {
		t.Errorf("got %+v, want a used plugin last active when the use ended", usage)
	}
	if sup.BeginDeactivation(testIdleLimit, start.Add(3*time.Second)) {
		t.Error("a used plugin was stopped before its idle limit")
	}
	if !sup.BeginDeactivation(testIdleLimit, start.Add(2*time.Second+testIdleLimit)) {
		t.Error("a used plugin was not stopped at its idle limit")
	}
}

func TestAStartedProcessCountsAsUnusedAgain(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	defer wiretest.KillProcess(t, cmd.Process)
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(useLease, start)
	sup.Release(useLease, start)
	if err := sup.Teardown(t.Context(), 0, nil); err != nil {
		t.Fatal(err)
	}

	if err := sup.Attach(t.Context(), cmd.Process, start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if usage, _ := sup.GetUsage(); usage.IsUsed {
		t.Errorf("got %+v, want a process that is not used yet", usage)
	}
}

// testIdleLimit is the idle limit the tests of the countdown stop a used plugin after.
const testIdleLimit = 3 * time.Minute
