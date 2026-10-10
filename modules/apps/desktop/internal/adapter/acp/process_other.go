//go:build !unix

package acp

import "os/exec"

func detach(_ *exec.Cmd) {}

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
