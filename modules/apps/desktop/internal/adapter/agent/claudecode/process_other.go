//go:build !unix

package claudecode

import "os/exec"

// detach is where a system that groups processes says so. This one does not.
func detach(*exec.Cmd) {}

// kill ends the child. What the child started outlives it here.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
