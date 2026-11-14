//go:build linux

package process

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
)

func TestNewCommandOnLinux(t *testing.T) {
	cmd := NewCommand(t.Context(), "sleep", 41235)
	if want := []string{"sleep", "--port=41235"}; !slices.Equal(cmd.Args, want) {
		t.Errorf("the command line is %v, want %v", cmd.Args, want)
	}
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
