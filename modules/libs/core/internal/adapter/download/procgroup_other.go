//go:build !unix && !windows

package download

import "os/exec"

// groupOf bounds how long a cancelled run keeps its caller waiting.
func groupOf(running *exec.Cmd) {
	running.WaitDelay = stopGrace
}
