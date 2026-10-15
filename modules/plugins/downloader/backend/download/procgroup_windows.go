//go:build windows

package download

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setProcessGroup puts the run in a process group of its own and makes cancelling it
// end the whole process tree.
func setProcessGroup(running *exec.Cmd) {
	running.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	running.Cancel = func() error {
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(running.Process.Pid))
		if err := kill.Run(); err != nil {
			return running.Process.Kill()
		}
		return nil
	}
	running.WaitDelay = stopGrace
}
