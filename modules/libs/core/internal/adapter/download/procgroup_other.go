//go:build !unix && !windows

package download

import "os/exec"

// setProcessGroup bounds how long a cancelled run keeps its caller waiting.
func setProcessGroup(running *exec.Cmd) {
	running.WaitDelay = stopGrace
}
