package service

import (
	"os/exec"
	"time"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is cancelled.
func supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}
