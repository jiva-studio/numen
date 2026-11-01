//go:build unix

package service

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = new(syscall.SysProcAttr)
	}
	cmd.SysProcAttr.Setpgid = true
}

// kill ends the process group of the command. A group with no running process is reported as
// os.ErrProcessDone, which Wait does not report in place of the exit. macOS refuses with EPERM to
// signal a group whose processes have all exited but are not yet reaped, and every system refuses
// with ESRCH a group that is gone.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
