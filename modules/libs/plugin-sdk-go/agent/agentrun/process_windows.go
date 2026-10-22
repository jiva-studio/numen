//go:build windows

package agentrun

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// detach is a no-op: the tree of a child is ended by kill.
func detach(*exec.Cmd) {}

// kill ends the child and everything it started, which taskkill finds through the tree of the
// child.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	ended := exec.Command(taskkill(), "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	if ended == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	return ended
}

// folder.
func taskkill() string {
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
}
