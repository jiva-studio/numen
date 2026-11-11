//go:build !unix && !windows

package codexcli

import "os/exec"

func detach(*exec.Cmd) {}

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
