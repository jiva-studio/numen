//go:build linux

package process

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func TestConfigureSysProcAttrOnLinux(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	ConfigureSysProcAttr(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("expected non-nil SysProcAttr on linux")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Error("expected Setpgid to be true on linux")
	}
	if cmd.SysProcAttr.Pdeathsig != syscall.SIGTERM {
		t.Errorf("expected Pdeathsig SIGTERM, got %v", cmd.SysProcAttr.Pdeathsig)
	}
}

func TestLinuxProcessLimitsRLIMIT_ASNotSet(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mock process: %v", err)
	}
	defer func() {
		testsupport.Kill(t, cmd)
		plugintest.ReapProcess(t, cmd.Process)
	}()

	limits := Limits{CPULimitSeconds: 10}
	if err := ApplyLimits(cmd.Process.Pid, limits); err != nil {
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

// A process started under the host's default limits has no RLIMIT_CPU, and one
// started under a CPU limit has exactly that.
func TestLinuxADefaultProcessHasNoCPULimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits Limits
		want   uint64
	}{
		{"the defaults", DefaultLimits(), unix.RLIM_INFINITY},
		{"a limit of ten seconds", Limits{CPULimitSeconds: 10}, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "sleep", "5")
			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start mock process: %v", err)
			}
			defer func() {
				if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Errorf("kill the mock process: %v", err)
				}
				if _, err := cmd.Process.Wait(); err != nil {
					t.Errorf("wait for the mock process: %v", err)
				}
			}()
			if err := ApplyLimits(cmd.Process.Pid, tc.limits); err != nil {
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
