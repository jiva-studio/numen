//go:build linux

package plugin

import (
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinuxProcessLimitsRLIMIT_ASNotSet(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	limits := ProcessLimits{
		CPULimitSeconds:  10,
		MemoryLimitBytes: 1024 * 1024 * 1024,
	}
	if err := applyProcessLimits(cmd.Process.Pid, limits); err != nil {
		t.Fatalf("applyProcessLimits failed: %v", err)
	}

	var rlim unix.Rlimit
	if err := unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_AS, nil, &rlim); err != nil {
		t.Fatalf("get RLIMIT_AS: %v", err)
	}
	if rlim.Cur != unix.RLIM_INFINITY {
		t.Fatalf("expected RLIMIT_AS to remain infinity, got %d", rlim.Cur)
	}
}

func TestLinuxReadProcessRSS(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	rss, err := readProcessRSS(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("readProcessRSS failed: %v", err)
	}
	if rss == 0 {
		t.Fatal("expected non-zero RSS for running process")
	}
}

func TestLinuxProcessLimitsProcessAliveAfterMemoryLimit(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	limits := ProcessLimits{
		MemoryLimitBytes: 1024 * 1024 * 1024,
	}
	if err := applyProcessLimits(cmd.Process.Pid, limits); err != nil {
		t.Fatalf("applyProcessLimits failed: %v", err)
	}

	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("process exited unexpectedly after applyProcessLimits: %v", err)
	}
}

// A process started under the host's default limits has no RLIMIT_CPU, and one
// started under a CPU limit has exactly that.
func TestLinuxADefaultProcessHasNoCPULimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits ProcessLimits
		want   uint64
	}{
		{"the defaults", DefaultProcessLimits(), unix.RLIM_INFINITY},
		{"a limit of ten seconds", ProcessLimits{CPULimitSeconds: 10}, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("sleep", "5")
			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start mock process: %v", err)
			}
			defer func() {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
			}()
			if err := applyProcessLimits(cmd.Process.Pid, tc.limits); err != nil {
				t.Fatalf("applyProcessLimits failed: %v", err)
			}
			var rlim unix.Rlimit
			if err := unix.Prlimit(cmd.Process.Pid, unix.RLIMIT_CPU, nil, &rlim); err != nil {
				t.Fatalf("get RLIMIT_CPU: %v", err)
			}
			if rlim.Cur != tc.want {
				t.Errorf("RLIMIT_CPU is %d, want %d", rlim.Cur, tc.want)
			}
		})
	}
}
