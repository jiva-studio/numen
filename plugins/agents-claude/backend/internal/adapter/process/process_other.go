//go:build !unix && !windows

package process

import "os/exec"

func detach(*exec.Cmd) {}

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
