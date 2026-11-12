//go:build !unix && !windows

package process

import "os/exec"

func setProcessGroup(running *exec.Cmd) {
	running.WaitDelay = stopGrace
}
