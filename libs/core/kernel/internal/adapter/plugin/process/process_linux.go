//go:build linux

package process

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func ConfigureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

func ApplyLimits(pid int, limits Limits) error {
	if limits.CPULimitSeconds > 0 {
		if err := setRlimit(pid, unix.RLIMIT_CPU, limits.CPULimitSeconds); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
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

// HideInheritedFiles marks every open file descriptor above stderr close-on-exec,
// so a process started next inherits none of them. The window's asset server
// creates the pipe of each response without that flag: a plugin started while a
// response is open would keep the pipe's write end, and the page would not see
// the end of that response until the plugin exits.
func HideInheritedFiles() error {
	err := unix.CloseRange(3, ^uint(0), unix.CLOSE_RANGE_CLOEXEC)
	if errors.Is(err, unix.ENOSYS) {
		// An older kernel lacks close_range; those descriptors stay inherited.
		return nil
	}
	if err != nil {
		return fmt.Errorf("close_range: %w", err)
	}
	return nil
}
