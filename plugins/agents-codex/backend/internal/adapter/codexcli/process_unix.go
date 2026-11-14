//go:build unix

package codexcli

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

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return translateKillError(syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL))
}

func translateKillError(err error) error {
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
