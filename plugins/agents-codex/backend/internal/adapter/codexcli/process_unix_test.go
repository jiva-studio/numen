//go:build unix

package codexcli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSuperviseAndWaitForExit(t *testing.T) {
	unstarted := exec.CommandContext(t.Context(), "true")
	Supervise(unstarted)
	if unstarted.WaitDelay != 2*time.Second || unstarted.Cancel == nil {
		t.Fatalf("Supervise WaitDelay = %v, Cancel = %v", unstarted.WaitDelay, unstarted.Cancel != nil)
	}
	if unstarted.SysProcAttr == nil || !unstarted.SysProcAttr.Setpgid {
		t.Errorf("expected Setpgid = true, got %+v", unstarted.SysProcAttr)
	}
	if err := unstarted.Cancel(); err != nil {
		t.Errorf("Cancel on unstarted command = %v, want nil", err)
	}

	dead := exec.CommandContext(t.Context(), "true")
	Supervise(dead)
	dead.Process = &os.Process{Pid: 99999999}
	if err := dead.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("Cancel on non-existent PID = %v, want os.ErrProcessDone", err)
	}

	if got := translateKillError(nil); got != nil {
		t.Errorf("translateKillError(nil) = %v, want nil", got)
	}
	for _, sigErr := range []error{syscall.EPERM, syscall.ESRCH} {
		if got := translateKillError(sigErr); !errors.Is(got, os.ErrProcessDone) {
			t.Errorf("translateKillError(%v) = %v, want os.ErrProcessDone", sigErr, got)
		}
	}
	if got := translateKillError(syscall.EINVAL); !errors.Is(got, syscall.EINVAL) {
		t.Errorf("translateKillError(EINVAL) = %v, want EINVAL", got)
	}

	running, stop := context.WithCancel(t.Context())
	defer stop()
	script := writeScript(t, "sleeper", "sleep 30\n")
	cmd := exec.CommandContext(running, script)
	Supervise(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := WaitForExit(cmd, stop); err == nil {
		t.Error("expected non-nil error from WaitForExit when command was killed after grace period")
	}
}
