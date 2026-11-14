//go:build linux

package process

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
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
		wiretest.Kill(t, cmd)
		wiretest.ReapProcess(t, cmd.Process)
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

func TestHideInheritedFilesKeepsPipeOutOfChild(t *testing.T) {
	var ends [2]int
	if err := syscall.Pipe2(ends[:], 0); err != nil {
		t.Fatalf("pipe2: %v", err)
	}
	defer func() {
		if err := errors.Join(syscall.Close(ends[0]), syscall.Close(ends[1])); err != nil {
			t.Errorf("close pipe: %v", err)
		}
	}()

	if err := HideInheritedFiles(); err != nil {
		t.Fatalf("hide inherited files: %v", err)
	}

	out, err := exec.CommandContext(t.Context(), "ls", "/proc/self/fd").Output()
	if err != nil {
		t.Fatalf("list descriptors of the child: %v", err)
	}
	for line := range strings.FieldsSeq(string(out)) {
		if n, convErr := strconv.Atoi(line); convErr == nil && (n == ends[0] || n == ends[1]) {
			t.Errorf("the child inherited descriptor %d of the pipe", n)
		}
	}
}

func TestAPluginKeepsRunningAfterTheThreadThatRequestedItsStartExits(t *testing.T) {
	exitRead, exitWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := exitRead.Close(); err != nil {
			t.Error(err)
		}
	}()
	cmd := newLifetimePlugin(t.Context(), roleIdlePlugin, exitWrite)
	started := make(chan error, 1)
	go func() {
		// The goroutine returns with its thread locked, and Go ends that thread.
		runtime.LockOSThread()
		started <- StartCommand(cmd)
	}()
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	defer func() {
		wiretest.Kill(t, cmd)
		wiretest.ReapProcess(t, cmd.Process)
	}()
	if err := exitWrite.Close(); err != nil {
		t.Fatal(err)
	}
	if isClosedWithin(t, exitRead, time.Second) {
		t.Fatal("the plugin exited when the thread of the goroutine that started it ended")
	}
}
