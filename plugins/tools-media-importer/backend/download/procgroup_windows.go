//go:build windows

package download

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// setProcessGroup puts the run in a process group of its own and makes cancelling it
// end the whole process tree.
func setProcessGroup(running *exec.Cmd) {
	running.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	running.Cancel = func() error {
		kill := exec.Command(getTaskkillPath(), "/T", "/F", "/PID", strconv.Itoa(running.Process.Pid))
		if err := kill.Run(); err != nil {
			return running.Process.Kill()
		}
		return nil
	}
	running.WaitDelay = stopGrace
}

// getTaskkillPath returns the path of taskkill.exe in the Windows system folder.
func getTaskkillPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "taskkill.exe")
}
