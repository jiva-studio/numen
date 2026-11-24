//go:build windows

package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func detach(*exec.Cmd) {}

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	ended := exec.Command(getTaskkillPath(), "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	if ended == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	return ended
}

func getTaskkillPath() string {
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
}
