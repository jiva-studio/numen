//go:build !unix && !windows

package process

import "os/exec"

// detach is a no-op where processes are not grouped.
func detach(*exec.Cmd) {}

// kill ends the child. Where processes are not grouped, what the child started outlives it.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
