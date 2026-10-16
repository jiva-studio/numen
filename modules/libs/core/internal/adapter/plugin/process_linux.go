//go:build linux

package plugin

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func configureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

func applyProcessLimits(pid int, limits ProcessLimits) error {
	if limits.CPULimitSeconds > 0 {
		if err := setRlimit(pid, unix.RLIMIT_CPU, limits.CPULimitSeconds); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
		}
	}
	if limits.MemoryLimitBytes > 0 {
		if err := setRlimit(pid, unix.RLIMIT_AS, limits.MemoryLimitBytes); err != nil {
			return fmt.Errorf("set memory limit: %w", err)
		}
	}
	return nil
}

func setRlimit(pid int, resource int, limit uint64) error {
	var old unix.Rlimit
	if err := unix.Prlimit(pid, resource, nil, &old); err != nil {
		return err
	}
	newLim := old
	if limit < old.Max || old.Max == unix.RLIM_INFINITY {
		newLim.Cur = limit
		if limit < old.Max {
			newLim.Max = limit
		}
	} else {
		newLim.Cur = old.Max
	}
	return unix.Prlimit(pid, resource, &newLim, nil)
}
