//go:build unix

package agentrun

import (
	"os/exec"
	"syscall"
)

// detach puts the child in a process group of its own, which is what ends it and everything it
// started in one act.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// kill ends the child and everything it started.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
