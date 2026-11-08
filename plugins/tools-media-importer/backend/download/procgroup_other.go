//go:build !unix && !windows

package download

import "os/exec"

func setProcessGroup(running *exec.Cmd) {
	running.WaitDelay = stopGrace
}
