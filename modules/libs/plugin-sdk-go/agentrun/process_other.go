//go:build !unix

package agentrun

import "os/exec"

// detach is a no-op where processes are not grouped.
func detach(*exec.Cmd) {}

// kill ends the child; what the child started outlives it here.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
