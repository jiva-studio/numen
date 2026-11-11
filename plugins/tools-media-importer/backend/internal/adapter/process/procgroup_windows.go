//go:build windows

package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

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

func getTaskkillPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "taskkill.exe")
}
